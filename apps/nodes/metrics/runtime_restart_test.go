package metrics

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/host/nodehost"
	"github.com/yttydcs/myflowhub/internal/keystore"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestNormalRuntimeRestartPreservesRemoteDurableSubscriptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	identity, err := auth.GenerateIdentity(1)
	if err != nil {
		t.Fatal(err)
	}
	trust := auth.NewTrustStore()
	if err := trust.Add(1, identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	store, err := keystore.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	admission, err := auth.LoadAdmission(identity, store, auth.AdmissionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	hub, err := node.New(ctx, node.Config{Identity: identity, Trust: trust, Admission: admission, Policy: auth.AllowAll{}})
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Close()
	network := memory.NewNetwork()
	defer network.Close()
	endpoint, err := hub.Listen(network, "metrics-normal-restart")
	if err != nil {
		t.Fatal(err)
	}
	prepare := func(id protocol.NodeID) (string, protocol.ProvisioningPermitV1) {
		t.Helper()
		state, err := auth.OpenState(t.TempDir(), id)
		if err != nil {
			t.Fatal(err)
		}
		permit, err := admission.Issue(id, state.Identity.PublicKey, "restart-test", time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if err := state.Policy.Close(); err != nil {
			t.Fatal(err)
		}
		return state.Directory, permit
	}
	metricDirectory, metricPermit := prepare(2)
	consumerDirectory, consumerPermit := prepare(3)
	consumer, err := nodehost.New(ctx, nodehost.Config{StateDirectory: consumerDirectory, NodeID: 3,
		Parent: &nodehost.ParentConfig{NodeID: 1, PublicKey: identity.PublicKey, Permit: &consumerPermit, Driver: network, Endpoint: endpoint}})
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	if err := consumer.Start(); err != nil {
		t.Fatal(err)
	}
	status, ok := consumer.ParentStatus()
	if !ok {
		t.Fatal("consumer has no parent status")
	}
	for status.Snapshot().State != sdk.ConnectionConnected {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(time.Millisecond):
		}
	}
	start := func() *Runtime {
		t.Helper()
		r, err := Start(ctx, RuntimeConfig{StateDirectory: metricDirectory, NodeID: 2, ParentID: 1,
			ParentKey: identity.PublicKey, Permit: &metricPermit, Driver: network, Endpoint: endpoint, Platform: "windows"})
		if err != nil {
			t.Fatal(err)
		}
		waitRuntimeConnected(t, r)
		if err := r.Metrics.Update(CPUPercent, 40, nil); err != nil {
			t.Fatal(err)
		}
		return r
	}
	current := start()
	defer func() { _ = current.Close() }()
	resource := protocol.ResourceID{Owner: 2, Name: ResourceName(CPUPercent)}
	subscriptions := make([]*sdk.DurableSubscription, 16)
	instances := make([]string, len(subscriptions))
	snapshot := func(sub *sdk.DurableSubscription) string {
		t.Helper()
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		for {
			select {
			case event, ok := <-sub.Events:
				if !ok {
					t.Fatal("durable subscription ended during normal owner restart")
				}
				if event.Kind != sdk.EventSnapshot {
					continue
				}
				var value protocol.VariableValueV1
				if err := json.Unmarshal(event.Value, &value); err != nil {
					t.Fatal(err)
				}
				return value.InstanceID
			case err := <-sub.Errors:
				t.Fatalf("normal restart became a terminal subscription failure: %v", err)
			case <-timer.C:
				t.Fatal("normal restart did not produce a new snapshot")
			}
		}
	}
	for i := range subscriptions {
		subscriptions[i], err = consumer.Client().SubscribeDurableStatus(ctx, status, resource, time.Minute, 32)
		if err != nil {
			t.Fatal(err)
		}
		defer subscriptions[i].Cancel()
		instances[i] = snapshot(subscriptions[i])
	}
	for round := 0; round < 5; round++ {
		if err := current.Close(); err != nil {
			t.Fatal(err)
		}
		current = start()
		for i, sub := range subscriptions {
			next := snapshot(sub)
			if next == instances[i] {
				t.Fatalf("round %d retained retired resource instance", round)
			}
			instances[i] = next
		}
	}
}
