package sdk

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
	"github.com/yttydcs/myflowhub/runtime/subscription"
)

type renewalConnection struct{ disconnected atomic.Bool }

func (s *renewalConnection) Snapshot() ConnectionSnapshot {
	if s.disconnected.Load() {
		return ConnectionSnapshot{State: ConnectionDisconnected, Generation: 2}
	}
	return ConnectionSnapshot{State: ConnectionConnected, Generation: 1}
}
func (s *renewalConnection) WaitChange(ctx context.Context, _ uint64) (ConnectionSnapshot, error) {
	<-ctx.Done()
	return ConnectionSnapshot{}, ctx.Err()
}

type renewalVariable struct {
	*resource.Variable
	fail atomic.Bool
}

func (v *renewalVariable) Observe(observer func(resource.Observation)) (*resource.Observation, func(), error) {
	if v.fail.Load() {
		return nil, nil, subscription.ErrSubscriptionLimit
	}
	return v.Variable.Observe(observer)
}

func TestDurableRenewalInvalidatesPreviousValue(t *testing.T) {
	for _, scenario := range []string{"disconnected-before-renewal", "renewal-setup-fails"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			identity, err := auth.GenerateIdentity(91)
			if err != nil {
				t.Fatal(err)
			}
			runtime, err := node.New(ctx, node.Config{Identity: identity, Trust: auth.NewTrustStore(), Policy: auth.AllowAll{}, Subscriptions: subscription.Config{MinLease: time.Millisecond, MaxLease: time.Minute}})
			if err != nil {
				t.Fatal(err)
			}
			defer runtime.Close()
			id := protocol.ResourceID{Owner: 91, Name: "test/renewal-gap"}
			variable, err := resource.NewVariable(resource.VariableDescriptor(id, "read", 2048), protocol.DataSchemaDefinition{Type: "boolean"}, resource.VariableState{Status: "ready", Value: json.RawMessage(`false`)}, nil)
			if err != nil {
				t.Fatal(err)
			}
			provider := &renewalVariable{Variable: variable}
			if err := runtime.Registry().Register(provider); err != nil {
				t.Fatal(err)
			}
			client, err := NewAttachedClient(runtime)
			if err != nil {
				t.Fatal(err)
			}
			status := &renewalConnection{}
			sub, err := client.SubscribeDurableStatus(ctx, status, id, 200*time.Millisecond, 4)
			if err != nil {
				t.Fatal(err)
			}
			defer sub.Cancel()
			if first := receiveDurable(t, sub); first.Kind != EventSnapshot {
				t.Fatalf("first event = %+v", first)
			}
			if scenario == "disconnected-before-renewal" {
				status.disconnected.Store(true)
			} else {
				provider.fail.Store(true)
			}
			select {
			case event := <-sub.Events:
				if event.Kind != EventGap || event.Resource != id {
					t.Fatalf("previous value was not invalidated: %+v", event)
				}
			case err := <-sub.Errors:
				t.Fatalf("transient interruption became terminal: %v", err)
			case <-time.After(time.Second):
				t.Fatal("old snapshot remained live while renewal was waiting or retrying")
			}
			select {
			case event := <-sub.Events:
				t.Fatalf("repeated gap without a new snapshot: %+v", event)
			case <-time.After(250 * time.Millisecond):
			}
		})
	}
}
