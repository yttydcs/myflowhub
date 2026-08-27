package main

import (
	"flag"
	"log"
	"os"

	"github.com/yttydcs/myflowhub/apps/desktop/mcp"
	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
)

func main() {
	stateDirectory := flag.String("state-dir", "", "dedicated MCP identity state directory")
	nodeID := flag.Int64("node-id", 0, "local NodeID")
	allowWrite := flag.Bool("allow-write", false, "allow canonical Command invocation")
	endpoint := flag.String("endpoint", "", "optional parent TCP endpoint")
	parentID := flag.Int64("parent-id", 0, "parent NodeID")
	parentKey := flag.String("parent-key", "", "raw-base64 parent Ed25519 public key")
	permit := flag.String("permit", "", "optional provisioning permit JSON")
	flag.Parse()
	if *stateDirectory == "" || *nodeID <= 0 {
		flag.Usage()
		os.Exit(2)
	}
	client := &desktopbinding.Client{}
	if err := client.Open(*stateDirectory, *nodeID); err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	if *endpoint != "" {
		if *parentID <= 0 || *parentKey == "" {
			log.Fatal("parent-id and parent-key are required with endpoint")
		}
		if err := client.TrustParent(*parentID, *parentKey); err != nil {
			log.Fatal(err)
		}
		if err := client.StartTCP(*endpoint, *parentID, *permit); err != nil {
			log.Fatal(err)
		}
	}
	server, err := mcp.New(client, *allowWrite)
	if err != nil {
		log.Fatal(err)
	}
	if err := server.Serve(os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
