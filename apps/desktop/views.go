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
	desktopViewsVersion = 3

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
	LayoutMode      string          `json:"layout_mode,omitempty"`
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
	ID           string           `json:"id"`
	OwnerNodeID  string           `json:"owner_node_id"`
	ResourceName string           `json:"resource_name"`
	Renderer     string           `json:"renderer"`
	Settings     json.RawMessage  `json:"settings,omitempty"`
	CompactSize  *ViewCompactSize `json:"compact_size,omitempty"`
}

type ViewCompactSize struct {
	Columns int  `json:"columns"`
	Height  *int `json:"height,omitempty"`
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
	data, err := os.ReadFile(s.path())
	if errors.Is(err, os.ErrNotExist) {
		return ViewDocument{Version: desktopViewsVersion, Views: []ViewDefinition{}}, nil
	}
	if err != nil {
		return ViewDocument{}, fmt.Errorf("read views: %w", err)
	}
	document, err := decodeViewDocument(data)
	if err != nil {
		return ViewDocument{}, fmt.Errorf("view store is corrupt or unsupported and was not overwritten: %w", err)
	}
	return document, nil
}

func decodeViewDocument(data []byte) (ViewDocument, error) {
	var document ViewDocument
	if err := decodeStrictJSON(data, &document); err != nil {
		return ViewDocument{}, err
	}
	if document.Version != desktopViewsVersion {
		return ViewDocument{}, fmt.Errorf("unsupported views version %d; expected %d", document.Version, desktopViewsVersion)
	}
	if err := validateViewDocument(document); err != nil {
		return ViewDocument{}, err
	}
	return document, nil
}

func (s *viewStore) saveView(next ViewDefinition) (ViewDefinition, error) {
	document, err := s.load()
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
	if err := s.save(document); err != nil {
		return ViewDefinition{}, err
	}
	return next, nil
}

func (s *viewStore) deleteView(id string, revision uint64) error {
	document, err := s.load()
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
		return s.save(document)
	}
	return errors.New("view not found")
}

func (s *viewStore) save(document ViewDocument) error {
	document.Version = desktopViewsVersion
	if err := validateViewDocument(document); err != nil {
		return err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode views: %w", err)
	}
	return writeAtomicDesktop(s.root, s.path(), append(data, '\n'))
}

func (s *viewStore) path() string { return filepath.Join(s.root, "views.json") }

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
	if view.LayoutMode != "" && view.LayoutMode != "compact" && view.LayoutMode != "split" {
		return errors.New("view layout_mode must be compact or split")
	}
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
	if size := widget.CompactSize; size != nil {
		if size.Columns < 1 || size.Columns > 4 {
			return fmt.Errorf("widgets[%d] compact columns must be between 1 and 4", index)
		}
		if size.Height != nil && (*size.Height < 104 || *size.Height > 1200) {
			return fmt.Errorf("widgets[%d] compact height must be between 104 and 1200", index)
		}
	}
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
