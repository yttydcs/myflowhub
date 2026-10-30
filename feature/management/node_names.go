package management

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yttydcs/myflowhub/protocol"
)

// NodeDisplayNamePrefix reserves host Values entries for presentation-only
// descendant names. They never establish membership, identity or authorization.
// The local owner's name still comes from ManagementConfigV1.DisplayName.
const NodeDisplayNamePrefix = "node.display_name."

func nodeDisplayNames(values map[string]string) (map[string]string, error) {
	names := make(map[string]string)
	for key, name := range values {
		if !strings.HasPrefix(key, NodeDisplayNamePrefix) {
			continue
		}
		id := strings.TrimPrefix(key, NodeDisplayNamePrefix)
		parsed, err := strconv.ParseUint(id, 10, 64)
		if err != nil || parsed == 0 || strconv.FormatUint(parsed, 10) != id {
			return nil, fmt.Errorf("invalid node display-name key %q", key)
		}
		if err := (protocol.ManagementConfigV1{Version: 1, Revision: 1, DisplayName: name}).Validate(); err != nil {
			return nil, fmt.Errorf("invalid display name for node %s: %w", id, err)
		}
		// An explicit empty entry resets old clients' cached names to the fallback.
		// Omission cannot clear a name supplied by another topology provider.
		if name == "" {
			name = "Node " + id
		}
		names[id] = name
	}
	return names, nil
}
