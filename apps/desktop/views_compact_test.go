package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestViewLayoutModesKeepIndependentGeometry(t *testing.T) {
	store, err := newViewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	height := 232
	input := ViewDefinition{ID: "compact", Name: "Cards", LayoutMode: "compact",
		Widgets:    []ViewWidget{{ID: "b", OwnerNodeID: "1", ResourceName: "qa/b", Renderer: "mfh.variable", CompactSize: &ViewCompactSize{Columns: 2, Height: &height}}, {ID: "a", OwnerNodeID: "1", ResourceName: "qa/a", Renderer: "mfh.variable"}},
		LayoutRoot: &ViewLayoutNode{Kind: "split", Axis: "vertical", Weights: []float64{0.3, 0.7}, Children: []ViewLayoutNode{{Kind: "leaf", WidgetID: "a"}, {Kind: "leaf", WidgetID: "b"}}},
	}
	saved, err := store.saveView(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"split", "compact"} {
		saved.LayoutMode = mode
		saved, err = store.saveView(saved)
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := store.load()
		if err != nil {
			t.Fatal(err)
		}
		actual := loaded.Views[0]
		if actual.LayoutMode != mode || !reflect.DeepEqual(actual.LayoutRoot, input.LayoutRoot) || !reflect.DeepEqual(actual.Widgets, input.Widgets) {
			t.Fatalf("layout state lost: %+v", actual)
		}
	}
	before, _ := os.ReadFile(store.path())
	saved.LayoutMode = "unknown"
	if _, err = store.saveView(saved); err == nil {
		t.Fatal("invalid layout accepted")
	}
	after, _ := os.ReadFile(store.path())
	if string(before) != string(after) {
		t.Fatal("invalid layout overwrote saved view")
	}
}

func TestViewCompactSizeBounds(t *testing.T) {
	for _, payload := range []string{`{"columns":0}`, `{"columns":5}`, `{"columns":1,"height":103}`, `{"columns":1,"height":1201}`, `{"columns":1.5}`} {
		t.Run(payload, func(t *testing.T) {
			var size ViewCompactSize
			if err := json.Unmarshal([]byte(payload), &size); err != nil {
				return
			}
			widget := ViewWidget{ID: "card", OwnerNodeID: "1", ResourceName: "qa/card", Renderer: "mfh.variable", CompactSize: &size}
			if err := validateViewWidget(widget, 0); err == nil {
				t.Fatal("invalid size accepted")
			}
		})
	}
}

func TestViewWithoutModeKeepsSplitLayout(t *testing.T) {
	document, err := decodeViewDocument([]byte(`{"version":3,"views":[{"id":"old","name":"Old","revision":1,"widgets":[],"created_at_unix_ms":0,"updated_at_unix_ms":0}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if document.Views[0].LayoutMode != "" {
		t.Fatal("existing layout mode changed")
	}
}
