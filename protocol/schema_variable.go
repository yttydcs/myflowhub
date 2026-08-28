package protocol

import "errors"

const SchemaVariableWriteV2 = "mfh.variable-write.v2"

type VariableWriteV2 struct {
	Version          int    `json:"version"`
	ExpectedRevision uint64 `json:"expected_revision"`
	Value            []byte `json:"value"`
}

func (w VariableWriteV2) Validate() error {
	if w.Version != SchemaVersionV2 {
		return errors.New("variable write version must be 2")
	}
	if w.ExpectedRevision == 0 {
		return errors.New("variable write expected_revision must be non-zero")
	}
	return nil
}
