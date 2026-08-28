package protocol

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

func validEnvelope() Envelope {
	return Envelope{
		Version:        CurrentVersion,
		Phase:          PhaseControl,
		Operation:      OperationOperate,
		MessageID:      MustMessageID(),
		Source:         10,
		Principal:      11,
		Target:         20,
		Resource:       ResourceID{Owner: 20, Name: "system/restart"},
		TopologyEpoch:  7,
		DeadlineUnixMS: 123456,
		ContentType:    "application/json",
		Schema:         "command.restart.v1",
		Capability:     CapabilityInvoke,
		Payload:        []byte(`{"delay_ms":10}`),
	}
}

func TestCodecRoundTripHandlesShortWrites(t *testing.T) {
	want := validEnvelope()
	var buffer bytes.Buffer
	writer := shortWriter{writer: &buffer, max: 3}
	if err := (Codec{}).Encode(writer, want); err != nil {
		t.Fatal(err)
	}
	got, err := (Codec{}).Decode(&buffer)
	if err != nil {
		t.Fatal(err)
	}
	if !envelopesEqual(got, want) {
		t.Fatalf("roundtrip mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestCodecRejectsMalformedAndOversizeFrames(t *testing.T) {
	want := validEnvelope()
	var encoded bytes.Buffer
	if err := (Codec{}).Encode(&encoded, want); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "truncated header", input: encoded.Bytes()[:20]},
		{name: "truncated body", input: encoded.Bytes()[:len(encoded.Bytes())-1]},
		{name: "bad magic", input: append([]byte("NOPE"), encoded.Bytes()[4:]...)},
	}
	badVersion := append([]byte(nil), encoded.Bytes()...)
	badVersion[5] = byte(CurrentVersion + 1)
	tests = append(tests, struct {
		name  string
		input []byte
	}{name: "unknown version", input: badVersion})
	badOperation := append([]byte(nil), encoded.Bytes()...)
	badOperation[7] = 255
	tests = append(tests, struct {
		name  string
		input []byte
	}{name: "unknown operation", input: badOperation})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := (Codec{}).Decode(bytes.NewReader(test.input)); err == nil {
				t.Fatal("expected decode failure")
			}
		})
	}
	if err := (Codec{MaxPayload: 1}).Encode(io.Discard, want); !errors.Is(err, ErrPayloadTooLarge) {
		t.Fatalf("expected payload limit error, got %v", err)
	}
}

func TestEnvelopeValidatesPhaseAndResource(t *testing.T) {
	envelope := validEnvelope()
	envelope.Phase = PhaseEvent
	if err := envelope.Validate(DefaultMaxPayload); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("expected invalid phase, got %v", err)
	}
	envelope = validEnvelope()
	envelope.Resource.Name = "../escape"
	if err := envelope.Validate(DefaultMaxPayload); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("expected invalid resource, got %v", err)
	}
}

func FuzzCodecRoundTrip(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, payload []byte) {
		if len(payload) > 4096 {
			t.Skip()
		}
		envelope := validEnvelope()
		envelope.Payload = payload
		var buffer bytes.Buffer
		codec := Codec{MaxPayload: 4096}
		if err := codec.Encode(&buffer, envelope); err != nil {
			t.Fatal(err)
		}
		got, err := codec.Decode(&buffer)
		if err != nil {
			t.Fatal(err)
		}
		if !envelopesEqual(got, envelope) {
			t.Fatal("roundtrip mismatch")
		}
	})
}

func envelopesEqual(left, right Envelope) bool {
	if !bytes.Equal(left.Payload, right.Payload) {
		return false
	}
	left.Payload = nil
	right.Payload = nil
	return reflect.DeepEqual(left, right)
}

type shortWriter struct {
	writer io.Writer
	max    int
}

func (w shortWriter) Write(data []byte) (int, error) {
	if len(data) > w.max {
		data = data[:w.max]
	}
	return w.writer.Write(data)
}
