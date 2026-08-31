package filesystem

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

type provider struct {
	id             protocol.ResourceID
	originalRoot   string
	rootKey        string
	rootIdentity   rootIdentity
	label          string
	maxReadBytes   int
	maxScanEntries int
	resource       *resource.HandlerResource
	revisionMu     sync.Mutex
	revisions      map[string]collectionRevisionState
	revisionOrder  *list.List
}

type collectionRevisionState struct {
	fingerprint [sha256.Size]byte
	revision    uint64
	order       *list.Element
}

type listCursorV1 struct {
	Version     int    `json:"version"`
	Parent      string `json:"parent,omitempty"`
	After       string `json:"after"`
	Revision    uint64 `json:"revision"`
	Fingerprint string `json:"fingerprint"`
}

func (p *provider) descriptor() resource.Descriptor {
	descriptor := protocol.ResourceDescriptorV2{
		ID: p.id, Type: protocol.ResourceTypeCollection, TypeVersion: 1,
		Capabilities: []protocol.CapabilityDescriptorV2{
			{Name: protocol.CapabilityGet, Permission: "filesystem.get", InputSchema: protocol.SchemaCollectionMemberRequestV1, OutputSchema: protocol.SchemaCollectionMemberV1, MaxPayloadBytes: protocol.DefaultMaxPayload},
			{Name: protocol.CapabilityList, Permission: "filesystem.list", InputSchema: protocol.SchemaCollectionListRequestV1, OutputSchema: protocol.SchemaCollectionPageV1, MaxPayloadBytes: protocol.DefaultMaxPayload},
			{Name: protocol.CapabilityRead, Permission: "filesystem.read", InputSchema: protocol.SchemaFilesystemReadRequestV1, OutputSchema: protocol.SchemaFilesystemContentV1, MaxPayloadBytes: protocol.DefaultMaxPayload},
		},
		Schemas: []protocol.SchemaDescriptorV2{
			{ID: protocol.SchemaCollectionListRequestV1, ContentType: "application/json"},
			{ID: protocol.SchemaCollectionMemberRequestV1, ContentType: "application/json"},
			{ID: protocol.SchemaCollectionMemberV1, ContentType: "application/json"},
			{ID: protocol.SchemaCollectionPageV1, ContentType: "application/json"},
			{ID: protocol.SchemaFilesystemReadRequestV1, ContentType: "application/json"},
			{ID: protocol.SchemaFilesystemContentV1, ContentType: "application/json"},
		},
		Limits: protocol.ResourceLimitsV2{MaxPayloadBytes: protocol.DefaultMaxPayload},
		Presentation: protocol.PresentationHintV2{
			Renderer: string(protocol.ResourceTypeCollection), Label: p.label,
			Description: "Read-only filesystem collection", Icon: "folder",
		},
	}
	descriptor.Sort()
	return descriptor
}

func (p *provider) list(ctx context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	var input protocol.CollectionListRequestV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	if err := validateLocator(input.Parent, true); err != nil {
		return resource.OperationResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return resource.OperationResult{}, err
	}
	_, directory, info, err := p.resolve(input.Parent, true)
	if err != nil {
		return resource.OperationResult{}, err
	}
	if !info.IsDir() {
		return resource.OperationResult{}, errors.New("filesystem list target is not a directory")
	}
	opened, err := os.Open(directory)
	if err != nil {
		return resource.OperationResult{}, ErrRootUnavailable
	}
	defer opened.Close()
	openedInfo, err := opened.Stat()
	if err != nil || !os.SameFile(info, openedInfo) || !openedInfo.IsDir() {
		return resource.OperationResult{}, ErrRootChanged
	}
	entries, err := opened.ReadDir(p.maxScanEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return resource.OperationResult{}, errors.New("filesystem directory cannot be read")
	}
	if len(entries) > p.maxScanEntries {
		return resource.OperationResult{}, ErrDirectoryLimit
	}

	members := make([]protocol.CollectionMemberV1, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return resource.OperationResult{}, err
		}
		key := entry.Name()
		if input.Parent != "" {
			key = input.Parent + "/" + key
		}
		member, err := p.member(key)
		if errors.Is(err, ErrUnsafeLocator) {
			continue
		}
		if err != nil {
			return resource.OperationResult{}, err
		}
		if member.Kind != "directory" && member.Kind != "file" {
			continue
		}
		members = append(members, member)
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Key < members[j].Key })
	revision, fingerprint, err := p.collectionRevision(input.Parent, members)
	if err != nil {
		return resource.OperationResult{}, err
	}
	start, err := pageStart(input, members, revision, fingerprint)
	if err != nil {
		return resource.OperationResult{}, err
	}
	end := start + input.Limit
	if end > len(members) {
		end = len(members)
	}
	pageMembers := append([]protocol.CollectionMemberV1(nil), members[start:end]...)
	next := ""
	if end < len(members) {
		next, err = encodeCursor(listCursorV1{Version: 1, Parent: input.Parent, After: members[end-1].Key, Revision: revision, Fingerprint: fingerprint})
		if err != nil {
			return resource.OperationResult{}, err
		}
	}
	page := protocol.CollectionPageV1{Version: 1, Revision: revision, Parent: input.Parent, Members: pageMembers, NextCursor: next}
	if err := page.ValidateForDescriptor(p.resource.Descriptor()); err != nil {
		return resource.OperationResult{}, err
	}
	payload, err := protocol.EncodeJSONPayload(&page, protocol.DefaultMaxPayload)
	return resource.OperationResult{Schema: protocol.SchemaCollectionPageV1, Payload: payload}, err
}

