package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

const (
	MaxMounts                   = 64
	DefaultScanEntries          = 1024
	MaxScanEntries              = protocol.MaxItems
	MaxTrackedCollectionParents = 1024
)

var (
	ErrUnsafeLocator   = errors.New("filesystem member locator is unsafe")
	ErrRootUnavailable = errors.New("filesystem mount root is unavailable")
	ErrRootChanged     = errors.New("filesystem mount root changed")
	ErrDirectoryLimit  = errors.New("filesystem directory exceeds scan limit")
	ErrStaleCursor     = errors.New("filesystem cursor is stale or invalid")
	ErrReadLimit       = errors.New("filesystem file exceeds read limit")
)

// Mount configures one independently authorized Collection Resource. Root is
// owner-local configuration and never appears in descriptors or payloads.
type Mount struct {
	Name           string
	Root           string
	Label          string
	MaxReadBytes   int
	MaxScanEntries int
}

// Registration owns only the Resource IDs successfully installed by Register.
type Registration struct {
	node      *node.Node
	resources []registeredResource
	closeOnce sync.Once
	closeErr  error
}

type registeredResource struct {
	id       protocol.ResourceID
	resource *resource.HandlerResource
}

// Register validates every mount before changing the Registry, then installs
// one read-only Collection Resource per mount. A partial Registry failure is
// rolled back in reverse order.
func Register(runtimeNode *node.Node, mounts []Mount) (*Registration, error) {
	if runtimeNode == nil {
		return nil, errors.New("filesystem node is required")
	}
	if len(mounts) == 0 || len(mounts) > MaxMounts {
		return nil, fmt.Errorf("filesystem mounts must contain between 1 and %d entries", MaxMounts)
	}

	providers := make([]*provider, 0, len(mounts))
	names := make(map[string]struct{}, len(mounts))
	roots := make(map[string]struct{}, len(mounts))
	for index, mount := range mounts {
		current, err := prepareProvider(runtimeNode.ID(), mount)
		if err != nil {
			return nil, fmt.Errorf("filesystem mount %d: %w", index, err)
		}
		id := current.resource.Descriptor().ID
		if _, exists := names[id.Name]; exists {
			return nil, fmt.Errorf("filesystem mount %d: duplicate resource name %q", index, id.Name)
		}
		if _, exists := roots[current.rootKey]; exists {
			return nil, fmt.Errorf("filesystem mount %d: duplicate canonical root", index)
		}
		if _, exists := runtimeNode.Registry().Resolve(id); exists {
			return nil, fmt.Errorf("filesystem mount %d: resource %q already exists", index, id.Name)
		}
		names[id.Name] = struct{}{}
		roots[current.rootKey] = struct{}{}
		providers = append(providers, current)
	}

	registered := make([]registeredResource, 0, len(providers))
	for index, current := range providers {
		if err := runtimeNode.Registry().Register(current.resource); err != nil {
			for rollback := len(registered) - 1; rollback >= 0; rollback-- {
				_ = runtimeNode.Registry().Remove(registered[rollback].id)
			}
			return nil, fmt.Errorf("register filesystem mount %d resource %q: %w", index, current.resource.Descriptor().ID.Name, err)
		}
		registered = append(registered, registeredResource{id: current.resource.Descriptor().ID, resource: current.resource})
	}
	return &Registration{node: runtimeNode, resources: registered}, nil
}

func (r *Registration) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		for index := len(r.resources) - 1; index >= 0; index-- {
			registered := r.resources[index]
			current, ok := r.node.Registry().Resolve(registered.id)
			if !ok || current != registered.resource {
				continue
			}
			if err := r.node.Registry().Remove(registered.id); err != nil && !errors.Is(err, resource.ErrNotFound) {
				r.closeErr = errors.Join(r.closeErr, err)
			}
		}
	})
	return r.closeErr
}

