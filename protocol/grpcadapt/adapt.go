// Package grpcadapt는 논리 Envelope와 protobuf Envelope를 변환한다 (ADR-003).
// 도구 핸들러는 이 패키지와 proto 생성 타입에 의존하지 않는다.
package grpcadapt

import (
	"encoding/json"
	"fmt"

	toolv1 "oncode/protocol/gen/toolv1"
	protoerr "oncode/protocol/errors"
	"oncode/protocol/messages"
)

// ToProto는 도메인 Envelope를 gRPC 전송용 protobuf로 바꾼다.
// payload JSON을 message_type에 맞는 proto 필드로 옮긴다. 핸들러 DTO를 바꾸지 않는다.
func ToProto(env *messages.Envelope) (*toolv1.Envelope, error) {
	if env == nil {
		return nil, fmt.Errorf("nil envelope")
	}
	out := &toolv1.Envelope{
		Protocol:    env.Protocol,
		MessageType: toProtoMessageType(env.MessageType),
		MessageId:   env.MessageID,
		Timestamp:   env.Timestamp,
		SessionId:   env.SessionID,
		ProjectId:   env.ProjectID,
		WorkspaceId: env.WorkspaceID,
		WorkflowId:  env.WorkflowID,
		WorkItemId:  env.WorkItemID,
		TaskId:      env.TaskID,
		Actor:       toProtoActor(env.Actor),
		Target:      toProtoTarget(env.Target),
	}
	switch env.MessageType {
	case messages.TypeRequest:
		var req messages.ToolRequestPayload
		if err := json.Unmarshal(env.Payload, &req); err != nil {
			return nil, fmt.Errorf("tool request payload: %w", err)
		}
		args, err := json.Marshal(req.Arguments)
		if err != nil {
			return nil, err
		}
		timeout := int32(0)
		if req.Options != nil {
			timeout = int32(req.Options.TimeoutMS)
		}
		out.ToolRequest = &toolv1.ToolRequestPayload{
			CallId:        req.CallID,
			Tool:          req.Tool,
			ArgumentsJson: string(args),
			TimeoutMs:     timeout,
		}
	case messages.TypeResponse:
		var resp messages.ToolResponsePayload
		if err := json.Unmarshal(env.Payload, &resp); err != nil {
			return nil, fmt.Errorf("tool response payload: %w", err)
		}
		out.ToolResponse = toProtoToolResponse(resp)
	case messages.TypeEvent:
		var ev messages.EventPayload
		if err := json.Unmarshal(env.Payload, &ev); err != nil {
			return nil, fmt.Errorf("event payload: %w", err)
		}
		data := ""
		if len(ev.Data) > 0 {
			data = string(ev.Data)
		}
		out.EventPayload = &toolv1.EventPayload{
			Event:       ev.Event,
			ExecutionId: ev.ExecutionID,
			DataJson:    data,
		}
	}
	return out, nil
}

// FromProto는 protobuf Envelope를 도메인 Envelope로 되돌린다.
// 도구 핸들러가 받는 타입은 messages.Envelope다.
func FromProto(in *toolv1.Envelope) (*messages.Envelope, error) {
	if in == nil {
		return nil, fmt.Errorf("nil proto envelope")
	}
	env := &messages.Envelope{
		Protocol:    in.GetProtocol(),
		MessageType: fromProtoMessageType(in.GetMessageType()),
		MessageID:   in.GetMessageId(),
		Timestamp:   in.GetTimestamp(),
		SessionID:   in.GetSessionId(),
		ProjectID:   in.GetProjectId(),
		WorkspaceID: in.GetWorkspaceId(),
		WorkflowID:  in.GetWorkflowId(),
		WorkItemID:  in.GetWorkItemId(),
		TaskID:      in.GetTaskId(),
		Actor:       fromProtoActor(in.GetActor()),
		Target:      fromProtoTarget(in.GetTarget()),
	}
	var err error
	switch env.MessageType {
	case messages.TypeRequest:
		tr := in.GetToolRequest()
		if tr == nil {
			return nil, fmt.Errorf("REQUEST missing tool_request")
		}
		args := map[string]any{}
		if tr.GetArgumentsJson() != "" {
			if err := json.Unmarshal([]byte(tr.GetArgumentsJson()), &args); err != nil {
				return nil, fmt.Errorf("arguments_json: %w", err)
			}
		}
		req := messages.ToolRequestPayload{
			CallID:    tr.GetCallId(),
			Tool:      tr.GetTool(),
			Arguments: args,
		}
		if tr.GetTimeoutMs() > 0 {
			req.Options = &messages.ToolOptions{TimeoutMS: int(tr.GetTimeoutMs())}
		}
		env.Payload, err = json.Marshal(req)
	case messages.TypeResponse:
		env.Payload, err = json.Marshal(fromProtoToolResponse(in.GetToolResponse()))
	case messages.TypeEvent:
		ep := in.GetEventPayload()
		if ep == nil {
			return nil, fmt.Errorf("EVENT missing event_payload")
		}
		ev := messages.EventPayload{Event: ep.GetEvent(), ExecutionID: ep.GetExecutionId()}
		if ep.GetDataJson() != "" {
			ev.Data = json.RawMessage(ep.GetDataJson())
		}
		env.Payload, err = json.Marshal(ev)
	}
	if err != nil {
		return nil, err
	}
	return env, nil
}