func (p *provider) get(ctx context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	var input protocol.CollectionMemberRequestV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return resource.OperationResult{}, err
	}
	member, err := p.member(input.Key)
	if err != nil {
		return resource.OperationResult{}, err
	}
	payload, err := protocol.EncodeJSONPayload(&member, protocol.DefaultMaxPayload)
	return resource.OperationResult{Schema: protocol.SchemaCollectionMemberV1, Payload: payload}, err
}

func (p *provider) read(ctx context.Context, request resource.OperationRequest) (resource.OperationResult, error) {
	var input protocol.FilesystemReadRequestV1
	if err := protocol.DecodeJSONPayload(request.Payload, protocol.DefaultMaxPayload, &input); err != nil {
		return resource.OperationResult{}, err
	}
	if err := validateLocator(input.Key, false); err != nil {
		return resource.OperationResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return resource.OperationResult{}, err
	}
	_, target, before, err := p.resolve(input.Key, false)
	if err != nil {
		return resource.OperationResult{}, err
	}
	if !before.Mode().IsRegular() {
		return resource.OperationResult{}, errors.New("filesystem read target is not a regular file")
	}
	revision := memberRevision(input.Key, before)
	if input.ExpectedRevision != "" && input.ExpectedRevision != revision {
		return resource.OperationResult{}, resource.ErrRevisionConflict
	}
	limit := input.MaxBytes
	if p.maxReadBytes < limit {
		limit = p.maxReadBytes
	}
	if before.Size() > int64(limit) {
		return resource.OperationResult{}, ErrReadLimit
	}
	opened, err := os.Open(target)
	if err != nil {
		return resource.OperationResult{}, errors.New("filesystem member cannot be opened")
	}
	defer opened.Close()
	openedInfo, err := opened.Stat()
	if err != nil || !os.SameFile(before, openedInfo) || !openedInfo.Mode().IsRegular() {
		return resource.OperationResult{}, ErrRootChanged
	}
	data, err := io.ReadAll(io.LimitReader(opened, int64(limit)+1))
	if err != nil {
		return resource.OperationResult{}, errors.New("filesystem member cannot be read")
	}
	if len(data) > limit {
		return resource.OperationResult{}, ErrReadLimit
	}
	after, err := opened.Stat()
	if err != nil || !os.SameFile(before, after) || after.Size() != int64(len(data)) || memberRevision(input.Key, after) != revision {
		return resource.OperationResult{}, resource.ErrRevisionConflict
	}
	if err := ctx.Err(); err != nil {
		return resource.OperationResult{}, err
	}
	contentType := detectContentType(input.Key, data)
	encoding := protocol.FilesystemEncodingBase64
	content := base64.StdEncoding.EncodeToString(data)
	if isTextContentType(contentType) && utf8.Valid(data) {
		encoding = protocol.FilesystemEncodingUTF8
		content = string(data)
	}
	modified := after.ModTime().UnixMilli()
	if modified < 0 {
		modified = 0
	}
	value := protocol.FilesystemContentV1{
		Version: 1, Key: input.Key, ContentType: contentType, Encoding: encoding, Data: content,
		Size: int64(len(data)), ModifiedUnixMS: modified, Revision: revision,
	}
	payload, err := protocol.EncodeJSONPayload(&value, protocol.DefaultMaxPayload)
	return resource.OperationResult{Schema: protocol.SchemaFilesystemContentV1, Payload: payload}, err
}

