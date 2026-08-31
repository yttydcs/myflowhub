package filesystem

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestRegisterThreeMountsAndCatalogHidesPhysicalRoots(t *testing.T) {
	runtimeNode := newTestNode(t)
	base := t.TempDir()
	mounts := make([]Mount, 0, 3)
	for _, name := range []string{"documents", "media", "reports"} {
		root := filepath.Join(base, name)
		mustMkdir(t, root)
		mounts = append(mounts, Mount{Name: name, Root: root, Label: strings.ToUpper(name[:1]) + name[1:]})
	}

	registration, err := Register(runtimeNode, mounts)
	if err != nil {
		t.Fatal(err)
	}
	for _, mount := range mounts {
		descriptor, ok := runtimeNode.Registry().Descriptor(protocol.ResourceID{Owner: runtimeNode.ID(), Name: mount.Name})
		if !ok {
			t.Fatalf("missing descriptor for %q", mount.Name)
		}
		if descriptor.Type != protocol.ResourceTypeCollection {
			t.Fatalf("unexpected type for %q: %q", mount.Name, descriptor.Type)
		}
		got := make([]protocol.CapabilityID, 0, len(descriptor.Capabilities))
		for _, capability := range descriptor.Capabilities {
			got = append(got, capability.Name)
		}
		want := []protocol.CapabilityID{protocol.CapabilityGet, protocol.CapabilityList, protocol.CapabilityRead}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected capabilities for %q: got %v want %v", mount.Name, got, want)
		}
	}
	catalog := runtimeNode.Registry().Catalog().Snapshot().Value
	for _, mount := range mounts {
		if bytes.Contains(catalog, []byte(mount.Root)) || bytes.Contains(catalog, []byte(filepath.ToSlash(mount.Root))) {
			t.Fatalf("catalog leaks physical root for %q: %s", mount.Name, catalog)
		}
	}
	if err := registration.Close(); err != nil {
		t.Fatal(err)
	}
	if err := registration.Close(); err != nil {
		t.Fatal(err)
	}
	for _, mount := range mounts {
		if _, ok := runtimeNode.Registry().Resolve(protocol.ResourceID{Owner: runtimeNode.ID(), Name: mount.Name}); ok {
			t.Fatalf("resource %q remained after close", mount.Name)
		}
	}
}

