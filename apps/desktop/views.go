package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	desktopViewsVersion        = 3
	legacyDesktopViewsVersion  = 1
	legacyDesktopViewsVersion2 = 2

	maxViewsPerProfile  = 32
	maxWidgetsPerView   = 64
	maxViewLayoutDepth  = 32
	maxViewLayoutNodes  = 127
	layoutWeightEpsilon = 1e-6
)

type ViewDocument struct {
	Version int              `json:"version"`
	Views   []ViewDefinition `json:"views"`
}

type ViewDefinition struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Revision        uint64          `json:"revision"`
	Widgets         []ViewWidget    `json:"widgets"`
	LayoutRoot      *ViewLayoutNode `json:"layout_root,omitempty"`
	CreatedAtUnixMS int64           `json:"created_at_unix_ms"`
	UpdatedAtUnixMS int64           `json:"updated_at_unix_ms"`
}

type ViewLayoutNode struct {
	Kind     string           `json:"kind"`
	WidgetID string           `json:"widget_id,omitempty"`
	Axis     string           `json:"axis,omitempty"`
	Children []ViewLayoutNode `json:"children,omitempty"`
	Weights  []float64        `json:"weights,omitempty"`
}

type ViewWidget struct {
	ID           string          `json:"id"`
	OwnerNodeID  string          `json:"owner_node_id"`
	ResourceName string          `json:"resource_name"`
	Renderer     string          `json:"renderer"`
	Settings     json.RawMessage `json:"settings,omitempty"`
}

type legacyViewDocument struct {
	Version int                    `json:"version"`
	Views   []legacyViewDefinition `json:"views"`
}

type legacyViewDefinition struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Revision        uint64             `json:"revision"`
	Widgets         []legacyViewWidget `json:"widgets"`
	Layout          *legacyViewLayout  `json:"layout,omitempty"`
	CreatedAtUnixMS int64              `json:"created_at_unix_ms"`
	UpdatedAtUnixMS int64              `json:"updated_at_unix_ms"`
}

type legacyViewLayout struct {
	Direction  string  `json:"direction"`
	SplitRatio float64 `json:"split_ratio"`
}

type legacyViewWidget struct {
	ID           string          `json:"id"`
	OwnerNodeID  string          `json:"owner_node_id"`
	ResourceName string          `json:"resource_name"`
	Renderer     string          `json:"renderer"`
	X            int             `json:"x"`
	Y            int             `json:"y"`
	W            int             `json:"w"`
	H            int             `json:"h"`
	Settings     json.RawMessage `json:"settings,omitempty"`
}

type viewStore struct{ root string }

func newViewStore(root string) (*viewStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("view store root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve view store: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create view store: %w", err)
	}
	return &viewStore{root: abs}, nil
}

func (s *viewStore) load() (ViewDocument, error) {
	document, _, err := s.loadWithSourceVersion()
	return document, err
}

func (s *viewStore) loadWithSourceVersion() (ViewDocument, int, error) {
	data, err := os.ReadFile(s.path())
	if errors.Is(err, os.ErrNotExist) {
		return ViewDocument{Version: desktopViewsVersion, Views: []ViewDefinition{}}, desktopViewsVersion, nil
	}
	if err != nil {
		return ViewDocument{}, 0, fmt.Errorf("read views: %w", err)
	}
	document, sourceVersion, err := decodeViewDocument(data)
	if err != nil {
		return ViewDocument{}, 0, fmt.Errorf("view store is corrupt and was not overwritten: %w", err)
	}
	return document, sourceVersion, nil
}

func decodeViewDocument(data []byte) (ViewDocument, int, error) {
	var header struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return ViewDocument{}, 0, err
	}
	switch header.Version {
	case desktopViewsVersion:
		var document ViewDocument
		if err := decodeStrictJSON(data, &document); err != nil {
			return ViewDocument{}, 0, err
		}
		if err := validateViewDocument(document); err != nil {
			return ViewDocument{}, 0, err
		}
		return document, desktopViewsVersion, nil
	case legacyDesktopViewsVersion, legacyDesktopViewsVersion2:
		var legacy legacyViewDocument
		if err := decodeStrictJSON(data, &legacy); err != nil {
			return ViewDocument{}, 0, err
		}
		if err := validateLegacyViewDocument(legacy); err != nil {
			return ViewDocument{}, 0, err
		}
		document := migrateLegacyViewDocument(legacy)
		if err := validateViewDocument(document); err != nil {
			return ViewDocument{}, 0, fmt.Errorf("migrate views version %d: %w", legacy.Version, err)
		}
		return document, legacy.Version, nil
	default:
		return ViewDocument{}, 0, fmt.Errorf("views version must be %d, %d, or %d", legacyDesktopViewsVersion, legacyDesktopViewsVersion2, desktopViewsVersion)
	}
}

