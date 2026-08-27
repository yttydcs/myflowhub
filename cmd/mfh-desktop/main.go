package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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
	client := &desktopbinding.Client{}
	if err := client.Open(*stateDirectory, *nodeID); err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	identity, err := client.IdentityJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(identity)
}