func TestRegisterValidatesAllMountsBeforeChangingRegistry(t *testing.T) {
	t.Run("duplicate-name", func(t *testing.T) {
		runtimeNode := newTestNode(t)
		first, second := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")
		mustMkdir(t, first)
		mustMkdir(t, second)
		assertRegisterLeavesNone(t, runtimeNode, []Mount{{Name: "same", Root: first}, {Name: "same", Root: second}}, "same")
	})
	t.Run("duplicate-root", func(t *testing.T) {
		runtimeNode := newTestNode(t)
		root := t.TempDir()
		assertRegisterLeavesNone(t, runtimeNode, []Mount{{Name: "first", Root: root}, {Name: "second", Root: root}}, "first", "second")
	})
	t.Run("relative-root", func(t *testing.T) {
		runtimeNode := newTestNode(t)
		assertRegisterLeavesNone(t, runtimeNode, []Mount{{Name: "first", Root: t.TempDir()}, {Name: "second", Root: "relative"}}, "first", "second")
	})
	t.Run("missing-root", func(t *testing.T) {
		runtimeNode := newTestNode(t)
		base := t.TempDir()
		assertRegisterLeavesNone(t, runtimeNode, []Mount{{Name: "first", Root: base}, {Name: "second", Root: filepath.Join(base, "missing")}}, "first", "second")
	})
	t.Run("non-directory-root", func(t *testing.T) {
		runtimeNode := newTestNode(t)
		base := t.TempDir()
		file := filepath.Join(base, "file")
		mustWrite(t, file, []byte("not a directory"))
		assertRegisterLeavesNone(t, runtimeNode, []Mount{{Name: "first", Root: base}, {Name: "second", Root: file}}, "first", "second")
	})
	t.Run("preexisting-resource", func(t *testing.T) {
		runtimeNode := newTestNode(t)
		root := t.TempDir()
		takenID := protocol.ResourceID{Owner: runtimeNode.ID(), Name: "taken"}
		variable, err := resource.NewVariable(resource.VariableDescriptor(takenID, "application/octet-stream", "test.value.v1", "test.read", 64), []byte("existing"))
		if err != nil {
			t.Fatal(err)
		}
		if err := runtimeNode.Registry().Register(variable); err != nil {
			t.Fatal(err)
		}
		other := filepath.Join(root, "other")
		mustMkdir(t, other)
		if _, err := Register(runtimeNode, []Mount{{Name: "first", Root: root}, {Name: "taken", Root: other}}); err == nil {
			t.Fatal("expected preexisting resource validation failure")
		}
		if _, ok := runtimeNode.Registry().Resolve(protocol.ResourceID{Owner: runtimeNode.ID(), Name: "first"}); ok {
			t.Fatal("first mount was registered before complete validation")
		}
		if current, ok := runtimeNode.Registry().Resolve(takenID); !ok || current != variable {
			t.Fatal("preexisting resource changed")
		}
	})
	t.Run("symlink-root", func(t *testing.T) {
		runtimeNode := newTestNode(t)
		base := t.TempDir()
		realRoot := filepath.Join(base, "real")
		linkRoot := filepath.Join(base, "link")
		mustMkdir(t, realRoot)
		if err := os.Symlink(realRoot, linkRoot); err != nil {
			t.Skipf("symlink creation is unavailable: %v", err)
		}
		assertRegisterLeavesNone(t, runtimeNode, []Mount{{Name: "linked", Root: linkRoot}}, "linked")
	})
}

