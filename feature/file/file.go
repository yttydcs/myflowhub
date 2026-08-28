package file

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

type Config struct {
	Node          *node.Node
	Root          string
	MaxFileBytes  int64
	MaxTotalBytes int64
	MaxActive     int
	MaxHistory    int
	MaxLifetime   time.Duration
	SweepInterval time.Duration
	Now           func() time.Time
}

type transfer struct {
	offer     protocol.FileOfferV1
	owner     protocol.NodeID
	tempPath  string
	finalPath string
	file      *os.File
	received  int64
	state     string
	err       string
}

type Controller struct {
	mu            sync.Mutex
	ctx           context.Context
	stop          context.CancelFunc
	wg            sync.WaitGroup
	node          *node.Node
	root          string
	tempRoot      string
	maxFileBytes  int64
	maxTotalBytes int64
	maxActive     int
	maxHistory    int
	maxLifetime   time.Duration
	sweepInterval time.Duration
	now           func() time.Time
	transfers     map[string]*transfer
	activeBytes   int64
	revision      uint64
	summaries     *resource.Variable
	progress      *resource.Stream
	upload        *uploadResource
}

func Register(config Config) (*Controller, error) {
	if config.Node == nil || config.Root == "" {
		return nil, errors.New("file node and storage root are required")
	}
	if config.MaxFileBytes <= 0 {
		config.MaxFileBytes = 1 << 30
	}
	if config.MaxTotalBytes <= 0 {
		config.MaxTotalBytes = 4 << 30
	}
	if config.MaxActive <= 0 {
		config.MaxActive = 64
	}
	if config.MaxHistory <= 0 {
		config.MaxHistory = 256
	}
	if config.MaxLifetime <= 0 {
		config.MaxLifetime = 24 * time.Hour
	}
	if config.SweepInterval <= 0 {
		config.SweepInterval = time.Minute
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.MaxFileBytes > config.MaxTotalBytes || config.MaxActive < 1 || config.MaxHistory < config.MaxActive || config.MaxHistory > protocol.MaxItems || config.MaxLifetime < time.Second || config.SweepInterval < 10*time.Millisecond {
		return nil, errors.New("file transfer limits are invalid")
	}
	root, err := filepath.Abs(config.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve file storage root: %w", err)
	}
	if err := ensureDirectory(root); err != nil {
		return nil, err
	}
	tempRoot := filepath.Join(root, ".mfh-tmp")
	if err := ensureDirectory(tempRoot); err != nil {
		return nil, err
	}
	if err := cleanupTemp(tempRoot); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	value := &Controller{
		ctx: ctx, stop: cancel,
		node: config.Node, root: root, tempRoot: tempRoot, maxFileBytes: config.MaxFileBytes, maxTotalBytes: config.MaxTotalBytes,
		maxActive: config.MaxActive, maxHistory: config.MaxHistory, maxLifetime: config.MaxLifetime, sweepInterval: config.SweepInterval, now: config.Now,
		transfers: make(map[string]*transfer), revision: 1,
	}
	initial, _ := protocol.EncodeJSONPayload(&protocol.FileTransfersV1{Version: 1, Revision: 1, Transfers: []protocol.FileTransferSummaryV1{}}, protocol.DefaultMaxPayload)
	value.summaries, err = resource.NewVariable(resource.VariableDescriptor(protocol.ResourceID{Owner: config.Node.ID(), Name: protocol.BuiltinFileTransfers}, "application/json", protocol.SchemaFileTransfersV1, "file.read", protocol.DefaultMaxPayload), initial)
	if err != nil {
		return nil, err
	}
	value.progress, err = resource.NewStream(resource.StreamDescriptor(protocol.ResourceID{Owner: config.Node.ID(), Name: protocol.BuiltinFileProgress}, "application/json", protocol.SchemaFileProgressV1, "file.read", protocol.DefaultMaxPayload))
	if err != nil {
		return nil, err
	}
	value.upload, err = newUploadResource(value)
	if err != nil {
		return nil, err
	}
	resources := []resource.Resource{value.summaries, value.progress, value.upload}
	registered := make([]protocol.ResourceID, 0, len(resources))
	for _, current := range resources {
		if err := config.Node.Registry().Register(current); err != nil {
			for index := len(registered) - 1; index >= 0; index-- {
				_ = config.Node.Registry().Remove(registered[index])
			}
			cancel()
			return nil, fmt.Errorf("register file resource %s: %w", current.Descriptor().ID.Name, err)
		}
		registered = append(registered, current.Descriptor().ID)
	}
	value.wg.Add(1)
	go value.sweepLoop()
	return value, nil
}

func (c *Controller) Close() error {
	if c == nil {
		return nil
	}
	c.stop()
	c.wg.Wait()
	c.mu.Lock()
	defer c.mu.Unlock()
	var result error
	for _, current := range c.transfers {
		if current.file != nil {
			if err := current.file.Close(); err != nil && result == nil {
				result = err
			}
			current.file = nil
		}
		if current.state == "offered" || current.state == "receiving" {
			if err := os.Remove(current.tempPath); err != nil && !errors.Is(err, os.ErrNotExist) && result == nil {
				result = err
			}
		}
	}
	return result
}

func (c *Controller) Sweep() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.expireLocked(c.now().UTC())
	c.mu.Unlock()
}