func prepareProvider(owner protocol.NodeID, mount Mount) (*provider, error) {
	if strings.TrimSpace(mount.Name) == "" || strings.TrimSpace(mount.Name) != mount.Name {
		return nil, errors.New("resource name is required without surrounding whitespace")
	}
	id := protocol.ResourceID{Owner: owner, Name: mount.Name}
	if err := id.Validate(); err != nil {
		return nil, fmt.Errorf("resource name: %w", err)
	}
	if id.Name == protocol.BuiltinResourceCatalog {
		return nil, errors.New("resource name is reserved")
	}
	if mount.Root == "" || !filepath.IsAbs(mount.Root) {
		return nil, errors.New("root must be an absolute path")
	}
	if mount.MaxReadBytes == 0 {
		mount.MaxReadBytes = protocol.MaxFilesystemReadBytes
	}
	if mount.MaxReadBytes < 1 || mount.MaxReadBytes > protocol.MaxFilesystemReadBytes {
		return nil, fmt.Errorf("max read bytes must be between 1 and %d", protocol.MaxFilesystemReadBytes)
	}
	if mount.MaxScanEntries == 0 {
		mount.MaxScanEntries = DefaultScanEntries
	}
	if mount.MaxScanEntries < 1 || mount.MaxScanEntries > MaxScanEntries {
		return nil, fmt.Errorf("max scan entries must be between 1 and %d", MaxScanEntries)
	}
	label := mount.Label
	if label == "" {
		label = mount.Name
	}
	if strings.TrimSpace(label) == "" || strings.TrimSpace(label) != label || len(label) > protocol.MaxLabelBytes {
		return nil, fmt.Errorf("label must contain between 1 and %d UTF-8 bytes without surrounding whitespace", protocol.MaxLabelBytes)
	}

	originalRoot, err := filepath.Abs(filepath.Clean(mount.Root))
	if err != nil {
		return nil, ErrRootUnavailable
	}
	originalInfo, err := os.Lstat(originalRoot)
	if err != nil || !originalInfo.IsDir() || originalInfo.Mode()&os.ModeSymlink != 0 {
		return nil, ErrRootUnavailable
	}
	canonicalRoot, err := filepath.EvalSymlinks(originalRoot)
	if err != nil {
		return nil, ErrRootUnavailable
	}
	canonicalRoot, err = filepath.Abs(canonicalRoot)
	if err != nil {
		return nil, ErrRootUnavailable
	}
	info, err := os.Stat(canonicalRoot)
	if err != nil || !info.IsDir() {
		return nil, ErrRootUnavailable
	}
	canonicalParent, err := filepath.EvalSymlinks(filepath.Dir(originalRoot))
	if err != nil {
		return nil, ErrRootUnavailable
	}
	canonicalParent, err = filepath.Abs(canonicalParent)
	if err != nil {
		return nil, ErrRootUnavailable
	}
	expectedRoot := filepath.Join(canonicalParent, filepath.Base(originalRoot))
	if pathComparisonKey(expectedRoot) != pathComparisonKey(canonicalRoot) {
		return nil, errors.New("root must be a real directory, not a symlink or junction")
	}

	value := &provider{
		id: id, originalRoot: originalRoot,
		rootKey: pathComparisonKey(canonicalRoot), rootIdentity: captureRootIdentity(info),
		label: label, maxReadBytes: mount.MaxReadBytes, maxScanEntries: mount.MaxScanEntries,
	}
	valueResource, err := resource.NewHandlerResource(value.descriptor(), map[protocol.CapabilityID]resource.Handler{
		protocol.CapabilityGet:  value.get,
		protocol.CapabilityList: value.list,
		protocol.CapabilityRead: value.read,
	})
	if err != nil {
		return nil, err
	}
	value.resource = valueResource
	return value, nil
}

func pathComparisonKey(value string) string {
	cleaned := filepath.Clean(value)
	if runtime.GOOS == "windows" {
		return strings.ToLower(cleaned)
	}
	return cleaned
}