func TestRegistrationCloseDoesNotRemoveSuccessor(t *testing.T) {
	runtimeNode := newTestNode(t)
	registration, err := Register(runtimeNode, []Mount{{Name: "files", Root: t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	id := protocol.ResourceID{Owner: runtimeNode.ID(), Name: "files"}
	if err := runtimeNode.Registry().Remove(id); err != nil {
		t.Fatal(err)
	}
	successor, err := resource.NewVariable(resource.VariableDescriptor(id, "application/octet-stream", "test.value.v1", "test.read", 64), []byte("successor"))
	if err != nil {
		t.Fatal(err)
	}
	if err := runtimeNode.Registry().Register(successor); err != nil {
		t.Fatal(err)
	}
	if err := registration.Close(); err != nil {
		t.Fatal(err)
	}
	if current, ok := runtimeNode.Registry().Resolve(id); !ok || current != successor {
		t.Fatal("registration close removed a successor resource")
	}
}

func TestListGetPaginationAndStaleCursor(t *testing.T) {
	runtimeNode := newTestNode(t)
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "b.txt"), []byte("b"))
	mustWrite(t, filepath.Join(root, "a.txt"), []byte("a"))
	mustWrite(t, filepath.Join(root, "你好.txt"), []byte("hello"))
	mustMkdir(t, filepath.Join(root, "empty"))
	mustMkdir(t, filepath.Join(root, "nested"))
	mustWrite(t, filepath.Join(root, "nested", "child.json"), []byte(`{"ok":true}`))
	id := registerOne(t, runtimeNode, Mount{Name: "files", Root: root})

	first, err := listMembers(runtimeNode, id, "", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := memberKeys(first.Members), []string{"a.txt", "b.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected first page: got %v want %v", got, want)
	}
	if first.Revision == 0 || first.Revision > protocol.MaxCollectionRevision || first.NextCursor == "" {
		t.Fatalf("unexpected first page metadata: %#v", first)
	}
	cursor, err := decodeCursor(first.NextCursor)
	if err != nil || cursor.Revision != first.Revision || len(cursor.Fingerprint) != sha256.Size*2 {
		t.Fatalf("cursor does not preserve the safe page revision and full fingerprint: %#v: %v", cursor, err)
	}
	second, err := listMembers(runtimeNode, id, "", first.NextCursor, protocol.MaxCollectionPageMembers)
	if err != nil {
		t.Fatal(err)
	}
	if second.Revision != first.Revision || second.NextCursor != "" {
		t.Fatalf("unexpected final page metadata: %#v", second)
	}
	all := append(memberKeys(first.Members), memberKeys(second.Members)...)
	wantAll := append([]string(nil), all...)
	sort.Strings(wantAll)
	if !reflect.DeepEqual(all, wantAll) {
		t.Fatalf("pages are not lexically ordered: %v", all)
	}

	nested, err := listMembers(runtimeNode, id, "nested", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := memberKeys(nested.Members), []string{"nested/child.json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected nested page: got %v want %v", got, want)
	}
	empty, err := listMembers(runtimeNode, id, "empty", "", 10)
	if err != nil || empty.Revision == 0 || len(empty.Members) != 0 {
		t.Fatalf("unexpected empty page %#v: %v", empty, err)
	}

	member, err := getMember(runtimeNode, id, "nested/child.json")
	if err != nil {
		t.Fatal(err)
	}
	if member.Kind != "file" || !reflect.DeepEqual(member.Capabilities, []protocol.CapabilityID{protocol.CapabilityGet, protocol.CapabilityRead}) {
		t.Fatalf("unexpected file member: %#v", member)
	}
	directory, err := getMember(runtimeNode, id, "nested")
	if err != nil {
		t.Fatal(err)
	}
	if directory.Kind != "directory" || !reflect.DeepEqual(directory.Capabilities, []protocol.CapabilityID{protocol.CapabilityGet, protocol.CapabilityList}) {
		t.Fatalf("unexpected directory member: %#v", directory)
	}

	if _, err := listMembers(runtimeNode, id, "", "not-a-cursor", 2); !errors.Is(err, ErrStaleCursor) {
		t.Fatalf("invalid cursor error = %v", err)
	}
	mustWrite(t, filepath.Join(root, "changed.txt"), []byte("changed"))
	if _, err := listMembers(runtimeNode, id, "", first.NextCursor, 2); !errors.Is(err, ErrStaleCursor) {
		t.Fatalf("stale cursor error = %v", err)
	}
}

func TestCollectionRevisionIsStableMonotonicAndJSONSafe(t *testing.T) {
	p := &provider{}
	membersA := []protocol.CollectionMemberV1{{
		Key: "a.txt", Kind: "file", Attributes: map[string]string{"revision": "content-a"},
	}}
	membersB := []protocol.CollectionMemberV1{{
		Key: "a.txt", Kind: "file", Attributes: map[string]string{"revision": "content-b"},
	}}

	first, firstFingerprint, err := p.collectionRevision("", membersA)
	if err != nil {
		t.Fatal(err)
	}
	stable, stableFingerprint, err := p.collectionRevision("", membersA)
	if err != nil {
		t.Fatal(err)
	}
	changed, changedFingerprint, err := p.collectionRevision("", membersB)
	if err != nil {
		t.Fatal(err)
	}
	restored, restoredFingerprint, err := p.collectionRevision("", membersA)
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || stable != first || changed != first+1 || restored != changed+1 {
		t.Fatalf("unexpected revision sequence: first=%d stable=%d changed=%d restored=%d", first, stable, changed, restored)
	}
	if restored > protocol.MaxCollectionRevision {
		t.Fatalf("revision is not JSON safe: %d", restored)
	}
	if firstFingerprint != stableFingerprint || firstFingerprint == changedFingerprint || firstFingerprint != restoredFingerprint {
		t.Fatalf("fingerprints do not track exact snapshots: first=%q stable=%q changed=%q restored=%q", firstFingerprint, stableFingerprint, changedFingerprint, restoredFingerprint)
	}

	const readers = 32
	var wg sync.WaitGroup
	errors := make(chan error, readers)
	for range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			revision, _, err := p.collectionRevision("other", membersA)
			if err != nil {
				errors <- err
				return
			}
			if revision != 1 {
				errors <- fmt.Errorf("concurrent stable revision = %d", revision)
			}
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

func TestCollectionRevisionTrackingIsBoundedAndEvictedCursorsStaySafe(t *testing.T) {
	membersA := []protocol.CollectionMemberV1{{
		Key: "a.txt", Kind: "file", Attributes: map[string]string{"revision": "content-a"},
	}}
	membersB := []protocol.CollectionMemberV1{{
		Key: "a.txt", Kind: "file", Attributes: map[string]string{"revision": "content-b"},
	}}
	prime := func(t *testing.T, p *provider) (uint64, string, string) {
		t.Helper()
		revision, fingerprint, err := p.collectionRevision("", membersA)
		if err != nil {
			t.Fatal(err)
		}
		cursor, err := encodeCursor(listCursorV1{
			Version: 1, After: membersA[0].Key, Revision: revision, Fingerprint: fingerprint,
		})
		if err != nil {
			t.Fatal(err)
		}
		return revision, fingerprint, cursor
	}
	fill := func(t *testing.T, p *provider, count int) {
		t.Helper()
		for index := range count {
			if _, _, err := p.collectionRevision(fmt.Sprintf("parent-%04d", index), nil); err != nil {
				t.Fatal(err)
			}
		}
	}

	t.Run("least-recent-parent-is-evicted", func(t *testing.T) {
		p := &provider{}
		oldRevision, oldFingerprint, cursor := prime(t, p)
		fill(t, p, MaxTrackedCollectionParents-1)
		if _, _, err := p.collectionRevision("", membersA); err != nil {
			t.Fatal(err)
		}
		if _, _, err := p.collectionRevision("overflow", nil); err != nil {
			t.Fatal(err)
		}
		if len(p.revisions) != MaxTrackedCollectionParents || p.revisionOrder.Len() != MaxTrackedCollectionParents {
			t.Fatalf("tracked parents grew past the cap: map=%d order=%d", len(p.revisions), p.revisionOrder.Len())
		}
		if _, exists := p.revisions["parent-0000"]; exists {
			t.Fatal("least-recently-used parent was not evicted")
		}
		if state, exists := p.revisions[""]; !exists || state.revision != oldRevision || state.fingerprint != collectionFingerprint("", membersA) {
			t.Fatalf("recently used parent was not retained: %#v", state)
		}
		request := protocol.CollectionListRequestV1{Version: 1, Cursor: cursor, Limit: 1}
		if _, err := pageStart(request, membersA, oldRevision, oldFingerprint); err != nil {
			t.Fatalf("cursor for an identical retained snapshot became stale: %v", err)
		}
	})

	t.Run("evicted-parent-resets-with-full-fingerprint", func(t *testing.T) {
		p := &provider{}
		oldRevision, _, cursor := prime(t, p)
		fill(t, p, MaxTrackedCollectionParents)
		if _, exists := p.revisions[""]; exists {
			t.Fatal("oldest parent was not evicted at the tracking cap")
		}
		newRevision, newFingerprint, err := p.collectionRevision("", membersB)
		if err != nil {
			t.Fatal(err)
		}
		if newRevision != oldRevision {
			t.Fatalf("evicted revision did not reset deterministically: old=%d new=%d", oldRevision, newRevision)
		}
		request := protocol.CollectionListRequestV1{Version: 1, Cursor: cursor, Limit: 1}
		if _, err := pageStart(request, membersB, newRevision, newFingerprint); !errors.Is(err, ErrStaleCursor) {
			t.Fatalf("cursor with a reused numeric revision accepted a changed snapshot: %v", err)
		}
		if len(p.revisions) != MaxTrackedCollectionParents {
			t.Fatalf("tracking map grew after reinsertion: %d", len(p.revisions))
		}
	})
}

func TestListIsBoundedAndLocatorsFailClosed(t *testing.T) {
	runtimeNode := newTestNode(t)
	root := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		mustWrite(t, filepath.Join(root, name), []byte(name))
	}
	id := registerOne(t, runtimeNode, Mount{Name: "bounded", Root: root, MaxScanEntries: 2})
	if _, err := listMembers(runtimeNode, id, "", "", 2); !errors.Is(err, ErrDirectoryLimit) {
		t.Fatalf("directory limit error = %v", err)
	}

	if _, err := getMember(runtimeNode, id, ""); err == nil {
		t.Fatal("empty locator was accepted")
	}
	for _, key := range []string{".", "..", "../a", "a/../b", "/absolute", `C:/absolute`, `a\b`, "a//b", "a/./b", "a\x00b"} {
		t.Run(strings.ReplaceAll(key, "/", "_"), func(t *testing.T) {
			if _, err := getMember(runtimeNode, id, key); !errors.Is(err, ErrUnsafeLocator) {
				t.Fatalf("locator %q error = %v", key, err)
			}
		})
	}
}

func TestReadContentTypesEncodingsLimitsAndConcurrency(t *testing.T) {
	runtimeNode := newTestNode(t)
	root := t.TempDir()
	files := map[string][]byte{
		"note.txt":    []byte("hello"),
		"data.json":   []byte(`{"ok":true}`),
		"page.html":   []byte("<!doctype html><title>safe data</title>"),
		"image.svg":   []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		"pixel.png":   {0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0},
		"invalid.txt": {0xff, 0xfe, 0x00},
		"binary.bin":  {0x00, 0xff, 0x01},
		"empty.txt":   {},
		"large.txt":   bytes.Repeat([]byte("x"), 65),
	}
	for name, data := range files {
		mustWrite(t, filepath.Join(root, name), data)
	}
	id := registerOne(t, runtimeNode, Mount{Name: "files", Root: root, MaxReadBytes: 64})

	tests := []struct {
		name        string
		contentType string
		encoding    string
	}{
		{"note.txt", "text/plain", protocol.FilesystemEncodingUTF8},
		{"data.json", "application/json", protocol.FilesystemEncodingUTF8},
		{"page.html", "text/html", protocol.FilesystemEncodingUTF8},
		{"image.svg", "image/svg+xml", protocol.FilesystemEncodingUTF8},
		{"pixel.png", "image/png", protocol.FilesystemEncodingBase64},
		{"invalid.txt", "text/plain", protocol.FilesystemEncodingBase64},
		{"binary.bin", "application/octet-stream", protocol.FilesystemEncodingBase64},
		{"empty.txt", "text/plain", protocol.FilesystemEncodingUTF8},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := readMember(runtimeNode, id, protocol.FilesystemReadRequestV1{Version: 1, Key: test.name, MaxBytes: 64})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(value.ContentType, test.contentType) || value.Encoding != test.encoding || value.Size != int64(len(files[test.name])) {
				t.Fatalf("unexpected content: %#v", value)
			}
			if got := decodedContent(t, value); !bytes.Equal(got, files[test.name]) {
				t.Fatalf("content mismatch: got %v want %v", got, files[test.name])
			}
		})
	}
	if _, err := readMember(runtimeNode, id, protocol.FilesystemReadRequestV1{Version: 1, Key: "large.txt", MaxBytes: 64}); !errors.Is(err, ErrReadLimit) {
		t.Fatalf("oversize read error = %v", err)
	}
	if _, err := readMember(runtimeNode, id, protocol.FilesystemReadRequestV1{Version: 1, Key: "note.txt", MaxBytes: 4}); !errors.Is(err, ErrReadLimit) {
		t.Fatalf("request limit error = %v", err)
	}
	if _, err := readMember(runtimeNode, id, protocol.FilesystemReadRequestV1{Version: 1, Key: "empty.txt", MaxBytes: 64, ExpectedRevision: "stale"}); !errors.Is(err, resource.ErrRevisionConflict) {
		t.Fatalf("stale read error = %v", err)
	}
	if _, err := readMember(runtimeNode, id, protocol.FilesystemReadRequestV1{Version: 1, Key: "not-there", MaxBytes: 64}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing read error = %v", err)
	}

	var group sync.WaitGroup
	errorsFound := make(chan error, 32)
	for index := 0; index < cap(errorsFound); index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			value, err := readMemberResult(runtimeNode, id, protocol.FilesystemReadRequestV1{Version: 1, Key: "note.txt", MaxBytes: 64})
			if err != nil {
				errorsFound <- err
				return
			}
			if value.Encoding != protocol.FilesystemEncodingUTF8 || value.Data != "hello" {
				errorsFound <- errors.New("concurrent read returned unexpected content")
			}
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
}

