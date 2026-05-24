// package mcp implements a minimal stdio json-rpc model context
// protocol server that proxies tool calls to a running smtp-dev-server
// http api. lets an llm client list mailboxes, read messages, and view
// raw rfc 5322 source without leaving the conversation.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const protocolVersion = "2024-11-05"

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

type server struct {
	upstream string
	http     *http.Client
}

// run reads json-rpc requests from r and writes responses to w. blocks
// until the input is closed or ctx is done.
func Run(ctx context.Context, r io.Reader, w io.Writer, upstream string) error {
	s := &server{
		upstream: strings.TrimRight(upstream, "/"),
		http:     &http.Client{Timeout: 10 * time.Second},
	}
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		var req rpcRequest
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		resp := s.handle(ctx, req)
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
}

func (s *server) handle(ctx context.Context, req rpcRequest) rpcResponse {
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "smtp-dev-server", "version": "0.2.0"},
		}
	case "tools/list":
		resp.Result = map[string]any{"tools": s.tools()}
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &p)
		result, err := s.callTool(ctx, p.Name, p.Arguments)
		if err != nil {
			resp.Result = map[string]any{
				"content": []map[string]any{{"type": "text", "text": "error: " + err.Error()}},
				"isError": true,
			}
		} else {
			b, _ := json.MarshalIndent(result, "", "  ")
			resp.Result = map[string]any{
				"content": []map[string]any{{"type": "text", "text": string(b)}},
			}
		}
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp
}

func (s *server) tools() []map[string]any {
	return []map[string]any{
		{
			"name":        "list_mailboxes",
			"description": "List distinct recipient addresses (lowercased) that have received at least one captured message, with per-mailbox message and unread counts. Useful for showing the dev-mock SMTP server as multiple user mailboxes.",
			"inputSchema": map[string]any{"type": "object"},
		},
		{
			"name":        "list_messages",
			"description": "List captured messages newest-first. Optional `to` filter narrows to messages addressed to a single recipient.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"to": map[string]any{"type": "string", "description": "recipient address to filter by"},
				},
			},
		},
		{
			"name":        "get_message",
			"description": "Get a single message's metadata (subject, from, to, receivedAt, size).",
			"inputSchema": map[string]any{
				"type":       "object",
				"required":   []string{"id"},
				"properties": map[string]any{"id": map[string]any{"type": "string"}},
			},
		},
		{
			"name":        "get_message_source",
			"description": "Get the raw RFC 5322 source of a message (headers + MIME body). Useful for inspecting full HTML, attachments, and exact rendered content.",
			"inputSchema": map[string]any{
				"type":       "object",
				"required":   []string{"id"},
				"properties": map[string]any{"id": map[string]any{"type": "string"}},
			},
		},
		{
			"name":        "delete_message",
			"description": "Delete a single message by id.",
			"inputSchema": map[string]any{
				"type":       "object",
				"required":   []string{"id"},
				"properties": map[string]any{"id": map[string]any{"type": "string"}},
			},
		},
		{
			"name":        "clear_messages",
			"description": "Delete every captured message.",
			"inputSchema": map[string]any{"type": "object"},
		},
	}
}

func (s *server) callTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "list_mailboxes":
		return s.getJSON(ctx, "/api/mailboxes")
	case "list_messages":
		var a struct {
			To string `json:"to"`
		}
		_ = json.Unmarshal(args, &a)
		path := "/api/messages"
		if a.To != "" {
			path += "?to=" + url.QueryEscape(a.To)
		}
		return s.getJSON(ctx, path)
	case "get_message":
		var a struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(args, &a)
		if a.ID == "" {
			return nil, fmt.Errorf("id required")
		}
		return s.getJSON(ctx, "/api/messages/"+url.PathEscape(a.ID))
	case "get_message_source":
		var a struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(args, &a)
		if a.ID == "" {
			return nil, fmt.Errorf("id required")
		}
		return s.getText(ctx, "/api/messages/"+url.PathEscape(a.ID)+"/raw")
	case "delete_message":
		var a struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(args, &a)
		if a.ID == "" {
			return nil, fmt.Errorf("id required")
		}
		return s.deleteJSON(ctx, "/api/messages/"+url.PathEscape(a.ID))
	case "clear_messages":
		return s.deleteJSON(ctx, "/api/messages")
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *server) getJSON(ctx context.Context, path string) (any, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.upstream+path, nil)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream %d: %s", resp.StatusCode, string(body))
	}
	var v any
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *server) getText(ctx context.Context, path string) (any, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.upstream+path, nil)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream %d: %s", resp.StatusCode, string(body))
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func (s *server) deleteJSON(ctx context.Context, path string) (any, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, s.upstream+path, nil)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream %d: %s", resp.StatusCode, string(body))
	}
	return map[string]any{"ok": true}, nil
}

// guard against unused imports drifting on changes above.
var _ = bytes.NewReader
