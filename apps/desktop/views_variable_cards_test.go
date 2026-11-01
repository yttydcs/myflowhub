package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func variableCardView() ViewDefinition {
	return ViewDefinition{ID: "cards", Name: "设备", Revision: 1, LayoutMode: "compact", Widgets: []ViewWidget{
		{ID: "group", Kind: "variable-list", Title: "笔记本状态", Variables: []ViewVariable{
			{ID: "cpu", OwnerNodeID: "1", ResourceName: "metrics/cpu", Renderer: "mfh.variable", Title: "处理器"},
			{ID: "battery", OwnerNodeID: "2", ResourceName: "metrics/battery", Renderer: "mfh.variable.text.v2"},
		}},
		{ID: "empty", Kind: "variable-list"},
	}, LayoutRoot: &ViewLayoutNode{Kind: "split", Axis: "horizontal", Children: []ViewLayoutNode{{Kind: "leaf", WidgetID: "group"}, {Kind: "leaf", WidgetID: "empty"}}, Weights: []float64{0.5, 0.5}}}
}

func TestVariableCardStoreRoundTrip(t *testing.T) {
	store, err := newViewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	input := variableCardView()
	input.Revision = 0
	saved, err := store.saveView(input)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, loaded.Views[0]) {
		t.Fatalf("round trip mismatch: %#v", loaded)
	}
	data, _ := os.ReadFile(store.path())
	if strings.Contains(string(data), "draft") || strings.Contains(string(data), "\"value\"") {
		t.Fatal("runtime state leaked")
	}
	saved.Widgets[0].Title = ""
	saved.Widgets[0].Variables[0].Title = ""
	saved.Widgets[0].Variables[0].ID = "stable-reference"
	saved, err = store.saveView(saved)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision != 2 || saved.Widgets[0].Variables[0].Title != "" {
		t.Fatal("rename reset failed")
	}
}

func TestVariableCardValidation(t *testing.T) {
	tests := map[string]func(*ViewDefinition){
		"unknown kind":        func(v *ViewDefinition) { v.Widgets[0].Kind = "other" },
		"top binding":         func(v *ViewDefinition) { v.Widgets[0].OwnerNodeID = "1" },
		"oversize title":      func(v *ViewDefinition) { v.Widgets[0].Title = strings.Repeat("中", 54) },
		"invalid owner":       func(v *ViewDefinition) { v.Widgets[0].Variables[0].OwnerNodeID = "0" },
		"non-variable":        func(v *ViewDefinition) { v.Widgets[0].Variables[0].Renderer = "mfh.command" },
		"duplicate reference": func(v *ViewDefinition) { v.Widgets[0].Variables[1].ID = "cpu" },
		"duplicate resource": func(v *ViewDefinition) {
			v.Widgets[0].Variables[1].OwnerNodeID = "1"
			v.Widgets[0].Variables[1].ResourceName = "metrics/cpu"
		},
		"cross card duplicate": func(v *ViewDefinition) { v.Widgets[1].Variables = []ViewVariable{v.Widgets[0].Variables[0]} },
		"too many rows":        func(v *ViewDefinition) { v.Widgets[0].Variables = make([]ViewVariable, 65) },
		"invalid row settings": func(v *ViewDefinition) { v.Widgets[0].Variables[0].Settings = json.RawMessage("{") },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			view := variableCardView()
			change(&view)
			if validateView(view) == nil {
				t.Fatal("accepted invalid card")
			}
		})
	}
}

func TestLegacyAndStableSingleVariableReferences(t *testing.T) {
	view := ViewDefinition{ID: "legacy", Name: "Legacy", Revision: 1, Widgets: []ViewWidget{{ID: "single", OwnerNodeID: "1", ResourceName: "metrics/cpu", Renderer: "mfh.variable"}}, LayoutRoot: &ViewLayoutNode{Kind: "leaf", WidgetID: "single"}}
	if err := validateView(view); err != nil {
		t.Fatal(err)
	}
	view.Widgets[0].VariableID = "stable-reference"
	view.Widgets[0].Title = "CPU 使用率"
	if err := validateView(view); err != nil {
		t.Fatal(err)
	}
	view.Widgets[0].Variables = []ViewVariable{}
	if validateView(view) == nil {
		t.Fatal("accepted list on single card")
	}
}
