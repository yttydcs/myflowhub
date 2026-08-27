package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	desktopbinding "github.com/yttydcs/myflowhub/sdk/bindings/desktop"
)

const maxMessageBytes = 1 << 20

type Server struct {
	client     *desktopbinding.Client
	allowWrite bool
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func New(client *desktopbinding.Client, allowWrite bool) (*Server, error) {
	if client == nil {
		return nil, errors.New("desktop MCP client is required")
	}
	return &Server{client: client, allowWrite: allowWrite}, nil
}

func (s *Server) Serve(input io.Reader, output io.Writer) error {
	if input == nil || output == nil {
		return errors.New("desktop MCP input and output are required")
	}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64<<10), maxMessageBytes)
	encoder := json.NewEncoder(output)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		response, reply := s.handle(line)
		if !reply {
			continue
		}
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("write desktop MCP response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read desktop MCP request: %w", err)
	}
	return nil
}

func (s *Server) handle(data []byte) (rpcResponse, bool) {
	var request rpcRequest
	if err := json.Unmarshal(data, &request); err != nil || request.JSONRPC != "2.0" || request.Method == "" {
		return failure(nil, -32600, "invalid JSON-RPC request"), true
	}
	if len(request.ID) == 0 {
		return rpcResponse{}, false
	}
	switch request.Method {
	case "initialize":
		return success(request.ID, map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "myflowhub-desktop", "version": "1.0.0"}}), true
	case "ping":
		return success(request.ID, map[string]any{}), true
	case "tools/list":
		return success(request.ID, map[string]any{"tools": tools()}), true
	case "tools/call":
		result, err := s.call(request.Params)
		if err != nil {
			return success(request.ID, toolResult(nil, err)), true
		}
		return success(request.ID, toolResult(result, nil)), true
	default:
		return failure(request.ID, -32601, "method not found"), true
	}
}

func (s *Server) call(raw json.RawMessage) (any, error) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := decodeStrict(raw, &params); err != nil {
		return nil, err
	}
	switch params.Name {
	case "mfh_identity":
		value, err := s.client.IdentityJSON()
		return decodeResult(value, err)
	case "mfh_session_status":
		value, err := s.client.StatusJSON()
		return decodeResult(value, err)
	case "mfh_resource_catalog":
		owner, err := integerArgument(params.Arguments, "owner_node_id")
		if err != nil {
			return nil, err
		}
		value, err := s.client.CatalogJSON(owner, 15_000)
		return decodeResult(value, err)
	case "mfh_resource_snapshot":
		owner, name, err := resourceArguments(params.Arguments)
		if err != nil {
			return nil, err
		}
		value, err := s.client.SnapshotJSON(owner, name, 15_000)
		return decodeResult(value, err)
	case "mfh_command_invoke":
		if !s.allowWrite {
			return nil, errors.New("command invocation is disabled; start with explicit --allow-write")
		}
		owner, name, err := resourceArguments(params.Arguments)
		if err != nil {
			return nil, err
		}
		request, exists := params.Arguments["request"]
		if !exists {
			return nil, errors.New("request is required")
		}
		requestJSON, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		value, err := s.client.InvokeJSON(owner, name, string(requestJSON), 15_000)
		return decodeResult(value, err)
	default:
		return nil, errors.New("unknown desktop MCP tool")
	}
}

func tools() []tool {
	integer := map[string]any{"type": "integer", "minimum": 1}
	resource := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"owner_node_id", "name"}, "properties": map[string]any{"owner_node_id": integer, "name": map[string]any{"type": "string", "minLength": 1}}}
	return []tool{
		{Name: "mfh_identity", Description: "Read the persistent desktop node identity.", InputSchema: emptySchema()},
		{Name: "mfh_session_status", Description: "Read the managed parent connection status.", InputSchema: emptySchema()},
		{Name: "mfh_resource_catalog", Description: "Discover Variables, Streams, and Commands owned by a node.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"owner_node_id"}, "properties": map[string]any{"owner_node_id": integer}}},
		{Name: "mfh_resource_snapshot", Description: "Read a Variable snapshot by canonical resource identity.", InputSchema: resource},
		{Name: "mfh_command_invoke", Description: "Invoke a canonical Command. Requires the local allow-write gate.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"owner_node_id", "name", "request"}, "properties": map[string]any{"owner_node_id": integer, "name": map[string]any{"type": "string", "minLength": 1}, "request": map[string]any{"type": "object"}}}},
	}
}

func emptySchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{}}
}

func resourceArguments(arguments map[string]any) (int64, string, error) {
	owner, err := integerArgument(arguments, "owner_node_id")
	if err != nil {
		return 0, "", err
	}
	name, ok := arguments["name"].(string)
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return 0, "", errors.New("name is required")
	}
	return owner, name, nil
}

func integerArgument(arguments map[string]any, name string) (int64, error) {
	raw, exists := arguments[name]
	if !exists {
		return 0, fmt.Errorf("%s is required", name)
	}
	var text string
	switch value := raw.(type) {
	case float64:
		text = strconv.FormatFloat(value, 'f', -1, 64)
	case json.Number:
		text = value.String()
	default:
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive signed 64-bit integer", name)
	}
	return parsed, nil
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid tool call: %w", err)
	}
	return nil
}

func decodeResult(raw string, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	var result any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("decode SDK result: %w", err)
	}
	return result, nil
}

func toolResult(value any, err error) map[string]any {
	if err != nil {
		return map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": err.Error()}}}
	}
	data, marshalErr := json.MarshalIndent(value, "", "  ")
	if marshalErr != nil {
		return map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": marshalErr.Error()}}}
	}
	return map[string]any{"content": []map[string]string{{"type": "text", "text": string(data)}}, "structuredContent": value}
}

func success(id json.RawMessage, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func failure(id json.RawMessage, code int, message string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}
