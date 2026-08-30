package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/sdk/bindings"
	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
)

func main() {
	stateDirectory := flag.String("state-dir", "", "desktop identity state directory")
	nodeID := flag.Int64("node-id", 0, "local NodeID")
	flag.Parse()
	if *stateDirectory == "" || *nodeID <= 0 {
		flag.Usage()
		os.Exit(2)
	}
	host, err := nodehost.New(context.Background(), nodehost.Config{
		StateDirectory: *stateDirectory, NodeID: protocol.NodeID(*nodeID),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer host.Close()
	core, err := bindings.NewAttachedClient(host.Client(), bindings.PublicIdentity{
		NodeID: host.ID(), PublicKey: host.PublicKey(),
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	client, err := desktopbinding.NewAttachedClient(core)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	identity, err := client.IdentityJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(identity)
}
