package nodehost

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/yttydcs/myflowhub/runtime/link"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/subscription"
)

const DefaultProductSubscriptions = 4096

// ProductRuntimeConfig selects bounded desktop-product capacity without
// changing the lightweight runtime defaults or any persisted identity schema.
// Subscription requests retain pending entries for their lifetime, so the
// pending budget must include business subscriptions and control headroom.
func ProductRuntimeConfig() (RuntimeConfig, error) {
	limit := DefaultProductSubscriptions
	if value, present := os.LookupEnv("MFH_MAX_SUBSCRIPTIONS"); present {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65536 {
			return RuntimeConfig{}, fmt.Errorf("MFH_MAX_SUBSCRIPTIONS must be an integer in 1..65536, got %q", value)
		}
		limit = parsed
	}
	return RuntimeConfig{
		Subscriptions: subscription.Config{MaxSubscriptions: limit}, MaxPending: limit + 256,
		Session: link.SessionConfig{
			ControlQueue: 4096, DataQueue: 4096, InboundQueue: 4096,
			ControlQueueBytes: 8 << 20, DataQueueBytes: 32 << 20, InboundQueueBytes: 32 << 20,
			HeartbeatInterval: time.Second, IdleTimeout: 3 * time.Second,
			IdleAfterPeerHeartbeat: true,
		},
	}, nil
}

// ProductSupervisorConfig bounds the normal product reconnect delay while
// retaining caller overrides used by tests and embedded hosts.
func ProductSupervisorConfig(config node.SupervisorConfig) node.SupervisorConfig {
	if config.MinBackoff == 0 {
		config.MinBackoff = 100 * time.Millisecond
	}
	if config.MaxBackoff == 0 {
		config.MaxBackoff = time.Second
	}
	return config
}