func (s *viewStore) saveView(next ViewDefinition) (ViewDefinition, error) {
	document, sourceVersion, err := s.loadWithSourceVersion()
	if err != nil {
		return ViewDefinition{}, err
	}
	if !profileIDPattern.MatchString(next.ID) {
		return ViewDefinition{}, errors.New("view id is invalid")
	}
	now := time.Now().UTC().UnixMilli()
	found := false
	for index := range document.Views {
		current := document.Views[index]
		if current.ID != next.ID {
			continue
		}
		if next.Revision != current.Revision {
			return ViewDefinition{}, fmt.Errorf("view revision conflict: current %d, got %d", current.Revision, next.Revision)
		}
		if current.Revision == ^uint64(0) {
			return ViewDefinition{}, errors.New("view revision exhausted")
		}
		next.Revision = current.Revision + 1
		next.CreatedAtUnixMS = current.CreatedAtUnixMS
		next.UpdatedAtUnixMS = now
		document.Views[index] = next
		found = true
		break
	}
	if !found {
		if next.Revision != 0 {
			return ViewDefinition{}, errors.New("new view revision must be zero")
		}
		if len(document.Views) >= maxViewsPerProfile {
			return ViewDefinition{}, fmt.Errorf("profile supports at most %d views", maxViewsPerProfile)
		}
		next.Revision = 1
		next.CreatedAtUnixMS = now
		next.UpdatedAtUnixMS = now
		document.Views = append(document.Views, next)
	}
	if err := validateView(next); err != nil {
		return ViewDefinition{}, err
	}
	sort.Slice(document.Views, func(i, j int) bool { return document.Views[i].ID < document.Views[j].ID })
	if err := s.save(document, sourceVersion); err != nil {
		return ViewDefinition{}, err
	}
	return next, nil
}

func (s *viewStore) deleteView(id string, revision uint64) error {
	document, sourceVersion, err := s.loadWithSourceVersion()
	if err != nil {
		return err
	}
	for index, current := range document.Views {
		if current.ID != id {
			continue
		}
		if current.Revision != revision {
			return fmt.Errorf("view revision conflict: current %d, got %d", current.Revision, revision)
		}
		document.Views = append(document.Views[:index], document.Views[index+1:]...)
		return s.save(document, sourceVersion)
	}
	return errors.New("view not found")
}

func (s *viewStore) save(document ViewDocument, sourceVersion int) error {
	document.Version = desktopViewsVersion
	if err := validateViewDocument(document); err != nil {
		return err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode views: %w", err)
	}
	if sourceVersion < desktopViewsVersion {
		if err := s.ensurePreV3Snapshot(); err != nil {
			return err
		}
	}
	return writeAtomicDesktop(s.root, s.path(), append(data, '\n'))
}