func toProtoMessageType(v string) toolv1.MessageType {
	switch v {
	case messages.TypeRequest:
		return toolv1.MessageType_REQUEST
	case messages.TypeResponse:
		return toolv1.MessageType_RESPONSE
	case messages.TypeEvent:
		return toolv1.MessageType_EVENT
	case messages.TypeCancel:
		return toolv1.MessageType_CANCEL
	case messages.TypeAck:
		return toolv1.MessageType_ACK
	default:
		return toolv1.MessageType_MESSAGE_TYPE_UNSPECIFIED
	}
}

func fromProtoMessageType(v toolv1.MessageType) string {
	switch v {
	case toolv1.MessageType_REQUEST:
		return messages.TypeRequest
	case toolv1.MessageType_RESPONSE:
		return messages.TypeResponse
	case toolv1.MessageType_EVENT:
		return messages.TypeEvent
	case toolv1.MessageType_CANCEL:
		return messages.TypeCancel
	case toolv1.MessageType_ACK:
		return messages.TypeAck
	default:
		return ""
	}
}

func toProtoActor(a *messages.Actor) *toolv1.Actor {
	if a == nil {
		return nil
	}
	return &toolv1.Actor{Type: toProtoActorType(a.Type), Id: a.ID}
}

func fromProtoActor(a *toolv1.Actor) *messages.Actor {
	if a == nil {
		return nil
	}
	return &messages.Actor{Type: fromProtoActorType(a.GetType()), ID: a.GetId()}
}

func toProtoActorType(t messages.ActorType) toolv1.ActorType {
	switch t {
	case messages.ActorUser:
		return toolv1.ActorType_USER
	case messages.ActorAgent:
		return toolv1.ActorType_AGENT
	case messages.ActorOrchestrator:
		return toolv1.ActorType_ORCHESTRATOR
	case messages.ActorLocalAgent:
		return toolv1.ActorType_LOCAL_AGENT
	case messages.ActorIDE:
		return toolv1.ActorType_IDE
	case messages.ActorSystem:
		return toolv1.ActorType_SYSTEM
	default:
		return toolv1.ActorType_ACTOR_TYPE_UNSPECIFIED
	}
}

func fromProtoActorType(t toolv1.ActorType) messages.ActorType {
	switch t {
	case toolv1.ActorType_USER:
		return messages.ActorUser
	case toolv1.ActorType_AGENT:
		return messages.ActorAgent
	case toolv1.ActorType_ORCHESTRATOR:
		return messages.ActorOrchestrator
	case toolv1.ActorType_LOCAL_AGENT:
		return messages.ActorLocalAgent
	case toolv1.ActorType_IDE:
		return messages.ActorIDE
	case toolv1.ActorType_SYSTEM:
		return messages.ActorSystem
	default:
		return ""
	}
}

func toProtoTarget(t *messages.Target) *toolv1.Target {
	if t == nil {
		return nil
	}
	return &toolv1.Target{Type: string(t.Type), Id: t.ID}
}

func fromProtoTarget(t *toolv1.Target) *messages.Target {
	if t == nil {
		return nil
	}
	return &messages.Target{Type: messages.TargetType(t.GetType()), ID: t.GetId()}
}

func toProtoToolResponse(resp messages.ToolResponsePayload) *toolv1.ToolResponsePayload {
	out := &toolv1.ToolResponsePayload{
		CallId: resp.CallID,
		Status: toProtoStatus(resp.Status),
	}
	if len(resp.Result) > 0 {
		out.ResultJson = string(resp.Result)
	}
	if resp.Error != nil {
		details := ""
		if resp.Error.Details != nil {
			raw, _ := json.Marshal(resp.Error.Details)
			details = string(raw)
		}
		out.Error = &toolv1.ToolError{
			Code:        string(resp.Error.Code),
			Category:    string(resp.Error.Category),
			Message:     resp.Error.Message,
			Retryable:   resp.Error.Retryable,
			DetailsJson: details,
		}
	}
	if resp.Metrics != nil {
		out.Metrics = &toolv1.Metrics{DurationMs: resp.Metrics.DurationMS}
	}
	return out
}

func fromProtoToolResponse(in *toolv1.ToolResponsePayload) messages.ToolResponsePayload {
	if in == nil {
		return messages.ToolResponsePayload{}
	}
	out := messages.ToolResponsePayload{
		CallID: in.GetCallId(),
		Status: fromProtoStatus(in.GetStatus()),
	}
	if in.GetResultJson() != "" {
		out.Result = json.RawMessage(in.GetResultJson())
	}
	if err := in.GetError(); err != nil {
		var details map[string]any
		if err.GetDetailsJson() != "" {
			_ = json.Unmarshal([]byte(err.GetDetailsJson()), &details)
		}
		out.Error = &protoerr.Error{
			Code:      protoerr.Code(err.GetCode()),
			Category:  protoerr.Category(err.GetCategory()),
			Message:   err.GetMessage(),
			Retryable: err.GetRetryable(),
			Details:   details,
		}
	}
	if in.GetMetrics() != nil {
		out.Metrics = &messages.Metrics{DurationMS: in.GetMetrics().GetDurationMs()}
	}
	return out
}

func toProtoStatus(s string) toolv1.ToolStatus {
	if v, ok := toolv1.ToolStatus_value[s]; ok {
		return toolv1.ToolStatus(v)
	}
	return toolv1.ToolStatus_TOOL_STATUS_UNSPECIFIED
}

func fromProtoStatus(s toolv1.ToolStatus) string {
	if s == toolv1.ToolStatus_TOOL_STATUS_UNSPECIFIED {
		return ""
	}
	return s.String()
}