func TestSymlinkEscapeAndRootReplacementFailClosed(t *testing.T) {
	t.Run("symlink-escape", func(t *testing.T) {
		runtimeNode := newTestNode(t)
		base := t.TempDir()
		root := filepath.Join(base, "root")
		outside := filepath.Join(base, "outside")
		mustMkdir(t, root)
		mustMkdir(t, outside)
		mustWrite(t, filepath.Join(outside, "secret.txt"), []byte("secret"))
		if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "escape.txt")); err != nil {
			t.Skipf("symlink creation is unavailable: %v", err)
		}
		id := registerOne(t, runtimeNode, Mount{Name: "files", Root: root})
		page, err := listMembers(runtimeNode, id, "", "", 10)
		if err != nil {
			t.Fatal(err)
		}
		if got := memberKeys(page.Members); len(got) != 0 {
			t.Fatalf("escaping symlink was listed: %v", got)
		}
		if _, err := getMember(runtimeNode, id, "escape.txt"); !errors.Is(err, ErrUnsafeLocator) {
			t.Fatalf("escaping symlink get error = %v", err)
		}
		if _, err := readMember(runtimeNode, id, protocol.FilesystemReadRequestV1{Version: 1, Key: "escape.txt", MaxBytes: 64}); !errors.Is(err, ErrUnsafeLocator) {
			t.Fatalf("escaping symlink read error = %v", err)
		}
	})

	t.Run("root-replacement", func(t *testing.T) {
		runtimeNode := newTestNode(t)
		base := t.TempDir()
		root := filepath.Join(base, "root")
		backup := filepath.Join(base, "backup")
		mustMkdir(t, root)
		id := registerOne(t, runtimeNode, Mount{Name: "files", Root: root})
		if err := os.Rename(root, backup); err != nil {
			t.Skipf("root rename is unavailable: %v", err)
		}
		mustMkdir(t, root)
		if _, err := listMembers(runtimeNode, id, "", "", 10); !errors.Is(err, ErrRootChanged) {
			t.Fatalf("replaced root error = %v", err)
		}
	})
}

