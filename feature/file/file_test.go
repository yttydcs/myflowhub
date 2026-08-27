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
	invoke(t, runtime, protocol.BuiltinFileOffer, &offer, &protocol.FileProgressV1{})
	first := protocol.FileChunkV1{Version: 1, TransferID: transferOne, Offset: 0, Data: content[:5], SHA256: sum(content[:5])}
	invoke(t, runtime, protocol.BuiltinFileChunk, &first, &protocol.FileProgressV1{})
	duplicate := invoke(t, runtime, protocol.BuiltinFileChunk, &first, &protocol.FileProgressV1{})
	if duplicate.(*protocol.FileProgressV1).ReceivedBytes != 5 {
		t.Fatal("duplicate chunk advanced transfer")
	}
	second := protocol.FileChunkV1{Version: 1, TransferID: transferOne, Offset: 5, Data: content[5:10], SHA256: sum(content[5:10])}
	third := protocol.FileChunkV1{Version: 1, TransferID: transferOne, Offset: 10, Data: content[10:], SHA256: sum(content[10:])}
	invoke(t, runtime, protocol.BuiltinFileChunk, &second, &protocol.FileProgressV1{})
	invoke(t, runtime, protocol.BuiltinFileChunk, &third, &protocol.FileProgressV1{})
	complete := protocol.FileCompleteV1{Version: 1, TransferID: transferOne, Size: int64(len(content)), SHA256: sum(content)}
	result := invoke(t, runtime, protocol.BuiltinFileComplete, &complete, &protocol.FileProgressV1{}).(*protocol.FileProgressV1)
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
	invoke(t, runtime, protocol.BuiltinFileOffer, &offerEmpty, &protocol.FileProgressV1{})
	invoke(t, runtime, protocol.BuiltinFileComplete, &protocol.FileCompleteV1{Version: 1, TransferID: transferOne, Size: 0, SHA256: sum(empty)}, &protocol.FileProgressV1{})
	if info, err := os.Stat(filepath.Join(root, "empty.bin")); err != nil || info.Size() != 0 {
		t.Fatalf("zero-length file was not completed: %v", err)
	}

	content := []byte("abcdef")
	offer := protocol.FileOfferV1{Version: 1, TransferID: transferTwo, Path: "cancel.bin", Size: 6, SHA256: sum(content), ChunkSize: 3, ExpiresAtUnixMS: time.Now().Add(time.Hour).UnixMilli()}
	invoke(t, runtime, protocol.BuiltinFileOffer, &offer, &protocol.FileProgressV1{})
	gap := protocol.FileChunkV1{Version: 1, TransferID: transferTwo, Offset: 3, Data: content[3:], SHA256: sum(content[3:])}
	if _, err := invokeError(runtime, protocol.BuiltinFileChunk, &gap); err == nil {
		t.Fatal("chunk gap was accepted")
	}
	bad := protocol.FileChunkV1{Version: 1, TransferID: transferTwo, Offset: 0, Data: content[:3], SHA256: sum([]byte("wrong"))}
	if _, err := invokeError(runtime, protocol.BuiltinFileChunk, &bad); err == nil {
		t.Fatal("bad chunk checksum was accepted")
	}
	invoke(t, runtime, protocol.BuiltinFileCancel, &protocol.FileCancelV1{Version: 1, TransferID: transferTwo, Reason: "user"}, &protocol.FileProgressV1{})
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
		for _, name := range []string{protocol.BuiltinFileOffer, protocol.BuiltinFileChunk} {
			policy.Allow(auth.Request{Subject: child, Action: auth.ActionInvoke, Resource: protocol.ResourceID{Owner: 1, Name: name}})
		}
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
	if _, err := invokeRemote(childA, protocol.BuiltinFileOffer, &offer); err != nil {
		t.Fatal(err)
	}
	chunk := protocol.FileChunkV1{Version: 1, TransferID: transferTwo, Offset: 0, Data: content, SHA256: sum(content)}
	if _, err := invokeRemote(childB, protocol.BuiltinFileChunk, &chunk); err == nil {
		t.Fatal("a different caller wrote another node's transfer")
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
	invoke(t, runtime, protocol.BuiltinFileOffer, &offer, &protocol.FileProgressV1{})
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

func invoke(t *testing.T, runtime *node.Node, name string, request protocol.ValidatedPayload, response protocol.ValidatedPayload) protocol.ValidatedPayload {
	t.Helper()
	data, err := invokeRemote(runtime, name, request)
	if err != nil {
		t.Fatal(err)
	}
	if err := protocol.DecodeJSONPayload(data, protocol.DefaultMaxPayload, response); err != nil {
		t.Fatal(err)
	}
	return response
}

func invokeRemote(runtime *node.Node, name string, request protocol.ValidatedPayload) ([]byte, error) {
	payload, err := protocol.EncodeJSONPayload(request, protocol.DefaultMaxPayload)
	if err != nil {
		return nil, err
	}
	return runtime.Invoke(context.Background(), protocol.ResourceID{Owner: 1, Name: name}, payload)
}

func invokeError(runtime *node.Node, name string, request protocol.ValidatedPayload) ([]byte, error) {
	return invokeRemote(runtime, name, request)
}

func sum(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}
