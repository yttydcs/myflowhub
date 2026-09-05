package clipboard

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	sdk "github.com/yttydcs/myflowhub/sdk/go"
)

// SyncEngine owns the explicit peer subscriptions that feed a clipboard
// controller. Physical reconnects are handled by the SDK's durable
// subscriptions; configuration changes only add or remove logical peers.
type SyncEngine struct {
	client     *sdk.Client
	connection sdk.ConnectionStatus
	controller *Controller

	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	workers   map[protocol.NodeID]*peerWorker
	wg        sync.WaitGroup
	closeOnce sync.Once
}

type peerWorker struct{ cancel context.CancelFunc }

func StartSync(ctx context.Context, client *sdk.Client, connection sdk.ConnectionStatus, controller *Controller) (*SyncEngine, error) {
	if ctx == nil {
		return nil, errors.New("clipboard sync context is required")
	}
	if client == nil || connection == nil || controller == nil {
		return nil, errors.New("clipboard sync requires a client, managed connection, and controller")
	}
	runCtx, cancel := context.WithCancel(ctx)
	value := &SyncEngine{
		client: client, connection: connection, controller: controller,
		ctx: runCtx, cancel: cancel, workers: make(map[protocol.NodeID]*peerWorker),
	}
	value.wg.Add(1)
	go value.run()
	return value, nil
}

func (s *SyncEngine) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		s.cancel()
		s.mu.Lock()
		for _, worker := range s.workers {
			worker.cancel()
		}
		s.mu.Unlock()
		s.wg.Wait()
	})
}

func (s *SyncEngine) run() {
	defer s.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	s.reconcile()
	for {
		select {
		case <-ticker.C:
			s.reconcile()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *SyncEngine) reconcile() {
	config := s.controller.Config()
	wanted := make(map[protocol.NodeID]struct{})
	if config.Enabled {
		for _, peer := range config.Peers {
			id, err := parseNodeID(peer.NodeID)
			if err == nil && peer.Receive {
				wanted[id] = struct{}{}
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, worker := range s.workers {
		if _, ok := wanted[id]; ok {
			continue
		}
		worker.cancel()
		delete(s.workers, id)
	}
	for id := range wanted {
		if _, ok := s.workers[id]; ok {
			continue
		}
		workerCtx, cancel := context.WithCancel(s.ctx)
		worker := &peerWorker{cancel: cancel}
		s.workers[id] = worker
		s.wg.Add(1)
		go s.consumePeer(workerCtx, id, worker)
	}
}

func (s *SyncEngine) consumePeer(ctx context.Context, peer protocol.NodeID, worker *peerWorker) {
	defer s.wg.Done()
	defer func() {
		s.mu.Lock()
		if s.workers[peer] == worker {
			delete(s.workers, peer)
		}
		s.mu.Unlock()
	}()
	resourceID := protocol.ResourceID{Owner: peer, Name: ResourceEvents}
	for {
		subscription, err := s.client.SubscribeDurableStatus(ctx, s.connection, resourceID, time.Minute, MaxPendingEvents)
		if err != nil {
			s.controller.RecordError(fmt.Errorf("subscribe clipboard peer %d: %w", peer, err))
			if !waitController(ctx, time.Second) {
				return
			}
			continue
		}
		retry := s.consumeSubscription(ctx, peer, subscription)
		subscription.Cancel()
		if !retry || !waitController(ctx, time.Second) {
			return
		}
	}
}

func (s *SyncEngine) consumeSubscription(ctx context.Context, peer protocol.NodeID, current *sdk.DurableSubscription) bool {
	for {
		select {
		case event, ok := <-current.Events:
			if !ok {
				return ctx.Err() == nil
			}
			switch event.Kind {
			case sdk.EventData:
				var payload TextEventV1
				if err := protocol.DecodeJSONPayload(event.Value, protocol.DefaultMaxPayload, &payload); err != nil {
					s.controller.RecordError(fmt.Errorf("decode clipboard peer %d event: %w", peer, err))
					continue
				}
				if _, err := s.controller.Receive(ctx, peer, payload); err != nil {
					s.controller.RecordError(fmt.Errorf("receive clipboard peer %d event: %w", peer, redactClipboardError(err, payload.Text)))
				}
			case sdk.EventGap:
				s.controller.RecordError(fmt.Errorf("clipboard peer %d stream gap: %s", peer, event.Reason))
			}
		case err, ok := <-current.Errors:
			if ok && err != nil {
				s.controller.RecordError(fmt.Errorf("clipboard peer %d subscription: %w", peer, err))
			}
			return ctx.Err() == nil
		case <-ctx.Done():
			return false
		}
	}
}
