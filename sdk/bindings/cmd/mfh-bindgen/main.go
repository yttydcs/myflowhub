package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yttydcs/myflowhub/sdk/bindings/contract"
)

func main() {
	out := flag.String("out", "", "output contract manifest")
	flag.Parse()
	if *out == "" {
		fatal(errors.New("-out is required"))
	}
	manifest, err := contract.Canonical()
	if err != nil {
		fatal(err)
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(manifest); err != nil {
		fatal(fmt.Errorf("encode binding contract: %w", err))
	}
	directory := filepath.Dir(*out)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		fatal(fmt.Errorf("create binding output directory: %w", err))
	}
	temporary, err := os.CreateTemp(directory, ".contracts-*.json")
	if err != nil {
		fatal(fmt.Errorf("create binding output: %w", err))
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(buffer.Bytes()); err != nil {
		_ = temporary.Close()
		fatal(fmt.Errorf("write binding output: %w", err))
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		fatal(fmt.Errorf("sync binding output: %w", err))
	}
	if err := temporary.Close(); err != nil {
		fatal(fmt.Errorf("close binding output: %w", err))
	}
	if err := os.Rename(temporaryName, *out); err != nil {
		fatal(fmt.Errorf("replace binding output: %w", err))
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