func TestFilesystemCollectionsUseExactResourceAndCapabilityAuthorization(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rootIdentity, err := auth.GenerateIdentity(1)
	if err != nil {
		t.Fatal(err)
	}
	childIdentity, err := auth.GenerateIdentity(2)
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	if err := trust.Add(rootIdentity.NodeID, rootIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	if err := trust.Add(childIdentity.NodeID, childIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	listID := protocol.ResourceID{Owner: rootIdentity.NodeID, Name: "list-only"}
	readID := protocol.ResourceID{Owner: rootIdentity.NodeID, Name: "read-only"}
	policy := auth.NewStaticPolicy()
	policy.Allow(auth.Request{Subject: childIdentity.NodeID, Action: auth.Action(protocol.CapabilityList), Resource: listID})
	policy.Allow(auth.Request{Subject: childIdentity.NodeID, Action: auth.Action(protocol.CapabilityRead), Resource: readID})
	rootNode, err := node.New(ctx, node.Config{Identity: rootIdentity, Trust: trust, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	defer rootNode.Close()
	listRoot, readRoot := filepath.Join(t.TempDir(), "list"), filepath.Join(t.TempDir(), "read")
	mustMkdir(t, listRoot)
	mustMkdir(t, readRoot)
	mustWrite(t, filepath.Join(listRoot, "listed.txt"), []byte("listed"))
	mustWrite(t, filepath.Join(readRoot, "read.txt"), []byte("read"))
	registration, err := Register(rootNode, []Mount{{Name: listID.Name, Root: listRoot}, {Name: readID.Name, Root: readRoot}})
	if err != nil {
		t.Fatal(err)
	}
	defer registration.Close()

	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := rootNode.Listen(network, "root")
	if err != nil {
		t.Fatal(err)
	}
	childTrust := auth.NewTrustStore()
	if err := childTrust.Add(rootIdentity.NodeID, rootIdentity.PublicKey); err != nil {
		t.Fatal(err)
	}
	childNode, err := node.New(ctx, node.Config{Identity: childIdentity, Trust: childTrust})
	if err != nil {
		t.Fatal(err)
	}
	defer childNode.Close()
	if err := childNode.ConnectParent(ctx, network, endpoint, rootNode.ID()); err != nil {
		t.Fatal(err)
	}

	listRequest := protocol.CollectionListRequestV1{Version: 1, Limit: 10}
	if _, err := operateRemote(ctx, childNode, listID, protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, listRequest); err != nil {
		t.Fatalf("authorized list failed: %v", err)
	}
	memberRequest := protocol.CollectionMemberRequestV1{Version: 1, Key: "listed.txt"}
	if _, err := operateRemote(ctx, childNode, listID, protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, memberRequest); !errors.Is(err, auth.ErrForbidden) && !strings.Contains(errorText(err), auth.ErrForbidden.Error()) {
		t.Fatalf("list grant authorized get: %v", err)
	}
	readRequest := protocol.FilesystemReadRequestV1{Version: 1, Key: "listed.txt", MaxBytes: 64}
	if _, err := operateRemote(ctx, childNode, listID, protocol.CapabilityRead, protocol.SchemaFilesystemReadRequestV1, readRequest); !errors.Is(err, auth.ErrForbidden) && !strings.Contains(errorText(err), auth.ErrForbidden.Error()) {
		t.Fatalf("list grant authorized read: %v", err)
	}
	if _, err := operateRemote(ctx, childNode, readID, protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, listRequest); !errors.Is(err, auth.ErrForbidden) && !strings.Contains(errorText(err), auth.ErrForbidden.Error()) {
		t.Fatalf("read grant authorized list: %v", err)
	}
	readRequest.Key = "read.txt"
	if _, err := operateRemote(ctx, childNode, readID, protocol.CapabilityRead, protocol.SchemaFilesystemReadRequestV1, readRequest); err != nil {
		t.Fatalf("authorized read failed: %v", err)
	}
}

func newTestNode(t *testing.T) *node.Node {
	t.Helper()
	identity, err := auth.GenerateIdentity(1)
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	if err := trust.Add(identity.NodeID, identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	runtimeNode, err := node.New(context.Background(), node.Config{Identity: identity, Trust: trust, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtimeNode.Close() })
	return runtimeNode
}

func registerOne(t *testing.T, runtimeNode *node.Node, mount Mount) protocol.ResourceID {
	t.Helper()
	registration, err := Register(runtimeNode, []Mount{mount})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = registration.Close() })
	return protocol.ResourceID{Owner: runtimeNode.ID(), Name: mount.Name}
}

func assertRegisterLeavesNone(t *testing.T, runtimeNode *node.Node, mounts []Mount, names ...string) {
	t.Helper()
	if _, err := Register(runtimeNode, mounts); err == nil {
		t.Fatal("expected registration failure")
	}
	for _, name := range names {
		if _, ok := runtimeNode.Registry().Resolve(protocol.ResourceID{Owner: runtimeNode.ID(), Name: name}); ok {
			t.Fatalf("resource %q was registered after failed validation", name)
		}
	}
}

func listMembers(runtimeNode *node.Node, id protocol.ResourceID, parent, cursor string, limit int) (protocol.CollectionPageV1, error) {
	input := protocol.CollectionListRequestV1{Version: 1, Parent: parent, Cursor: cursor, Limit: limit}
	result, err := operate(runtimeNode, id, protocol.CapabilityList, protocol.SchemaCollectionListRequestV1, input)
	if err != nil {
		return protocol.CollectionPageV1{}, err
	}
	var value protocol.CollectionPageV1
	if err := protocol.DecodeJSONPayload(result.Payload, protocol.DefaultMaxPayload, &value); err != nil {
		return protocol.CollectionPageV1{}, err
	}
	return value, nil
}

func getMember(runtimeNode *node.Node, id protocol.ResourceID, key string) (protocol.CollectionMemberV1, error) {
	input := protocol.CollectionMemberRequestV1{Version: 1, Key: key}
	result, err := operate(runtimeNode, id, protocol.CapabilityGet, protocol.SchemaCollectionMemberRequestV1, input)
	if err != nil {
		return protocol.CollectionMemberV1{}, err
	}
	var value protocol.CollectionMemberV1
	if err := protocol.DecodeJSONPayload(result.Payload, protocol.DefaultMaxPayload, &value); err != nil {
		return protocol.CollectionMemberV1{}, err
	}
	return value, nil
}

func readMember(runtimeNode *node.Node, id protocol.ResourceID, input protocol.FilesystemReadRequestV1) (protocol.FilesystemContentV1, error) {
	return readMemberResult(runtimeNode, id, input)
}

func readMemberResult(runtimeNode *node.Node, id protocol.ResourceID, input protocol.FilesystemReadRequestV1) (protocol.FilesystemContentV1, error) {
	result, err := operate(runtimeNode, id, protocol.CapabilityRead, protocol.SchemaFilesystemReadRequestV1, input)
	if err != nil {
		return protocol.FilesystemContentV1{}, err
	}
	var value protocol.FilesystemContentV1
	if err := protocol.DecodeJSONPayload(result.Payload, protocol.DefaultMaxPayload, &value); err != nil {
		return protocol.FilesystemContentV1{}, err
	}
	return value, nil
}

func operate(runtimeNode *node.Node, id protocol.ResourceID, capability protocol.CapabilityID, schema string, input protocol.ValidatedPayload) (resource.OperationResult, error) {
	payload, err := protocol.EncodeJSONPayload(input, protocol.DefaultMaxPayload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	return runtimeNode.Registry().Operate(context.Background(), id, resource.OperationRequest{Capability: capability, Schema: schema, Payload: payload})
}

func operateRemote(ctx context.Context, runtimeNode *node.Node, id protocol.ResourceID, capability protocol.CapabilityID, schema string, input protocol.ValidatedPayload) (resource.OperationResult, error) {
	payload, err := protocol.EncodeJSONPayload(input, protocol.DefaultMaxPayload)
	if err != nil {
		return resource.OperationResult{}, err
	}
	return runtimeNode.Operate(ctx, id, capability, schema, payload)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func decodedContent(t *testing.T, value protocol.FilesystemContentV1) []byte {
	t.Helper()
	if value.Encoding == protocol.FilesystemEncodingUTF8 {
		return []byte(value.Data)
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(value.Data)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func memberKeys(members []protocol.CollectionMemberV1) []string {
	keys := make([]string, 0, len(members))
	for _, member := range members {
		keys = append(keys, member.Key)
	}
	return keys
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