func (p *provider) member(key string) (protocol.CollectionMemberV1, error) {
	if err := validateLocator(key, false); err != nil {
		return protocol.CollectionMemberV1{}, err
	}
	_, _, info, err := p.resolve(key, false)
	if err != nil {
		return protocol.CollectionMemberV1{}, err
	}
	modified := info.ModTime().UnixMilli()
	if modified < 0 {
		modified = 0
	}
	revision := memberRevision(key, info)
	member := protocol.CollectionMemberV1{
		Key: key, Label: filepath.Base(key),
		Attributes: map[string]string{
			"modified_unix_ms": strconv.FormatInt(modified, 10),
			"revision":         revision,
			"size":             strconv.FormatInt(info.Size(), 10),
		},
	}
	switch {
	case info.IsDir():
		member.Kind = "directory"
		member.ContentType = "application/json"
		member.Schema = protocol.SchemaCollectionPageV1
		member.Capabilities = []protocol.CapabilityID{protocol.CapabilityGet, protocol.CapabilityList}
	case info.Mode().IsRegular():
		member.Kind = "file"
		member.ContentType = metadataContentType(key)
		member.Schema = protocol.SchemaFilesystemContentV1
		member.Capabilities = []protocol.CapabilityID{protocol.CapabilityGet, protocol.CapabilityRead}
	default:
		member.Kind = "unsupported"
		member.Label = filepath.Base(key)
	}
	if err := member.Validate(); err != nil {
		return protocol.CollectionMemberV1{}, err
	}
	return member, nil
}

func (p *provider) currentRoot() (string, error) {
	evaluated, err := filepath.EvalSymlinks(p.originalRoot)
	if err != nil {
		return "", ErrRootUnavailable
	}
	evaluated, err = filepath.Abs(evaluated)
	if err != nil {
		return "", ErrRootUnavailable
	}
	evaluated = filepath.Clean(evaluated)
	if pathComparisonKey(evaluated) != p.rootKey {
		return "", ErrRootChanged
	}
	info, err := os.Stat(evaluated)
	if err != nil || !info.IsDir() || !p.rootIdentity.matches(info) {
		return "", ErrRootChanged
	}
	return evaluated, nil
}

func (p *provider) resolve(locator string, allowRoot bool) (string, string, os.FileInfo, error) {
	if err := validateLocator(locator, allowRoot); err != nil {
		return "", "", nil, err
	}
	root, err := p.currentRoot()
	if err != nil {
		return "", "", nil, err
	}
	target := root
	if locator != "" {
		target = filepath.Join(root, filepath.FromSlash(locator))
	}
	canonical, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", "", nil, resource.ErrNotFound
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil || !containsPath(root, canonical) {
		return "", "", nil, ErrUnsafeLocator
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", "", nil, resource.ErrNotFound
	}
	return root, filepath.Clean(canonical), info, nil
}

func validateLocator(value string, allowEmpty bool) error {
	if value == "" && allowEmpty {
		return nil
	}
	if value == "" || len(value) > protocol.MaxCollectionMemberKeyBytes || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return ErrUnsafeLocator
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "\\") || strings.Contains(value, "\\") || filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return ErrUnsafeLocator
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.Contains(segment, ":") {
			return ErrUnsafeLocator
		}
	}
	return nil
}

