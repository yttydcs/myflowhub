package protocol

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"testing"
)

func TestEnrollmentCodecRoundTripHandlesShortWrites(t *testing.T) {
	want := EnrollmentFrame{
		Version: EnrollmentProtocolVersion,
		Type:    EnrollmentMessageClientInit,
		Payload: []byte(`{"version":1}`),
	}
	var buffer bytes.Buffer
	if err := (EnrollmentCodec{}).Encode(shortWriter{writer: &buffer, max: 2}, want); err != nil {
		t.Fatal(err)
	}
	got, err := (EnrollmentCodec{}).Decode(&buffer)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != want.Version || got.Type != want.Type || !bytes.Equal(got.Payload, want.Payload) {
		t.Fatalf("roundtrip mismatch: want %#v, got %#v", want, got)
	}
}

func TestEnrollmentCodecRejectsMalformedFrames(t *testing.T) {
	frame := EnrollmentFrame{Version: EnrollmentProtocolVersion, Type: EnrollmentMessageProof, Payload: []byte("proof")}
	var encoded bytes.Buffer
	if err := (EnrollmentCodec{}).Encode(&encoded, frame); err != nil {
		t.Fatal(err)
	}
	tests := [][]byte{
		encoded.Bytes()[:5],
		encoded.Bytes()[:len(encoded.Bytes())-1],
		append([]byte("MFH4"), encoded.Bytes()[4:]...),
	}
	badFlags := append([]byte(nil), encoded.Bytes()...)
	badFlags[7] = 1
	tests = append(tests, badFlags)
	for _, input := range tests {
		if _, err := (EnrollmentCodec{}).Decode(bytes.NewReader(input)); err == nil {
			t.Fatalf("expected decode error for %x", input)
		}
	}
	if err := (EnrollmentCodec{MaxPayload: 1}).Encode(io.Discard, frame); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("expected payload limit error, got %v", err)
	}
}

func TestEnrollmentPayloadsValidateIdentityBeforeNodeID(t *testing.T) {
	publicKey := base64.RawStdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	init := EnrollmentClientInitV1{
		Version:         SchemaVersionV1,
		RequestID:       "00112233445566778899aabbccddeeff",
		DevicePublicKey: publicKey,
	}
	if err := init.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := EncodeJSONPayload(init, EnrollmentMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	var decoded EnrollmentClientInitV1
	if err := DecodeJSONPayload(data, EnrollmentMaxPayload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.DevicePublicKey != publicKey || decoded.ExpectedParentNodeID != "" {
		t.Fatalf("unexpected decoded init: %#v", decoded)
	}
}

func TestEnrollmentResultRequiresConsistentStatus(t *testing.T) {
	result := EnrollmentResultV1{
		Version:   SchemaVersionV1,
		RequestID: "00112233445566778899aabbccddeeff",
		Status:    "granted",
	}
	if err := result.Validate(); err == nil {
		t.Fatal("expected missing grant to fail")
	}
	result.Status = "pending"
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
}
