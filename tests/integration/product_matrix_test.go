package integration_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	clipboard "github.com/yttydcs/myflowhub/apps/nodes/clipboard"
	metrics "github.com/yttydcs/myflowhub/apps/nodes/metrics"
	"github.com/yttydcs/myflowhub/host/hub"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
	"github.com/yttydcs/myflowhub/transport/memory"
)

func TestCanonicalProductMatrixSharesAuthoritativeTree(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	network := memory.NewNetwork()
	defer network.Close()
	root, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: t.TempDir(), NodeID: 1,
		Listeners: []hub.ListenerConfig{{Driver: network, Endpoint: "product-matrix"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	clientState, clientNode := joinProductNode(t, ctx, root, 2)
	defer clientNode.Close()
	_ = clientState
	metricsState, metricsNode := joinProductNode(t, ctx, root, 3)
	defer metricsNode.Close()
	metricsController, err := metrics.Register(metrics.ControllerConfig{Node: metricsNode, Store: metricsState.Store, Platform: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	defer metricsController.Close()
	clipboardState, clipboardNode := joinProductNode(t, ctx, root, 4)
	defer clipboardNode.Close()
	clipboardController, err := clipboard.Register(clipboard.ControllerConfig{Node: clipboardNode, Store: clipboardState.Store})
	if err != nil {
		t.Fatal(err)
	}
	defer clipboardController.Close()
	currentClipboard := clipboard.DefaultConfig()
	_, err = clipboardController.ApplyConfig(ctx, clipboard.ConfigUpdateV1{
		Version: 1, ExpectedRevision: currentClipboard.Revision, Enabled: true,
		MaxInlineBytes: currentClipboard.MaxInlineBytes, AutoWatch: currentClipboard.AutoWatch, AutoApply: currentClipboard.AutoApply,
		HistoryRetention: currentClipboard.HistoryRetention, HistoryLimit: currentClipboard.HistoryLimit,
		HistoryMaxBytes: currentClipboard.HistoryMaxBytes, HistoryTTLMS: currentClipboard.HistoryTTLMS, Peers: currentClipboard.Peers,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, child := range []*node.Node{clientNode, metricsNode, clipboardNode} {
		if err := child.ConnectParent(ctx, network, root.Endpoint, root.Node.ID()); err != nil {
			t.Fatal(err)
		}
		waitDownRoute(t, root.Node.Tree(), child.ID())
	}
	client, err := sdk.NewAttachedClient(clientNode)
	if err != nil {
		t.Fatal(err)
	}
	metricID := protocol.ResourceID{Owner: metricsNode.ID(), Name: metrics.ResourceName(metrics.CPUPercent)}
	clipboardEventsID := protocol.ResourceID{Owner: clipboardNode.ID(), Name: clipboard.ResourceEvents}
	clipboardSendID := protocol.ResourceID{Owner: clipboardNode.ID(), Name: clipboard.CommandSend}

	if denied, err := client.Subscribe(ctx, metricID, time.Minute, 8); !isSDKCode(err, protocol.CodeForbidden) {
		if denied != nil {
			denied.Cancel()
		}
		t.Fatalf("default-deny policy did not reject product subscription: %v", err)
	}
	grants := []auth.Request{
		{Subject: clientNode.ID(), Action: auth.ActionSubscribe, Resource: metricID},
		{Subject: clientNode.ID(), Action: auth.ActionSubscribe, Resource: clipboardEventsID},
		{Subject: clientNode.ID(), Action: auth.ActionInvoke, Resource: clipboardSendID},
	}
	for _, grant := range grants {
		if err := root.Runtime.Policy.Grant(grant); err != nil {
			t.Fatal(err)
		}
	}
	if err := metricsController.Update(metrics.CPUPercent, "37", nil); err != nil {
		t.Fatal(err)
	}
	metricSubscription, err := client.Subscribe(ctx, metricID, time.Minute, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer metricSubscription.Cancel()
	assertMetricEvent(t, receiveProductEvent(t, ctx, metricSubscription), "37")
	for index := 0; index < 256; index++ {
		value := fmt.Sprintf("%d", index%101)
		if err := metricsController.Update(metrics.CPUPercent, value, nil); err != nil {
			t.Fatal(err)
		}
		assertMetricEvent(t, receiveProductEvent(t, ctx, metricSubscription), value)
	}

	clipboardSubscription, err := client.Subscribe(ctx, clipboardEventsID, time.Minute, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer clipboardSubscription.Cancel()
	for index := 0; index < 64; index++ {
		body := fmt.Sprintf("matrix-clipboard-%03d", index)
		input, err := protocol.EncodeJSONPayload(&clipboard.SendV1{Version: 1, Text: body}, protocol.DefaultMaxPayload)
		if err != nil {
			t.Fatal(err)
		}
		output, err := client.Invoke(ctx, clipboardSendID, input)
		if err != nil {
			t.Fatal(err)
		}
		var decision clipboard.DecisionV1
		if err := protocol.DecodeJSONPayload(output, protocol.DefaultMaxPayload, &decision); err != nil || decision.Action != "published" {
			t.Fatalf("unexpected clipboard decision: %+v (%v)", decision, err)
		}
		var event clipboard.TextEventV1
		if err := protocol.DecodeJSONPayload(receiveProductEvent(t, ctx, clipboardSubscription).Value, protocol.DefaultMaxPayload, &event); err != nil || event.Text != body {
			t.Fatalf("unexpected clipboard event: %+v (%v)", event, err)
		}
	}

	if err := root.Runtime.Policy.Revoke(grants[0]); err != nil {
		t.Fatal(err)
	}
	if err := receiveProductError(t, ctx, metricSubscription); !isSDKCode(err, protocol.CodeExpired) {
		t.Fatalf("policy generation change did not expire product subscription: %v", err)
	}
	if err := receiveProductError(t, ctx, clipboardSubscription); !isSDKCode(err, protocol.CodeExpired) {
		t.Fatalf("policy generation change did not expire sibling product subscription: %v", err)
	}
	waitSubscriptionCount(t, root.Node, 0)
	waitSubscriptionCount(t, metricsNode, 0)
	waitSubscriptionCount(t, clipboardNode, 0)
}

func TestScopedBindingCoversCurrentAndFutureMetricsNodes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	network := memory.NewNetwork()
	defer network.Close()
	root, err := hub.StartPersistent(ctx, hub.PersistentConfig{
		StateDirectory: t.TempDir(), NodeID: 1,
		Listeners: []hub.ListenerConfig{{Driver: network, Endpoint: "scoped-product-matrix"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, clientNode := joinProductNode(t, ctx, root, 41)
	client, err := sdk.NewAttachedClient(clientNode)
	if err != nil {
		t.Fatal(err)
	}
	defer clientNode.Close()
	metricsState, metricsNode := joinProductNode(t, ctx, root, 42)
	defer metricsNode.Close()
	metricsController, err := metrics.Register(metrics.ControllerConfig{Node: metricsNode, Store: metricsState.Store, Platform: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	defer metricsController.Close()
	for _, child := range []*node.Node{clientNode, metricsNode} {
		if err := child.ConnectParent(ctx, network, root.Endpoint, 1); err != nil {
			t.Fatal(err)
		}
		waitDownRoute(t, root.Node.Tree(), child.ID())
	}
	firstMetric := protocol.ResourceID{Owner: 42, Name: metrics.ResourceName(metrics.CPUPercent)}
	if denied, err := client.Subscribe(ctx, firstMetric, time.Minute, 8); !isSDKCode(err, protocol.CodeForbidden) {
		if denied != nil {
			denied.Cancel()
		}
		t.Fatalf("unbound Desktop-like client accessed Metrics: %v", err)
	}
	definition, err := root.Runtime.Policy.PutDefinition(protocol.PolicyDefinitionPutV1{
		Version: 1, ID: "metrics-observer", Label: "Metrics observer",
		Rules: []protocol.PolicyRuleV1{{
			Resource:   protocol.PolicyResourceSelectorV1{Kind: protocol.PolicySelectorPrefix, Value: "metrics"},
			Capability: protocol.PolicyCapabilitySelectorV1{Kind: protocol.PolicySelectorExact, Values: []protocol.CapabilityID{protocol.CapabilityRead, protocol.CapabilitySubscribe}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	bindingID := strings.Repeat("e", 32)
	if _, err := root.Runtime.Policy.CreateBinding(protocol.PolicyBindingCreateV1{
		Version: 1, BindingID: bindingID, Subject: "41", DefinitionID: definition.ID,
		Scope: protocol.PolicyOwnerScopeV1{Kind: protocol.PolicyScopeAuthorityDomain, NodeID: "1"},
	}, 1); err != nil {
		t.Fatal(err)
	}
	if err := metricsController.Update(metrics.CPUPercent, "42", nil); err != nil {
		t.Fatal(err)
	}
	firstSubscription, err := client.Subscribe(ctx, firstMetric, time.Minute, 8)
	if err != nil {
		t.Fatal(err)
	}
	defer firstSubscription.Cancel()
	assertMetricEvent(t, receiveProductEvent(t, ctx, firstSubscription), "42")

	futureState, futureNode := joinProductNode(t, ctx, root, 43)
	defer futureNode.Close()
	futureController, err := metrics.Register(metrics.ControllerConfig{Node: futureNode, Store: futureState.Store, Platform: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	defer futureController.Close()
	if err := futureNode.ConnectParent(ctx, network, root.Endpoint, 1); err != nil {
		t.Fatal(err)
	}
	waitDownRoute(t, root.Node.Tree(), futureNode.ID())
	if err := futureController.Update(metrics.CPUPercent, "43", nil); err != nil {
		t.Fatal(err)
	}
	futureMetric := protocol.ResourceID{Owner: 43, Name: metrics.ResourceName(metrics.CPUPercent)}
	futureSubscription, err := client.Subscribe(ctx, futureMetric, time.Minute, 8)
	if err != nil {
		t.Fatalf("scoped Binding did not cover future Metrics Node: %v", err)
	}
	defer futureSubscription.Cancel()
	assertMetricEvent(t, receiveProductEvent(t, ctx, futureSubscription), "43")

	if err := root.Runtime.Policy.RevokeBinding(bindingID); err != nil {
		t.Fatal(err)
	}
	for name, subscription := range map[string]*sdk.Subscription{"current": firstSubscription, "future": futureSubscription} {
		if err := receiveProductError(t, ctx, subscription); !isSDKCode(err, protocol.CodeExpired) {
			t.Fatalf("%s scoped subscription survived Binding revoke: %v", name, err)
		}
	}
}

func joinProductNode(t *testing.T, ctx context.Context, root *hub.Hub, id protocol.NodeID) (*auth.State, *node.Node) {
	t.Helper()
	state, err := auth.OpenState(t.TempDir(), id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = state.Policy.Close() })
	if err := state.Trust.Add(root.Node.ID(), root.Runtime.Identity.PublicKey); err != nil {
		t.Fatal(err)
	}
	permit, err := root.Runtime.Admission.Issue(id, state.Identity.PublicKey, "integration-product", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	value, err := node.New(ctx, node.Config{
		Identity: state.Identity, Trust: state.Trust, Policy: state.Policy, Admission: state.Admission, JoinPermit: &permit,
	})
	if err != nil {
		t.Fatal(err)
	}
	return state, value
}

func receiveProductEvent(t *testing.T, ctx context.Context, subscription *sdk.Subscription) sdk.Event {
	t.Helper()
	select {
	case event, ok := <-subscription.Events:
		if !ok {
			t.Fatal("product subscription closed without an event")
		}
		return event
	case err := <-subscription.Errors:
		t.Fatalf("product subscription failed: %v", err)
	case <-ctx.Done():
		t.Fatalf("product event timed out: %v", ctx.Err())
	}
	return sdk.Event{}
}

func receiveProductError(t *testing.T, ctx context.Context, subscription *sdk.Subscription) error {
	t.Helper()
	select {
	case err, ok := <-subscription.Errors:
		if !ok {
			t.Fatal("product subscription closed without its terminal error")
		}
		return err
	case <-ctx.Done():
		t.Fatalf("product subscription error timed out: %v", ctx.Err())
	}
	return nil
}

func assertMetricEvent(t *testing.T, event sdk.Event, want string) {
	t.Helper()
	var sample metrics.SampleV1
	if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &sample); err != nil {
		t.Fatal(err)
	}
	if sample.Status != "fresh" || sample.Value != want {
		t.Fatalf("unexpected metric sample: %+v", sample)
	}
}

func isSDKCode(err error, code protocol.ErrorCode) bool {
	var sdkError *sdk.Error
	return errors.As(err, &sdkError) && sdkError.Code == code
}

func waitSubscriptionCount(t *testing.T, value *node.Node, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if value.Stats().ActiveSubscriptions == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("node %d retained %d subscriptions, want %d", value.ID(), value.Stats().ActiveSubscriptions, want)
}