func containsPath(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil || filepath.IsAbs(relative) {
		return false
	}
	if relative == "." {
		return true
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func collectionFingerprint(parent string, members []protocol.CollectionMemberV1) [sha256.Size]byte {
	hash := sha256.New()
	_, _ = io.WriteString(hash, parent)
	for _, member := range members {
		_, _ = io.WriteString(hash, "\x00"+member.Key+"\x00"+member.Kind+"\x00"+member.Attributes["revision"])
	}
	var fingerprint [sha256.Size]byte
	copy(fingerprint[:], hash.Sum(nil))
	return fingerprint
}

func (p *provider) collectionRevision(parent string, members []protocol.CollectionMemberV1) (uint64, string, error) {
	fingerprint := collectionFingerprint(parent, members)
	p.revisionMu.Lock()
	defer p.revisionMu.Unlock()
	if p.revisions == nil {
		p.revisions = make(map[string]collectionRevisionState)
		p.revisionOrder = list.New()
	}
	state, exists := p.revisions[parent]
	if exists {
		p.revisionOrder.MoveToBack(state.order)
	}
	if exists && state.fingerprint == fingerprint {
		return state.revision, hex.EncodeToString(fingerprint[:]), nil
	}
	if exists {
		if state.revision >= protocol.MaxCollectionRevision {
			return 0, "", errors.New("filesystem collection revision exhausted")
		}
		state.revision++
	} else {
		if len(p.revisions) == MaxTrackedCollectionParents {
			oldest := p.revisionOrder.Front()
			delete(p.revisions, oldest.Value.(string))
			p.revisionOrder.Remove(oldest)
		}
		state.revision = 1
		state.order = p.revisionOrder.PushBack(parent)
	}
	state.fingerprint = fingerprint
	p.revisions[parent] = state
	return state.revision, hex.EncodeToString(fingerprint[:]), nil
}

func memberRevision(key string, info os.FileInfo) string {
	hash := sha256.New()
	_, _ = io.WriteString(hash, key)
	_, _ = io.WriteString(hash, "\x00"+strconv.FormatInt(info.Size(), 10))
	_, _ = io.WriteString(hash, "\x00"+strconv.FormatInt(info.ModTime().UnixNano(), 10))
	_, _ = io.WriteString(hash, "\x00"+info.Mode().String())
	return hex.EncodeToString(hash.Sum(nil))
}

func encodeCursor(value listCursorV1) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	if len(encoded) > protocol.MaxCollectionCursorBytes {
		return "", ErrStaleCursor
	}
	return encoded, nil
}

func decodeCursor(value string) (listCursorV1, error) {
	payload, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil {
		return listCursorV1{}, ErrStaleCursor
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	var cursor listCursorV1
	if err := decoder.Decode(&cursor); err != nil || cursor.Version != 1 || cursor.After == "" || cursor.Revision == 0 || cursor.Revision > protocol.MaxCollectionRevision || len(cursor.Fingerprint) != sha256.Size*2 {
		return listCursorV1{}, ErrStaleCursor
	}
	if _, err := hex.DecodeString(cursor.Fingerprint); err != nil {
		return listCursorV1{}, ErrStaleCursor
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return listCursorV1{}, ErrStaleCursor
	}
	return cursor, nil
}

func pageStart(request protocol.CollectionListRequestV1, members []protocol.CollectionMemberV1, revision uint64, fingerprint string) (int, error) {
	if request.Cursor == "" {
		return 0, nil
	}
	cursor, err := decodeCursor(request.Cursor)
	if err != nil || cursor.Parent != request.Parent || cursor.Revision != revision || cursor.Fingerprint != fingerprint {
		return 0, ErrStaleCursor
	}
	index := sort.Search(len(members), func(index int) bool { return members[index].Key >= cursor.After })
	if index >= len(members) || members[index].Key != cursor.After {
		return 0, ErrStaleCursor
	}
	return index + 1, nil
}

func detectContentType(name string, data []byte) string {
	sniffed := http.DetectContentType(data)
	extension := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if extension != "" && isTextContentType(extension) && (len(data) == 0 || sniffed == "application/octet-stream" || strings.HasPrefix(sniffed, "text/plain") || strings.Contains(sniffed, "xml")) {
		return extension
	}
	return sniffed
}

func metadataContentType(name string) string {
	if extension := mime.TypeByExtension(strings.ToLower(filepath.Ext(name))); extension != "" {
		return extension
	}
	return "application/octet-stream"
}

func isTextContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	return strings.HasPrefix(mediaType, "text/") || mediaType == "application/json" || mediaType == "application/xml" || mediaType == "application/javascript" || mediaType == "image/svg+xml"
}