func (c *Controller) sweepLoop() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.Sweep()
		case <-c.node.Done():
			return
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Controller) offer(ctx context.Context, input []byte) ([]byte, error) {
	owner, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	var request protocol.FileOfferV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	return c.offerFor(owner, request)
}

func (c *Controller) offerFor(owner protocol.NodeID, request protocol.FileOfferV1) ([]byte, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	now := c.now().UTC()
	if request.Size > c.maxFileBytes || request.ExpiresAtUnixMS <= now.UnixMilli() || request.ExpiresAtUnixMS > now.Add(c.maxLifetime).UnixMilli() {
		return nil, errors.New("file offer exceeds size or lifetime limits")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.expireLocked(now)
	if current := c.transfers[request.TransferID]; current != nil {
		if current.owner == owner && current.offer == request {
			return c.progressPayloadLocked(current)
		}
		return nil, errors.New("file transfer ID already exists")
	}
	if c.activeCountLocked() >= c.maxActive || request.Size > c.maxTotalBytes-c.activeBytes {
		return nil, errors.New("file transfer active session or byte limit reached")
	}
	finalPath, err := safeDestination(c.root, request.Path)
	if err != nil {
		return nil, err
	}
	tempPath := filepath.Join(c.tempRoot, request.TransferID+".part")
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create transfer temporary file: %w", err)
	}
	current := &transfer{offer: request, owner: owner, tempPath: tempPath, finalPath: finalPath, file: file, state: "offered"}
	c.transfers[request.TransferID] = current
	c.activeBytes += request.Size
	c.trimLocked()
	if err := c.publishLocked(current); err != nil {
		delete(c.transfers, request.TransferID)
		c.activeBytes -= request.Size
		_ = file.Close()
		_ = os.Remove(tempPath)
		return nil, err
	}
	return c.progressPayloadLocked(current)
}

func (c *Controller) chunk(ctx context.Context, input []byte) ([]byte, error) {
	owner, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	var request protocol.FileChunkV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	return c.chunkFor(owner, request)
}

