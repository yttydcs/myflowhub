package protocol

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestFilesystemPayloadsValidate(t *testing.T) {
	request := FilesystemReadRequestV1{
		Version: SchemaVersionV1, Key: "docs/readme.txt", MaxBytes: 4096, ExpectedRevision: "revision-1",
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	text := FilesystemContentV1{
		Version: SchemaVersionV1, Key: request.Key, ContentType: "text/plain; charset=utf-8",
		Encoding: FilesystemEncodingUTF8, Data: "hello", Size: 5, ModifiedUnixMS: 1, Revision: "revision-1",
	}
	if err := text.Validate(); err != nil {
		t.Fatal(err)
	}
	binary := text
	binary.ContentType = "application/octet-stream"
	binary.Encoding = FilesystemEncodingBase64
	binary.Data = base64.StdEncoding.EncodeToString([]byte{0xff, 0x00})
	binary.Size = 2
	if err := binary.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestFilesystemReadRequestRejectsInvalidFields(t *testing.T) {
	tests := []FilesystemReadRequestV1{
		{Version: 2, Key: "a", MaxBytes: 1},
		{Version: 1, MaxBytes: 1},
		{Version: 1, Key: strings.Repeat("k", MaxCollectionMemberKeyBytes+1), MaxBytes: 1},
		{Version: 1, Key: "a"},
		{Version: 1, Key: "a", MaxBytes: MaxFilesystemReadBytes + 1},
		{Version: 1, Key: "a", MaxBytes: 1, ExpectedRevision: strings.Repeat("r", MaxFilesystemRevisionBytes+1)},
	}
	for index, value := range tests {
		if err := value.Validate(); err == nil {
			t.Fatalf("invalid request %d was accepted: %#v", index, value)
		}
	}
}

func TestFilesystemContentRejectsInvalidEncodingSizeAndMetadata(t *testing.T) {
	valid := FilesystemContentV1{
		Version: SchemaVersionV1, Key: "a", ContentType: "text/plain", Encoding: FilesystemEncodingUTF8,
		Data: "a", Size: 1, ModifiedUnixMS: 1, Revision: "revision",
	}
	tests := []struct {
		name   string
		mutate func(*FilesystemContentV1)
	}{
		{"version", func(value *FilesystemContentV1) { value.Version = 2 }},
		{"key", func(value *FilesystemContentV1) { value.Key = "" }},
		{"content-type", func(value *FilesystemContentV1) { value.ContentType = "" }},
		{"revision", func(value *FilesystemContentV1) { value.Revision = "" }},
		{"size-negative", func(value *FilesystemContentV1) { value.Size = -1 }},
		{"size-large", func(value *FilesystemContentV1) { value.Size = MaxFilesystemReadBytes + 1 }},
		{"modified", func(value *FilesystemContentV1) { value.ModifiedUnixMS = -1 }},
		{"encoding", func(value *FilesystemContentV1) { value.Encoding = "raw" }},
		{"utf8", func(value *FilesystemContentV1) { value.Data = string([]byte{0xff}) }},
		{"utf8-size", func(value *FilesystemContentV1) { value.Size = 2 }},
		{"base64", func(value *FilesystemContentV1) { value.Encoding = FilesystemEncodingBase64; value.Data = "*" }},
		{"base64-size", func(value *FilesystemContentV1) {
			value.Encoding = FilesystemEncodingBase64
			value.Data = base64.StdEncoding.EncodeToString([]byte("a"))
			value.Size = 2
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := valid
			test.mutate(&value)
			if err := value.Validate(); err == nil {
				t.Fatalf("invalid content was accepted: %#v", value)
			}
		})
	}
}

func TestFilesystemPayloadsLeaveEnvelopeHeadroom(t *testing.T) {
	data := strings.Repeat("\x01", MaxFilesystemReadBytes)
	value := FilesystemContentV1{
		Version: SchemaVersionV1, Key: "a", ContentType: "text/plain", Encoding: FilesystemEncodingUTF8,
		Data: data, Size: MaxFilesystemReadBytes, ModifiedUnixMS: 1, Revision: strings.Repeat("r", MaxFilesystemRevisionBytes),
	}
	payload, err := EncodeJSONPayload(&value, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) >= DefaultMaxPayload {
		t.Fatalf("filesystem payload has no envelope headroom: %d", len(payload))
	}
}
