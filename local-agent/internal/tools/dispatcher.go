package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"oncode/local-agent/internal/audit"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/messages"
)

// Handler executes a single tool call.
type Handler func(ctx context.Context, req *messages.ToolRequestPayload) (*messages.ToolResponsePayload, error)

// Registry maps tool names to handlers.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

func (r *Registry) Register(name string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = h
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) Get(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[name]
	return h, ok
}

// Dispatcher routes ToolRequest payloads to registered handlers.
type Dispatcher struct {
	reg    *Registry
	audit  *audit.Logger
}

func NewDispatcher(reg *Registry, logger *audit.Logger) *Dispatcher {
	return &Dispatcher{reg: reg, audit: logger}
}

func (d *Dispatcher) Dispatch(ctx context.Context, req *messages.ToolRequestPayload) *messages.ToolResponsePayload {
	start := time.Now()
	if req == nil {
		return failed("", protoerr.InvalidRequest, "nil tool request")
	}
	if err := req.Validate(); err != nil {
		resp := failed(req.CallID, protoerr.InvalidRequest, err.Error())
		d.log(req, resp, start)
		return resp
	}
	h, ok := d.reg.Get(req.Tool)
	if !ok {
		resp := failed(req.CallID, protoerr.ToolNotSupported, fmt.Sprintf("tool %q is not supported", req.Tool))
		d.log(req, resp, start)
		return resp
	}
	resp, err := h(ctx, req)
	if err != nil {
		resp = mapError(req.CallID, err)
	}
	if resp == nil {
		resp = failed(req.CallID, protoerr.InternalError, "handler returned nil response")
	}
	if resp.Metrics == nil {
		resp.Metrics = &messages.Metrics{DurationMS: time.Since(start).Milliseconds()}
	}
	d.log(req, resp, start)
	return resp
}

func (d *Dispatcher) log(req *messages.ToolRequestPayload, resp *messages.ToolResponsePayload, start time.Time) {
	if d.audit == nil || req == nil || resp == nil {
		return
	}
	ev := audit.ToolEvent{
		CallID:     req.CallID,
		Tool:       req.Tool,
		Status:     resp.Status,
		DurationMS: time.Since(start).Milliseconds(),
	}
	if resp.Error != nil {
		ev.ErrorCode = string(resp.Error.Code)
	}
	d.audit.Tool(ev)
}

func Success(callID string, result any) *messages.ToolResponsePayload {
	raw, err := json.Marshal(result)
	if err != nil {
		return failed(callID, protoerr.InternalError, "marshal result")
	}
	return &messages.ToolResponsePayload{
		CallID: callID,
		Status: messages.StatusSuccess,
		Result: raw,
	}
}

func failed(callID string, code protoerr.Code, msg string) *messages.ToolResponsePayload {
	e, err := protoerr.New(code, msg)
	if err != nil {
		e, _ = protoerr.New(protoerr.InternalError, msg)
	}
	return &messages.ToolResponsePayload{
		CallID: callID,
		Status: messages.StatusFailed,
		Error:  e,
	}
}

func mapError(callID string, err error) *messages.ToolResponsePayload {
	if pe, ok := err.(*protoerr.Error); ok {
		return &messages.ToolResponsePayload{
			CallID: callID,
			Status: messages.StatusFailed,
			Error:  pe,
		}
	}
	return failed(callID, protoerr.InternalError, err.Error())
}
