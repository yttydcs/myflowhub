package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/sdk/bindings/contract"
)

func main() {
	out := flag.String("out", "", "output contract manifest")
	schemasOut := flag.String("schemas-out", "", "optional output for Desktop data schemas")
	flag.Parse()
	if *out == "" {
		fatal(errors.New("-out is required"))
	}
	manifest, err := contract.Canonical()
	if err != nil {
		fatal(err)
	}
	contractBytes, err := encodeJSON(manifest)
	if err != nil {
		fatal(fmt.Errorf("encode binding contract: %w", err))
	}
	if err := writeAtomic(*out, contractBytes); err != nil {
		fatal(err)
	}
	if *schemasOut != "" {
		schemaBytes, err := protocol.MarshalBuiltinDataSchemas()
		if err != nil {
			fatal(fmt.Errorf("build Desktop data schemas: %w", err))
		}
		schemaBytes = append(schemaBytes, '\n')
		if err := writeAtomic(*schemasOut, schemaBytes); err != nil {
			fatal(err)
		}
	}
}

func encodeJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeAtomic(path string, content []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create generated output directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".mfh-generated-*.tmp")
	if err != nil {
		return fmt.Errorf("create generated output: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write generated output: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync generated output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close generated output: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("replace generated output: %w", err)
	}
	return nil
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
