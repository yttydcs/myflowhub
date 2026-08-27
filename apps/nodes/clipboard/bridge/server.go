package bridge

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	clipboard "github.com/yttydcs/myflowhub/apps/nodes/clipboard"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/link"
)

const MaxMessageBytes = 512 * 1024

type AdapterFactory func() clipboard.Adapter

type Server struct {
	factory AdapterFactory

	mu      sync.Mutex
	runtime *clipboard.Runtime
	cancel  context.CancelFunc
}

type requestV1 struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Op      string          `json:"op"`
	Payload json.RawMessage `json:"payload"`
}

func (r requestV1) Validate() error {
	if r.Version != 1 || r.ID == "" || len(r.ID) > 128 || len(r.Op) < 1 || len(r.Op) > 64 || len(r.Payload) > MaxMessageBytes {
		return errors.New("clipboard bridge request is invalid")
	}
	for _, character := range r.ID {
		if character < 0x21 || character > 0x7e {
			return errors.New("clipboard bridge request ID is invalid")
		}
	}
	return nil
}

type responseV1 struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	OK      bool            `json:"ok"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type identityRequestV1 struct {
	Version        int    `json:"version"`
	StateDirectory string `json:"state_directory"`
	NodeID         string `json:"node_id"`
}

func (r identityRequestV1) Validate() error {
	if r.Version != 1 || r.StateDirectory == "" {
		return errors.New("clipboard bridge identity request is invalid")
	}
	_, err := parseNodeID(r.NodeID)
	return err
}

type startRequestV1 struct {
	Version         int                            `json:"version"`
	StateDirectory  string                         `json:"state_directory"`
	NodeID          string                         `json:"node_id"`
	ParentNodeID    string                         `json:"parent_node_id"`
	Endpoint        string                         `json:"endpoint"`
	ParentPublicKey string                         `json:"parent_public_key"`
	Permit          *protocol.ProvisioningPermitV1 `json:"permit,omitempty"`
}

func (r startRequestV1) Validate() error {
	if r.Version != 1 || r.StateDirectory == "" {
		return errors.New("clipboard bridge start request is invalid")
	}
	if _, err := parseNodeID(r.NodeID); err != nil {
		return err
	}
	if _, err := parseNodeID(r.ParentNodeID); err != nil {
		return err
	}
	if err := link.Endpoint(r.Endpoint).Validate(); err != nil {
		return err
	}
	key, err := base64.RawStdEncoding.DecodeString(r.ParentPublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return errors.New("clipboard bridge parent key must be raw-base64 Ed25519")
	}
	if r.Permit != nil {
		return r.Permit.Validate()
	}
	return nil
}

type emptyV1 struct {
	Version int `json:"version"`
}

func (r emptyV1) Validate() error {
	if r.Version != 1 {
		return errors.New("clipboard bridge request version must be 1")
	}
	return nil
}

func New(factory AdapterFactory) (*Server, error) {
	if factory == nil {
		return nil, errors.New("clipboard bridge adapter factory is required")
	}
	return &Server{factory: factory}, nil
}

func (s *Server) Serve(ctx context.Context, input io.Reader, output io.Writer) error {
	if ctx == nil || input == nil || output == nil {
		return errors.New("clipboard bridge requires context, input, and output")
	}
	defer s.Close()
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), MaxMessageBytes+1)
	writer := bufio.NewWriter(output)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		response := s.handle(ctx, line)
		encoded, err := json.Marshal(response)
		if err != nil {
			return fmt.Errorf("encode clipboard bridge response: %w", err)
		}
		if len(encoded) > MaxMessageBytes {
			return errors.New("clipboard bridge response exceeds limit")
		}
		if _, err := writer.Write(encoded); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) || strings.Contains(err.Error(), "token too long") {
			return errors.New("clipboard bridge request exceeds limit")
		}
		return err
	}
	return nil
}

func (s *Server) Close() error {
	s.mu.Lock()
	runtime := s.runtime
	cancel := s.cancel
	s.runtime = nil
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if runtime != nil {
		return runtime.Close()
	}
	return nil
}

func (s *Server) handle(ctx context.Context, encoded []byte) responseV1 {
	var request requestV1
	if err := protocol.DecodeJSONPayload(encoded, MaxMessageBytes, &request); err != nil {
		return responseV1{Version: 1, ID: request.ID, Error: safeError(err)}
	}
	result, err := s.dispatch(ctx, request)
	if err != nil {
		return responseV1{Version: 1, ID: request.ID, Error: safeError(err)}
	}
	return responseV1{Version: 1, ID: request.ID, OK: true, Result: result}
}

func (s *Server) dispatch(ctx context.Context, request requestV1) (json.RawMessage, error) {
	switch request.Op {
	case "identity":
		var value identityRequestV1
		if err := decodePayload(request.Payload, &value); err != nil {
			return nil, err
		}
		nodeID, _ := parseNodeID(value.NodeID)
		state, err := auth.OpenState(value.StateDirectory, nodeID)
		if err != nil {
			return nil, err
		}
		return encodeResult(map[string]any{
			"version": 1, "node_id": value.NodeID,
			"public_key": base64.RawStdEncoding.EncodeToString(state.Identity.PublicKey),
		})
	case "start":
		var value startRequestV1
		if err := decodePayload(request.Payload, &value); err != nil {
			return nil, err
		}
		return s.start(ctx, value)
	case "stop":
		var value emptyV1
		if err := decodePayload(request.Payload, &value); err != nil {
			return nil, err
		}
		if err := s.Close(); err != nil {
			return nil, err
		}
		return encodeResult(map[string]any{"version": 1, "stopped": true})
	case "status":
		if err := validateEmpty(request.Payload); err != nil {
			return nil, err
		}
		runtime, err := s.current()
		if err != nil {
			return nil, err
		}
		return encodeResult(map[string]any{
			"version": 1, "connection": runtime.Connection.Snapshot(), "clipboard": runtime.Clipboard.Status(),
		})
	case "configuration":
		if err := validateEmpty(request.Payload); err != nil {
			return nil, err
		}
		runtime, err := s.current()
		if err != nil {
			return nil, err
		}
		return encodeResult(runtime.Clipboard.Config())
	case "configuration.update":
		var value clipboard.ConfigUpdateV1
		if err := decodePayload(request.Payload, &value); err != nil {
			return nil, err
		}
		runtime, err := s.current()
		if err != nil {
			return nil, err
		}
		updated, err := runtime.Clipboard.ApplyConfig(ctx, value)
		if err != nil {
			return nil, err
		}
		return encodeResult(updated)
	case "send":
		var value clipboard.SendV1
		if err := decodePayload(request.Payload, &value); err != nil {
			return nil, err
		}
		runtime, err := s.current()
		if err != nil {
			return nil, err
		}
		decision, err := runtime.Clipboard.SendText(ctx, value.Text)
		if err != nil {
			return nil, err
		}
		return encodeResult(decision)
	case "apply":
		var value clipboard.ApplyV1
		if err := decodePayload(request.Payload, &value); err != nil {
			return nil, err
		}
		runtime, err := s.current()
		if err != nil {
			return nil, err
		}
		decision, err := runtime.Clipboard.ApplyPending(ctx, value.EventID)
		if err != nil {
			return nil, err
		}
		return encodeResult(decision)
	case "history":
		if err := validateEmpty(request.Payload); err != nil {
			return nil, err
		}
		runtime, err := s.current()
		if err != nil {
			return nil, err
		}
		return encodeResult(runtime.Clipboard.History())
	case "history.clear":
		if err := validateEmpty(request.Payload); err != nil {
			return nil, err
		}
		runtime, err := s.current()
		if err != nil {
			return nil, err
		}
		result, err := runtime.Clipboard.ClearHistory()
		if err != nil {
			return nil, err
		}
		return encodeResult(result)
	default:
		return nil, fmt.Errorf("clipboard bridge operation %q is unsupported", request.Op)
	}
}

func (s *Server) start(ctx context.Context, request startRequestV1) (json.RawMessage, error) {
	nodeID, _ := parseNodeID(request.NodeID)
	parentID, _ := parseNodeID(request.ParentNodeID)
	key, _ := base64.RawStdEncoding.DecodeString(request.ParentPublicKey)
	s.mu.Lock()
	if s.runtime != nil {
		s.mu.Unlock()
		return nil, errors.New("clipboard bridge runtime is already running")
	}
	runCtx, cancel := context.WithCancel(ctx)
	adapter := s.factory()
	if adapter == nil {
		cancel()
		s.mu.Unlock()
		return nil, errors.New("clipboard bridge adapter factory returned nil")
	}
	runtime, err := clipboard.Start(runCtx, clipboard.RuntimeConfig{
		StateDirectory: request.StateDirectory, NodeID: nodeID, ParentID: parentID, ParentKey: ed25519.PublicKey(key),
		Permit: request.Permit, Endpoint: link.Endpoint(request.Endpoint), Adapter: adapter,
	})
	if err != nil {
		_ = adapter.Close()
		cancel()
		s.mu.Unlock()
		return nil, err
	}
	s.runtime = runtime
	s.cancel = cancel
	s.mu.Unlock()
	return encodeResult(map[string]any{"version": 1, "running": true, "connection": runtime.Connection.Snapshot()})
}

func (s *Server) current() (*clipboard.Runtime, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtime == nil {
		return nil, errors.New("clipboard bridge runtime is not running")
	}
	return s.runtime, nil
}

func decodePayload[T protocol.ValidatedPayload](encoded json.RawMessage, target T) error {
	return protocol.DecodeJSONPayload(encoded, MaxMessageBytes, target)
}

func validateEmpty(encoded json.RawMessage) error {
	var value emptyV1
	return decodePayload(encoded, &value)
}

func encodeResult(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(encoded) > MaxMessageBytes {
		return nil, errors.New("clipboard bridge result exceeds limit")
	}
	return encoded, nil
}

func parseNodeID(value string) (protocol.NodeID, error) {
	if value == "" || strings.HasPrefix(value, "+") || (len(value) > 1 && strings.HasPrefix(value, "0")) {
		return 0, errors.New("node ID must be a canonical non-zero decimal value")
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, errors.New("node ID must be a canonical non-zero decimal value")
	}
	return protocol.NodeID(parsed), nil
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ReplaceAll(strings.ReplaceAll(err.Error(), "\r", " "), "\n", " ")
	if len(message) > 1024 {
		message = message[:1024]
	}
	return message
}
