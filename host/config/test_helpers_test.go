package config

import (
	"context"

	"github.com/yttydcs/myflowhub/protocol"
)

func resourceID(owner protocol.NodeID, name string) protocol.ResourceID {
	return protocol.ResourceID{Owner: owner, Name: name}
}

func testContext() context.Context { return context.Background() }