func (c *Controller) chunkFor(owner protocol.NodeID, request protocol.FileChunkV1) ([]byte, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(request.Data)
	if hex.EncodeToString(digest[:]) != request.SHA256 {
		return nil, errors.New("file chunk checksum mismatch")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.expireLocked(c.now().UTC())
	current, err := c.activeOwnedLocked(request.TransferID, owner)
	if err != nil {
		return nil, err
	}
	if len(request.Data) > current.offer.ChunkSize || request.Offset+int64(len(request.Data)) > current.offer.Size {
		return nil, errors.New("file chunk exceeds declared chunk or transfer size")
	}
	if request.Offset < current.received {
		if request.Offset+int64(len(request.Data)) > current.received {
			return nil, errors.New("file chunk overlaps the received boundary")
		}
		existing := make([]byte, len(request.Data))
		if _, err := current.file.ReadAt(existing, request.Offset); err != nil || !bytes.Equal(existing, request.Data) {
			return nil, errors.New("duplicate file chunk content conflicts with stored data")
		}
		return c.progressPayloadLocked(current)
	}
	if request.Offset != current.received {
		return nil, fmt.Errorf("file chunk gap: expected offset %d", current.received)
	}
	if _, err := current.file.WriteAt(request.Data, request.Offset); err != nil {
		c.failLocked(current, err)
		return nil, fmt.Errorf("write file chunk: %w", err)
	}
	current.received += int64(len(request.Data))
	current.state = "receiving"
	if err := c.publishLocked(current); err != nil {
		return nil, err
	}
	return c.progressPayloadLocked(current)
}

func (c *Controller) complete(ctx context.Context, input []byte) ([]byte, error) {
	owner, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	var request protocol.FileCompleteV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	return c.completeFor(owner, request)
}

func (c *Controller) completeFor(owner protocol.NodeID, request protocol.FileCompleteV1) ([]byte, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.expireLocked(c.now().UTC())
	current, err := c.activeOwnedLocked(request.TransferID, owner)
	if err != nil {
		return nil, err
	}
	if request.Size != current.offer.Size || request.SHA256 != current.offer.SHA256 || current.received != current.offer.Size {
		return nil, errors.New("file completion size, checksum, or received length mismatch")
	}
	if err := current.file.Sync(); err != nil {
		c.failLocked(current, err)
		return nil, err
	}
	if _, err := current.file.Seek(0, io.SeekStart); err != nil {
		c.failLocked(current, err)
		return nil, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(current.file, current.offer.Size+1)); err != nil {
		c.failLocked(current, err)
		return nil, err
	}
	if hex.EncodeToString(hash.Sum(nil)) != current.offer.SHA256 {
		err := errors.New("completed file checksum mismatch")
		c.failLocked(current, err)
		return nil, err
	}
	if err := current.file.Close(); err != nil {
		c.failLocked(current, err)
		return nil, err
	}
	current.file = nil
	if _, err := os.Lstat(current.finalPath); err == nil {
		c.failLocked(current, errors.New("destination already exists"))
		return nil, errors.New("file destination already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		c.failLocked(current, err)
		return nil, err
	}
	if err := os.Rename(current.tempPath, current.finalPath); err != nil {
		c.failLocked(current, err)
		return nil, fmt.Errorf("activate completed file: %w", err)
	}
	current.state = "completed"
	c.activeBytes -= current.offer.Size
	if err := c.publishLocked(current); err != nil {
		return nil, err
	}
	return c.progressPayloadLocked(current)
}

func (c *Controller) cancel(ctx context.Context, input []byte) ([]byte, error) {
	owner, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	var request protocol.FileCancelV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	return c.cancelFor(owner, request)
}

func (c *Controller) cancelFor(owner protocol.NodeID, request protocol.FileCancelV1) ([]byte, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.transfers[request.TransferID]
	if current == nil || current.owner != owner {
		return nil, errors.New("file transfer not found for caller")
	}
	if current.state == "cancelled" {
		return c.progressPayloadLocked(current)
	}
	if current.state == "completed" || current.state == "failed" {
		return nil, errors.New("terminal file transfer cannot be cancelled")
	}
	if current.file != nil {
		_ = current.file.Close()
		current.file = nil
	}
	_ = os.Remove(current.tempPath)
	current.state = "cancelled"
	current.err = request.Reason
	c.activeBytes -= current.offer.Size
	if err := c.publishLocked(current); err != nil {
		return nil, err
	}
	return c.progressPayloadLocked(current)
}

func (c *Controller) abortFor(owner protocol.NodeID, transferID string, reason error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.transfers[transferID]
	if current == nil || current.owner != owner || (current.state != "offered" && current.state != "receiving") {
		return
	}
	if reason == nil {
		reason = errors.New("upload session aborted")
	}
	c.failLocked(current, reason)
}

func (c *Controller) activeOwnedLocked(id string, owner protocol.NodeID) (*transfer, error) {
	current := c.transfers[id]
	if current == nil || current.owner != owner {
		return nil, errors.New("file transfer not found for caller")
	}
	if current.state != "offered" && current.state != "receiving" {
		return nil, errors.New("file transfer is terminal")
	}
	return current, nil
}

func (c *Controller) failLocked(current *transfer, failure error) {
	if current.state == "offered" || current.state == "receiving" {
		c.activeBytes -= current.offer.Size
	}
	if current.file != nil {
		_ = current.file.Close()
		current.file = nil
	}
	_ = os.Remove(current.tempPath)
	current.state = "failed"
	current.err = failure.Error()
	_ = c.publishLocked(current)
}

func (c *Controller) expireLocked(now time.Time) {
	changed := false
	for _, current := range c.transfers {
		if (current.state == "offered" || current.state == "receiving") && current.offer.ExpiresAtUnixMS <= now.UnixMilli() {
			c.activeBytes -= current.offer.Size
			if current.file != nil {
				_ = current.file.Close()
				current.file = nil
			}
			_ = os.Remove(current.tempPath)
			current.state = "failed"
			current.err = "transfer expired"
			payload, _ := c.progressPayloadLocked(current)
			_, _ = c.progress.Publish(payload)
			changed = true
		}
	}
	if changed {
		_ = c.updateSummariesLocked()
	}
}

func (c *Controller) publishLocked(current *transfer) error {
	payload, err := c.progressPayloadLocked(current)
	if err != nil {
		return err
	}
	if _, err := c.progress.Publish(payload); err != nil {
		return err
	}
	return c.updateSummariesLocked()
}

func (c *Controller) progressPayloadLocked(current *transfer) ([]byte, error) {
	return protocol.EncodeJSONPayload(&protocol.FileProgressV1{Version: 1, TransferID: current.offer.TransferID, ReceivedBytes: current.received, TotalBytes: current.offer.Size, State: current.state, Error: current.err}, protocol.DefaultMaxPayload)
}

func (c *Controller) updateSummariesLocked() error {
	if c.revision == ^uint64(0) {
		return errors.New("file transfer summary revision exhausted")
	}
	c.revision++
	ids := make([]string, 0, len(c.transfers))
	for id := range c.transfers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	summaries := make([]protocol.FileTransferSummaryV1, 0, len(ids))
	for _, id := range ids {
		current := c.transfers[id]
		summaries = append(summaries, protocol.FileTransferSummaryV1{TransferID: id, OwnerNodeID: strconv.FormatUint(uint64(current.owner), 10), Path: current.offer.Path, ReceivedBytes: current.received, TotalBytes: current.offer.Size, State: current.state, ExpiresAtUnixMS: current.offer.ExpiresAtUnixMS, Error: current.err})
	}
	payload, err := protocol.EncodeJSONPayload(&protocol.FileTransfersV1{Version: 1, Revision: c.revision, Transfers: summaries}, protocol.DefaultMaxPayload)
	if err != nil {
		return err
	}
	_, err = c.summaries.Set(payload)
	return err
}

func (c *Controller) activeCountLocked() int {
	count := 0
	for _, current := range c.transfers {
		if current.state == "offered" || current.state == "receiving" {
			count++
		}
	}
	return count
}

func (c *Controller) trimLocked() {
	if len(c.transfers) <= c.maxHistory {
		return
	}
	ids := make([]string, 0, len(c.transfers))
	for id, current := range c.transfers {
		if current.state != "offered" && current.state != "receiving" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for len(c.transfers) > c.maxHistory && len(ids) > 0 {
		delete(c.transfers, ids[0])
		ids = ids[1:]
	}
}

func caller(ctx context.Context) (protocol.NodeID, error) {
	delegation, ok := command.DelegationFromContext(ctx)
	if !ok {
		return 0, errors.New("file command requires an authenticated command context")
	}
	owner, ok := delegation.Subject()
	if !ok {
		return 0, errors.New("file command subject is unavailable")
	}
	return owner, nil
}

func ensureDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create file storage directory: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("file storage path must be a real directory")
	}
	return os.Chmod(path, 0o700)
}

func safeDestination(root, slashPath string) (string, error) {
	segments := strings.Split(slashPath, "/")
	current := root
	for _, segment := range segments[:len(segments)-1] {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil {
				return "", fmt.Errorf("create destination directory: %w", err)
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("file destination traverses a non-directory or symlink")
		}
	}
	destination := filepath.Join(current, segments[len(segments)-1])
	relative, err := filepath.Rel(root, destination)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", errors.New("file destination escapes storage root")
	}
	if info, err := os.Lstat(destination); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("file destination is a symlink")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return destination, nil
}

func cleanupTemp(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("scan transfer temporary directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) != 37 || !strings.HasSuffix(name, ".part") {
			continue
		}
		id := strings.TrimSuffix(name, ".part")
		decoded, err := hex.DecodeString(id)
		if err != nil || len(decoded) != 16 || id != strings.ToLower(id) {
			continue
		}
		if err := os.Remove(filepath.Join(root, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove stale transfer %s: %w", id, err)
		}
	}
	return nil
}
