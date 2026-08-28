package file_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	filefeature "github.com/yttydcs/myflowhub/feature/file"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/transport/memory"
)

const (
	transferOne = "00112233445566778899aabbccddeeff"
	transferTwo = "10112233445566778899aabbccddeeff"
)

func TestOrderedIdempotentTransferCompletesAtomically(t *testing.T) {
	runtime, root := localNode(t)
	controller, err := filefeature.Register(filefeature.Config{Node: runtime, Root: root, MaxFileBytes: 1024, MaxTotalBytes: 2048})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	content := []byte("hello world")
	offer := protocol.FileOfferV1{Version: 1, TransferID: transferOne, Path: "inbox/greeting.txt", Size: int64(len(content)), SHA256: sum(content), ChunkSize: 5, ExpiresAtUnixMS: time.Now().Add(time.Hour).UnixMilli()}
	session := openSession(t, runtime, offer)
	send := func(offset int64, data []byte) protocol.FileProgressV1 {
		result, err := session.Send(context.Background(), offset, data, sum(data))
		if err != nil {
			t.Fatal(err)
		}
		var progress protocol.FileProgressV1
		if err := protocol.DecodeJSONPayload(result.Payload, protocol.DefaultMaxPayload, &progress); err != nil {
			t.Fatal(err)
		}
		return progress
	}
	send(0, content[:5])
	duplicate := send(0, content[:5])
	if duplicate.ReceivedBytes != 5 {
		t.Fatal("duplicate chunk advanced transfer")
	}
	send(5, content[5:10])
	send(10, content[10:])
	closed, err := session.Close(context.Background(), true, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var result protocol.FileProgressV1
	if err := protocol.DecodeJSONPayload(closed.Payload, protocol.DefaultMaxPayload, &result); err != nil {
		t.Fatal(err)
	}
	if result.State != "completed" || result.ReceivedBytes != int64(len(content)) {
		t.Fatalf("unexpected completion: %#v", result)
	}
	got, err := os.ReadFile(filepath.Join(root, "inbox", "greeting.txt"))
	if err != nil || string(got) != string(content) {
		t.Fatalf("completed file mismatch: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".mfh-tmp", transferOne+".part")); !os.IsNotExist(err) {
		t.Fatal("temporary file remained after completion")
	}
}

func TestZeroLengthGapChecksumAndCancel(t *testing.T) {
	runtime, root := localNode(t)
	controller, err := filefeature.Register(filefeature.Config{Node: runtime, Root: root, MaxFileBytes: 1024, MaxTotalBytes: 2048})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	empty := []byte{}
	offerEmpty := protocol.FileOfferV1{Version: 1, TransferID: transferOne, Path: "empty.bin", Size: 0, SHA256: sum(empty), ChunkSize: 8, ExpiresAtUnixMS: time.Now().Add(time.Hour).UnixMilli()}
	emptySession := openSession(t, runtime, offerEmpty)
	if _, err := emptySession.Close(context.Background(), true, "", nil); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(root, "empty.bin")); err != nil || info.Size() != 0 {
		t.Fatalf("zero-length file was not completed: %v", err)
	}

	content := []byte("abcdef")
	offer := protocol.FileOfferV1{Version: 1, TransferID: transferTwo, Path: "cancel.bin", Size: 6, SHA256: sum(content), ChunkSize: 3, ExpiresAtUnixMS: time.Now().Add(time.Hour).UnixMilli()}
	session := openSession(t, runtime, offer)
	if _, err := session.Send(context.Background(), 3, content[3:], sum(content[3:])); err == nil {
		t.Fatal("chunk gap was accepted")
	}
	if _, err := session.Send(context.Background(), 0, content[:3], sum([]byte("wrong"))); err == nil {
		t.Fatal("bad chunk checksum was accepted")
	}
	if _, err := session.Close(context.Background(), false, "text/plain", []byte("user")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".mfh-tmp", transferTwo+".part")); !os.IsNotExist(err) {
		t.Fatal("cancelled transfer temporary file remained")
	}
}

