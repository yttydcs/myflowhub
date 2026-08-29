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
	desktopViewsVersion       = 2
	legacyDesktopViewsVersion = 1
)

type ViewDocument struct {
	Version int              `json:"version"`
	Views   []ViewDefinition `json:"views"`
}

type ViewDefinition struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Revision        uint64       `json:"revision"`
	Widgets         []ViewWidget `json:"widgets"`
	Layout          *ViewLayout  `json:"layout,omitempty"`
	CreatedAtUnixMS int64        `json:"created_at_unix_ms"`
	UpdatedAtUnixMS int64        `json:"updated_at_unix_ms"`
}

type ViewLayout struct {
	Direction  string  `json:"direction"`
	SplitRatio float64 `json:"split_ratio"`
}

type ViewWidget struct {
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
	data, err := os.ReadFile(s.path())
	if errors.Is(err, os.ErrNotExist) {
		return ViewDocument{Version: desktopViewsVersion, Views: []ViewDefinition{}}, nil
	}
	if err != nil {
		return ViewDocument{}, fmt.Errorf("read views: %w", err)
	}
	var document ViewDocument
	if err := decodeStrictJSON(data, &document); err != nil {
		return ViewDocument{}, fmt.Errorf("view store is corrupt and was not overwritten: %w", err)
	}
	if document.Version == legacyDesktopViewsVersion {
		for index, view := range document.Views {
			if view.Layout != nil {
				return ViewDocument{}, fmt.Errorf("view store is corrupt and was not overwritten: views[%d] uses layout with legacy version", index)
			}
		}
		document.Version = desktopViewsVersion
	}
	if err := validateViewDocument(document); err != nil {
		return ViewDocument{}, fmt.Errorf("view store is corrupt and was not overwritten: %w", err)
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
		if len(document.Views) >= 32 {
			return ViewDefinition{}, errors.New("profile supports at most 32 views")
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
	if len(document.Views) > 32 {
		return errors.New("view count exceeds 32")
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
	if len(view.Widgets) > 64 {
		return errors.New("view supports at most 64 widgets")
	}
	if view.Layout != nil {
		if view.Layout.Direction != "horizontal" && view.Layout.Direction != "vertical" {
			return errors.New("view layout direction must be horizontal or vertical")
		}
		if math.IsNaN(view.Layout.SplitRatio) || math.IsInf(view.Layout.SplitRatio, 0) || view.Layout.SplitRatio < 0.2 || view.Layout.SplitRatio > 0.8 {
			return errors.New("view layout split ratio must be between 0.2 and 0.8")
		}
	}
	seen := make(map[string]struct{}, len(view.Widgets))
	for index, widget := range view.Widgets {
		if !profileIDPattern.MatchString(widget.ID) {
			return fmt.Errorf("widgets[%d] id is invalid", index)
		}
		if _, exists := seen[widget.ID]; exists {
			return fmt.Errorf("duplicate widget %q", widget.ID)
		}
		seen[widget.ID] = struct{}{}
		if _, err := parsePositiveInt64(widget.OwnerNodeID, "widget owner_node_id"); err != nil {
			return err
		}
		if widget.ResourceName == "" || len(widget.ResourceName) > 255 || strings.TrimSpace(widget.Renderer) == "" {
			return fmt.Errorf("widgets[%d] resource or renderer is invalid", index)
		}
		if widget.X < 0 || widget.X >= 12 || widget.Y < 0 || widget.W < 1 || widget.W > 12 || widget.H < 1 || widget.H > 24 || widget.X+widget.W > 12 {
			return fmt.Errorf("widgets[%d] layout is outside the 12-column grid", index)
		}
		if len(widget.Settings) > 64<<10 || (len(widget.Settings) > 0 && !json.Valid(widget.Settings)) {
			return fmt.Errorf("widgets[%d] settings are invalid", index)
		}
	}
	return nil
}
