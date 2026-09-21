package grpcadapt

import (
	"encoding/json"
	"testing"

	"oncode/protocol/ids"
	"oncode/protocol/messages"
)

func TestEnvelopeRoundTripEvent(t *testing.T) {
	payload, err := json.Marshal(messages.EventPayload{
		Event: messages.EventChatSubmit,
		Data:  json.RawMessage(`{"message":"hello","ide_context":{"current_file":"a.java"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	env := &messages.Envelope{
		Protocol:    "oncode-tool/1.0",
		MessageType: messages.TypeEvent,
		MessageID:   ids.MustNew(ids.Message),
		Timestamp:   "2026-09-17T00:00:00Z",
		WorkItemID:  ids.MustNew(ids.WorkItem),
		Payload:     payload,
	}
	pb, err := ToProto(env)
	if err != nil {
		t.Fatal(err)
	}
	back, err := FromProto(pb)
	if err != nil {
		t.Fatal(err)
	}
	if back.MessageType != messages.TypeEvent || back.WorkItemID != env.WorkItemID {
		t.Fatalf("%+v", back)
	}
	var ev messages.EventPayload
	if err := json.Unmarshal(back.Payload, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Event != messages.EventChatSubmit {
		t.Fatalf("event=%s", ev.Event)
	}
}

func TestEnvelopeRoundTripToolRequest(t *testing.T) {
	payload, err := json.Marshal(messages.ToolRequestPayload{
		CallID:    ids.MustNew(ids.Call),
		Tool:      "system.ping",
		Arguments: map[string]any{},
		Options:   &messages.ToolOptions{TimeoutMS: 1000},
	})
	if err != nil {
		t.Fatal(err)
	}
	env := &messages.Envelope{
		Protocol:    "oncode-tool/1.0",
		MessageType: messages.TypeRequest,
		MessageID:   ids.MustNew(ids.Message),
		Timestamp:   "2026-09-17T00:00:00Z",
		Payload:     payload,
	}
	pb, err := ToProto(env)
	if err != nil {
		t.Fatal(err)
	}
	back, err := FromProto(pb)
	if err != nil {
		t.Fatal(err)
	}
	var req messages.ToolRequestPayload
	if err := json.Unmarshal(back.Payload, &req); err != nil {
		t.Fatal(err)
	}
	if req.Tool != "system.ping" || req.Options == nil || req.Options.TimeoutMS != 1000 {
		t.Fatalf("%+v", req)
	}
}