func TestTransferOwnershipAndStartupCleanup(t *testing.T) {
	root := t.TempDir()
	tempRoot := filepath.Join(root, ".mfh-tmp")
	if err := os.Mkdir(tempRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(tempRoot, transferOne+".part")
	if err := os.WriteFile(stale, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	rootIdentity, _ := auth.GenerateIdentity(1)
	childAIdentity, _ := auth.GenerateIdentity(2)
	childBIdentity, _ := auth.GenerateIdentity(3)
	rootTrust := auth.NewTrustStore()
	for _, identity := range []auth.Identity{rootIdentity, childAIdentity, childBIdentity} {
		_ = rootTrust.Add(identity.NodeID, identity.PublicKey)
	}
	policy := auth.NewStaticPolicy()
	for _, child := range []protocol.NodeID{2, 3} {
		policy.Allow(auth.Request{Subject: child, Action: auth.ActionOpen, Resource: protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFileUpload}})
	}
	rootNode, _ := node.New(context.Background(), node.Config{Identity: rootIdentity, Trust: rootTrust, Policy: policy})
	defer rootNode.Close()
	controller, err := filefeature.Register(filefeature.Config{Node: rootNode, Root: root, MaxFileBytes: 1024, MaxTotalBytes: 2048})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale partial transfer was not cleaned on startup")
	}
	network := memory.NewNetwork()
	defer network.Close()
	if _, err := rootNode.Listen(network, "root"); err != nil {
		t.Fatal(err)
	}
	childA := connectedChild(t, network, rootIdentity, childAIdentity)
	childB := connectedChild(t, network, rootIdentity, childBIdentity)
	content := []byte("abc")
	offer := protocol.FileOfferV1{Version: 1, TransferID: transferTwo, Path: "owned.bin", Size: 3, SHA256: sum(content), ChunkSize: 3, ExpiresAtUnixMS: time.Now().Add(time.Hour).UnixMilli()}
	_ = openSession(t, childA, offer)
	payload, _ := protocol.EncodeJSONPayload(&offer, protocol.DefaultMaxPayload)
	if _, err := childB.OpenSession(context.Background(), protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFileUpload}, protocol.CapabilityOpen, protocol.SchemaFileOfferV1, payload); err == nil {
		t.Fatal("a different caller reopened another node's transfer")
	}
}

func TestTransferExpiresWithoutAnotherCommand(t *testing.T) {
	runtime, root := localNode(t)
	controller, err := filefeature.Register(filefeature.Config{Node: runtime, Root: root, MaxFileBytes: 1024, MaxTotalBytes: 2048, MaxLifetime: time.Second, SweepInterval: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer controller.Close()
	content := []byte("abc")
	offer := protocol.FileOfferV1{Version: 1, TransferID: transferOne, Path: "expires.bin", Size: 3, SHA256: sum(content), ChunkSize: 3, ExpiresAtUnixMS: time.Now().Add(60 * time.Millisecond).UnixMilli()}
	_ = openSession(t, runtime, offer)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		value, _ := runtime.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFileTransfers})
		var transfers protocol.FileTransfersV1
		if err := protocol.DecodeJSONPayload(value.(*resource.Variable).Snapshot().Value, protocol.DefaultMaxPayload, &transfers); err != nil {
			t.Fatal(err)
		}
		if len(transfers.Transfers) == 1 && transfers.Transfers[0].State == "failed" {
			if _, err := os.Stat(filepath.Join(root, ".mfh-tmp", transferOne+".part")); !os.IsNotExist(err) {
				t.Fatal("expired temporary file remained")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("transfer did not expire automatically")
}

func localNode(t *testing.T) (*node.Node, string) {
	t.Helper()
	identity, _ := auth.GenerateIdentity(1)
	trust := auth.NewTrustStore()
	_ = trust.Add(1, identity.PublicKey)
	runtime, err := node.New(context.Background(), node.Config{Identity: identity, Trust: trust})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	return runtime, t.TempDir()
}

func connectedChild(t *testing.T, network *memory.Network, parent auth.Identity, child auth.Identity) *node.Node {
	t.Helper()
	trust := auth.NewTrustStore()
	_ = trust.Add(parent.NodeID, parent.PublicKey)
	runtime, err := node.New(context.Background(), node.Config{Identity: child, Trust: trust})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	if err := runtime.ConnectParent(context.Background(), network, "root", parent.NodeID); err != nil {
		t.Fatal(err)
	}
	return runtime
}

func openSession(t *testing.T, runtime *node.Node, offer protocol.FileOfferV1) *node.RemoteSession {
	t.Helper()
	payload, err := protocol.EncodeJSONPayload(&offer, protocol.DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	session, err := runtime.OpenSession(context.Background(), protocol.ResourceID{Owner: 1, Name: protocol.BuiltinFileUpload}, protocol.CapabilityOpen, protocol.SchemaFileOfferV1, payload)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func sum(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}