func (s *viewStore) ensurePreV3Snapshot() error {
	snapshotPath := s.preV3SnapshotPath()
	if snapshot, err := os.ReadFile(snapshotPath); err == nil {
		_, version, decodeErr := decodeViewDocument(snapshot)
		if decodeErr != nil || version >= desktopViewsVersion {
			return errors.New("existing pre-v3 view snapshot is invalid; refusing to overwrite it")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read pre-v3 view snapshot: %w", err)
	}

	data, err := os.ReadFile(s.path())
	if err != nil {
		return fmt.Errorf("read views for pre-v3 snapshot: %w", err)
	}
	_, version, err := decodeViewDocument(data)
	if err != nil {
		return fmt.Errorf("validate views for pre-v3 snapshot: %w", err)
	}
	if version >= desktopViewsVersion {
		return nil
	}

	file, err := os.OpenFile(snapshotPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return s.ensurePreV3Snapshot()
	}
	if err != nil {
		return fmt.Errorf("create pre-v3 view snapshot: %w", err)
	}
	removeOnFailure := true
	defer func() {
		_ = file.Close()
		if removeOnFailure {
			_ = os.Remove(snapshotPath)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write pre-v3 view snapshot: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync pre-v3 view snapshot: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close pre-v3 view snapshot: %w", err)
	}
	removeOnFailure = false
	return nil
}

func (s *viewStore) path() string { return filepath.Join(s.root, "views.json") }

func (s *viewStore) preV3SnapshotPath() string {
	return filepath.Join(s.root, "views.pre-v3.json")
}

func validateViewDocument(document ViewDocument) error {
	if document.Version != desktopViewsVersion {
		return fmt.Errorf("views version must be %d", desktopViewsVersion)
	}
	if len(document.Views) > maxViewsPerProfile {
		return fmt.Errorf("view count exceeds %d", maxViewsPerProfile)
	}
	seen := make(map[string]struct{}, len(document.Views))
	for index, view := range document.Views {
		if err := validateView(view); err != nil {
			return fmt.Errorf("views[%d]: %w", index, err)
		}
		if _, exists := seen[view.ID]; exists {
			return fmt.Errorf("duplicate view %q", view.ID)
		}
		seen[view.ID] = struct{}{}
	}
	return nil
}

func validateView(view ViewDefinition) error {
	if !profileIDPattern.MatchString(view.ID) {
		return errors.New("view id is invalid")
	}
	if name := strings.TrimSpace(view.Name); name == "" || len(name) > 80 {
		return errors.New("view name must contain between 1 and 80 bytes")
	}
	if view.Revision == 0 {
		return errors.New("persisted view revision must be non-zero")
	}
	if len(view.Widgets) > maxWidgetsPerView {
		return fmt.Errorf("view supports at most %d widgets", maxWidgetsPerView)
	}
	widgetIDs := make(map[string]struct{}, len(view.Widgets))
	for index, widget := range view.Widgets {
		if err := validateViewWidget(widget, index); err != nil {
			return err
		}
		if _, exists := widgetIDs[widget.ID]; exists {
			return fmt.Errorf("duplicate widget %q", widget.ID)
		}
		widgetIDs[widget.ID] = struct{}{}
	}
	if len(view.Widgets) == 0 {
		if view.LayoutRoot != nil {
			return errors.New("empty view must not have layout_root")
		}
		return nil
	}
	if view.LayoutRoot == nil {
		return errors.New("non-empty view requires layout_root")
	}
	leaves := make(map[string]struct{}, len(widgetIDs))
	nodeCount := 0
	if err := validateLayoutNode(*view.LayoutRoot, 1, "", &nodeCount, leaves); err != nil {
		return fmt.Errorf("layout_root: %w", err)
	}
	if nodeCount > maxViewLayoutNodes {
		return fmt.Errorf("layout node count exceeds %d", maxViewLayoutNodes)
	}
	if len(leaves) != len(widgetIDs) {
		return errors.New("layout leaves must match widgets exactly")
	}
	for id := range widgetIDs {
		if _, exists := leaves[id]; !exists {
			return fmt.Errorf("layout is missing widget %q", id)
		}
	}
	return nil
}

func validateViewWidget(widget ViewWidget, index int) error {
	if !profileIDPattern.MatchString(widget.ID) {
		return fmt.Errorf("widgets[%d] id is invalid", index)
	}
	if _, err := parsePositiveInt64(widget.OwnerNodeID, "widget owner_node_id"); err != nil {
		return err
	}
	if widget.ResourceName == "" || len(widget.ResourceName) > 255 || strings.TrimSpace(widget.Renderer) == "" {
		return fmt.Errorf("widgets[%d] resource or renderer is invalid", index)
	}
	if len(widget.Settings) > 64<<10 || (len(widget.Settings) > 0 && !json.Valid(widget.Settings)) {
		return fmt.Errorf("widgets[%d] settings are invalid", index)
	}
	return nil
}

func validateLayoutNode(node ViewLayoutNode, depth int, parentAxis string, nodeCount *int, leaves map[string]struct{}) error {
	(*nodeCount)++
	if *nodeCount > maxViewLayoutNodes {
		return fmt.Errorf("layout node count exceeds %d", maxViewLayoutNodes)
	}
	if depth > maxViewLayoutDepth {
		return fmt.Errorf("layout depth exceeds %d", maxViewLayoutDepth)
	}
	switch node.Kind {
	case "leaf":
		if !profileIDPattern.MatchString(node.WidgetID) {
			return errors.New("leaf widget_id is invalid")
		}
		if node.Axis != "" || node.Children != nil || node.Weights != nil {
			return errors.New("leaf contains split-only fields")
		}
		if _, exists := leaves[node.WidgetID]; exists {
			return fmt.Errorf("duplicate layout leaf %q", node.WidgetID)
		}
		leaves[node.WidgetID] = struct{}{}
		return nil
	case "split":
		if node.WidgetID != "" {
			return errors.New("split contains leaf-only widget_id")
		}
		if node.Axis != "horizontal" && node.Axis != "vertical" {
			return errors.New("split axis must be horizontal or vertical")
		}
		if parentAxis == node.Axis {
			return fmt.Errorf("nested %s splits must be flattened", node.Axis)
		}
		if len(node.Children) < 2 || len(node.Children) > maxWidgetsPerView {
			return fmt.Errorf("split must contain between 2 and %d children", maxWidgetsPerView)
		}
		if len(node.Weights) != len(node.Children) {
			return errors.New("split weights must match children")
		}
		sum := 0.0
		for index, weight := range node.Weights {
			if math.IsNaN(weight) || math.IsInf(weight, 0) || weight <= 0 {
				return fmt.Errorf("weights[%d] must be finite and positive", index)
			}
			sum += weight
		}
		if math.Abs(sum-1) > layoutWeightEpsilon {
			return fmt.Errorf("split weights must sum to 1, got %.12g", sum)
		}
		for index, child := range node.Children {
			if err := validateLayoutNode(child, depth+1, node.Axis, nodeCount, leaves); err != nil {
				return fmt.Errorf("children[%d]: %w", index, err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown layout node kind %q", node.Kind)
	}
}

func validateLegacyViewDocument(document legacyViewDocument) error {
	if document.Version != legacyDesktopViewsVersion && document.Version != legacyDesktopViewsVersion2 {
		return errors.New("legacy views version is invalid")
	}
	if len(document.Views) > maxViewsPerProfile {
		return fmt.Errorf("view count exceeds %d", maxViewsPerProfile)
	}
	seenViews := make(map[string]struct{}, len(document.Views))
	for viewIndex, view := range document.Views {
		if !profileIDPattern.MatchString(view.ID) {
			return fmt.Errorf("views[%d] id is invalid", viewIndex)
		}
		if name := strings.TrimSpace(view.Name); name == "" || len(name) > 80 {
			return fmt.Errorf("views[%d] name must contain between 1 and 80 bytes", viewIndex)
		}
		if view.Revision == 0 {
			return fmt.Errorf("views[%d] persisted revision must be non-zero", viewIndex)
		}
		if _, exists := seenViews[view.ID]; exists {
			return fmt.Errorf("duplicate view %q", view.ID)
		}
		seenViews[view.ID] = struct{}{}
		if len(view.Widgets) > maxWidgetsPerView {
			return fmt.Errorf("views[%d] supports at most %d widgets", viewIndex, maxWidgetsPerView)
		}
		if document.Version == legacyDesktopViewsVersion && view.Layout != nil {
			return fmt.Errorf("views[%d] uses layout with legacy version", viewIndex)
		}
		if view.Layout != nil {
			if view.Layout.Direction != "horizontal" && view.Layout.Direction != "vertical" {
				return fmt.Errorf("views[%d] layout direction must be horizontal or vertical", viewIndex)
			}
			if math.IsNaN(view.Layout.SplitRatio) || math.IsInf(view.Layout.SplitRatio, 0) || view.Layout.SplitRatio < 0.2 || view.Layout.SplitRatio > 0.8 {
				return fmt.Errorf("views[%d] layout split ratio must be between 0.2 and 0.8", viewIndex)
			}
		}
		seenWidgets := make(map[string]struct{}, len(view.Widgets))
		for widgetIndex, widget := range view.Widgets {
			current := ViewWidget{
				ID: widget.ID, OwnerNodeID: widget.OwnerNodeID, ResourceName: widget.ResourceName,
				Renderer: widget.Renderer, Settings: widget.Settings,
			}
			if err := validateViewWidget(current, widgetIndex); err != nil {
				return fmt.Errorf("views[%d]: %w", viewIndex, err)
			}
			if _, exists := seenWidgets[widget.ID]; exists {
				return fmt.Errorf("views[%d] duplicate widget %q", viewIndex, widget.ID)
			}
			seenWidgets[widget.ID] = struct{}{}
			if widget.X < 0 || widget.X >= 12 || widget.Y < 0 || widget.W < 1 || widget.W > 12 || widget.H < 1 || widget.H > 24 || widget.X+widget.W > 12 {
				return fmt.Errorf("views[%d] widgets[%d] layout is outside the 12-column grid", viewIndex, widgetIndex)
			}
		}
	}
	return nil
}

func migrateLegacyViewDocument(legacy legacyViewDocument) ViewDocument {
	document := ViewDocument{Version: desktopViewsVersion, Views: make([]ViewDefinition, len(legacy.Views))}
	for index, view := range legacy.Views {
		widgets := make([]ViewWidget, len(view.Widgets))
		for widgetIndex, widget := range view.Widgets {
			widgets[widgetIndex] = ViewWidget{
				ID: widget.ID, OwnerNodeID: widget.OwnerNodeID, ResourceName: widget.ResourceName,
				Renderer: widget.Renderer, Settings: widget.Settings,
			}
		}
		document.Views[index] = ViewDefinition{
			ID: view.ID, Name: view.Name, Revision: view.Revision, Widgets: widgets,
			LayoutRoot: migrateLegacyLayout(view), CreatedAtUnixMS: view.CreatedAtUnixMS, UpdatedAtUnixMS: view.UpdatedAtUnixMS,
		}
	}
	return document
}

func migrateLegacyLayout(view legacyViewDefinition) *ViewLayoutNode {
	if len(view.Widgets) == 0 {
		return nil
	}
	leaf := func(widget legacyViewWidget) ViewLayoutNode {
		return ViewLayoutNode{Kind: "leaf", WidgetID: widget.ID}
	}
	if len(view.Widgets) == 1 {
		node := leaf(view.Widgets[0])
		return &node
	}
	if len(view.Widgets) == 2 {
		axis := "horizontal"
		ratio := float64(view.Widgets[0].W) / float64(view.Widgets[0].W+view.Widgets[1].W)
		if view.Layout != nil {
			axis = view.Layout.Direction
			ratio = view.Layout.SplitRatio
		}
		node := ViewLayoutNode{
			Kind: "split", Axis: axis,
			Children: []ViewLayoutNode{leaf(view.Widgets[0]), leaf(view.Widgets[1])},
			Weights:  normalizeWeights([]float64{ratio, 1 - ratio}),
		}
		return &node
	}

	type positionedWidget struct {
		widget legacyViewWidget
		index  int
	}
	positioned := make([]positionedWidget, len(view.Widgets))
	for index, widget := range view.Widgets {
		positioned[index] = positionedWidget{widget: widget, index: index}
	}
	sort.SliceStable(positioned, func(i, j int) bool {
		if positioned[i].widget.Y != positioned[j].widget.Y {
			return positioned[i].widget.Y < positioned[j].widget.Y
		}
		if positioned[i].widget.X != positioned[j].widget.X {
			return positioned[i].widget.X < positioned[j].widget.X
		}
		return positioned[i].index < positioned[j].index
	})

	var rows [][]positionedWidget
	for _, item := range positioned {
		if len(rows) == 0 || rows[len(rows)-1][0].widget.Y != item.widget.Y {
			rows = append(rows, []positionedWidget{item})
		} else {
			rows[len(rows)-1] = append(rows[len(rows)-1], item)
		}
	}
	rowNodes := make([]ViewLayoutNode, 0, len(rows))
	rowSizes := make([]float64, 0, len(rows))
	for _, row := range rows {
		children := make([]ViewLayoutNode, len(row))
		widths := make([]float64, len(row))
		maxHeight := 0
		for index, item := range row {
			children[index] = leaf(item.widget)
			widths[index] = float64(item.widget.W)
			if item.widget.H > maxHeight {
				maxHeight = item.widget.H
			}
		}
		if len(children) == 1 {
			rowNodes = append(rowNodes, children[0])
		} else {
			rowNodes = append(rowNodes, ViewLayoutNode{Kind: "split", Axis: "horizontal", Children: children, Weights: normalizeWeights(widths)})
		}
		rowSizes = append(rowSizes, float64(maxHeight))
	}
	if len(rowNodes) == 1 {
		return &rowNodes[0]
	}
	root := ViewLayoutNode{Kind: "split", Axis: "vertical", Children: rowNodes, Weights: normalizeWeights(rowSizes)}
	return &root
}

func normalizeWeights(weights []float64) []float64 {
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	normalized := make([]float64, len(weights))
	for index, weight := range weights {
		normalized[index] = weight / total
	}
	return normalized
}
