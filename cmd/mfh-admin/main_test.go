package main

import (
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestAdminResourceOwnerRoutesAdmissionToAuthority(t *testing.T) {
	options := options{resource: protocol.BuiltinAdmissionListPermits}
	if got := adminResourceOwner(options, 9, 1); got != 1 {
		t.Fatalf("Admission owner = %d, want Authority 1", got)
	}
	options.resource = protocol.BuiltinResourceCatalog
	if got := adminResourceOwner(options, 9, 1); got != 9 {
		t.Fatalf("ordinary resource owner = %d, want parent 9", got)
	}
	options.ownerID = 7
	if got := adminResourceOwner(options, 9, 1); got != 7 {
		t.Fatalf("explicit owner = %d, want 7", got)
	}
}
