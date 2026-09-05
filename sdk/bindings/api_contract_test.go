package bindings_test

import (
	"reflect"
	"testing"

	"github.com/yttydcs/myflowhub/sdk/bindings"
	"github.com/yttydcs/myflowhub/sdk/bindings/contract"
	"github.com/yttydcs/myflowhub/sdk/bindings/desktop"
)

func TestMethodManifestsMatchAttachedFacades(t *testing.T) {
	manifest, err := contract.Canonical()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		client  any
		methods []string
	}{
		{"portable", (*bindings.Client)(nil), manifest.Methods},
		{"desktop", (*desktop.Client)(nil), manifest.DesktopMethods},
	} {
		t.Run(test.name, func(t *testing.T) {
			typ := reflect.TypeOf(test.client)
			methods := make([]string, typ.NumMethod())
			for i := range methods {
				methods[i] = typ.Method(i).Name
			}
			if !reflect.DeepEqual(methods, test.methods) {
				t.Fatalf("manifest advertises a different API: got %v, want %v", test.methods, methods)
			}
		})
	}
}
