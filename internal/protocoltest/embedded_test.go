package protocoltest

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
)

func TestEmbeddedEnvelopeGolden(t *testing.T) {
	wantHex, err := os.ReadFile(filepath.Join(fixtures(t), "..", "embedded", "envelope-command-v1.hex"))
	if err != nil {
		t.Fatal(err)
	}
	messageID := protocol.MessageID{0x10, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	envelope := protocol.Envelope{
		Version:        protocol.CurrentVersion,
		Phase:          protocol.PhaseRequest,
		Operation:      protocol.OperationCommandCall,
		MessageID:      messageID,
		Source:         2,
		Target:         1,
		Resource:       protocol.ResourceID{Owner: 1, Name: "device/led/set"},
		DeadlineUnixMS: 1000,
		ContentType:    "application/json",
		Schema:         "mfh.test.v1",
		Payload:        []byte(`{"version":1,"value":"true"}`),
	}
	var encoded bytes.Buffer
	if err := (protocol.Codec{MaxPayload: 8 << 10}).Encode(&encoded, envelope); err != nil {
		t.Fatal(err)
	}
	want, err := hex.DecodeString(strings.TrimSpace(string(wantHex)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded.Bytes(), want) {
		t.Fatalf("embedded frame drift\nwant: %x\n got: %x", want, encoded.Bytes())
	}
	decoded, err := (protocol.Codec{MaxPayload: 8 << 10}).Decode(bytes.NewReader(want))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Resource != envelope.Resource || !bytes.Equal(decoded.Payload, envelope.Payload) {
		t.Fatalf("decoded embedded fixture mismatch: %#v", decoded)
	}
}
