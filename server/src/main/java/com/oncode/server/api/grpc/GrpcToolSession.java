package com.oncode.server.api.grpc;

import com.oncode.protocol.tool.v1.Actor;
import com.oncode.protocol.tool.v1.ActorType;
import com.oncode.protocol.tool.v1.Envelope;
import com.oncode.protocol.tool.v1.MessageType;
import com.oncode.protocol.tool.v1.ToolError;
import com.oncode.protocol.tool.v1.ToolRequestPayload;
import com.oncode.protocol.tool.v1.ToolResponsePayload;
import com.oncode.server.toolgateway.ToolCall;
import com.oncode.server.toolgateway.ToolResponse;
import io.grpc.stub.StreamObserver;
import java.time.Instant;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import org.springframework.stereotype.Component;

/**
 * 연결된 Local Agent 스트림에 TOOL REQUEST를 쓰고 RESPONSE를 call_id로 받는다.
 * 워크플로 상태·파일 I/O는 없고, 세션이 없으면 호출하지 않는다.
 */
@Component
public class GrpcToolSession {

  private final Object lock = new Object();
  private StreamObserver<Envelope> outbound;
  private final ConcurrentHashMap<String, CompletableFuture<ToolResponse>> pending =
      new ConcurrentHashMap<>();

  /**
   * Open 스트림의 서버→에이전트 송신자를 붙인다. 이전 세션이 있으면 교체한다.
   *
   * @param out null이면 무시
   */
  public void attach(StreamObserver<Envelope> out) {
    if (out == null) {
      return;
    }
    synchronized (lock) {
      this.outbound = out;
    }
  }

  /**
   * 스트림이 끝나면 대기 중인 호출을 실패시키고 세션을 비운다.
   */
  public void detach() {
    synchronized (lock) {
      this.outbound = null;
    }
    pending.forEach(
        (id, future) ->
            future.complete(
                ToolResponse.failed(
                    id,
                    "FAILED",
                    new com.oncode.server.toolgateway.ToolError(
                        "UNAVAILABLE", "SYSTEM", "local agent disconnected", true),
                    0)));
    pending.clear();
  }

  /**
   * 에이전트 스트림이 붙어 있는지 본다.
   *
   * @return 붙어 있으면 true
   */
  public boolean isOpen() {
    synchronized (lock) {
      return outbound != null;
    }
  }

  /**
   * REQUEST/EVENT를 같은 스트림에 직렬로 보낸다. StreamObserver는 스레드 안전하지 않다.
   *
   * @param env null이면 무시
   */
  public void send(Envelope env) {
    if (env == null) {
      return;
    }
    synchronized (lock) {
      if (outbound == null) {
        return;
      }
      outbound.onNext(env);
    }
  }

  /**
   * REQUEST를 보내고 같은 와이어 call_id의 RESPONSE를 돌려준다. 타임아웃은 게이트웨이가 자른다.
   *
   * @param call call_id·tool이 채워진 호출. null 불가
   * @return 상관된 응답. 게이트웨이 call_id를 유지
   */
  public ToolResponse invoke(ToolCall call) {
    if (!isOpen()) {
      return ToolResponse.failed(
          call.callId(),
          "FAILED",
          new com.oncode.server.toolgateway.ToolError(
              "UNAVAILABLE", "SYSTEM", "local agent is not connected", true),
          0);
    }
    String wireCallId = ProtocolIds.call();
    CompletableFuture<ToolResponse> future = new CompletableFuture<>();
    pending.put(wireCallId, future);
    try {
      send(requestEnvelope(call, wireCallId));
      ToolResponse wire = future.get();
      return new ToolResponse(
          call.callId(), wire.status(), wire.resultJson(), wire.error(), wire.durationMs());
    } catch (InterruptedException interrupted) {
      Thread.currentThread().interrupt();
      pending.remove(wireCallId);
      return ToolResponse.failed(
          call.callId(),
          "FAILED",
          new com.oncode.server.toolgateway.ToolError(
              "INTERNAL_ERROR", "SYSTEM", "tool call interrupted", false),
          0);
    } catch (java.util.concurrent.ExecutionException | RuntimeException ex) {
      pending.remove(wireCallId);
      return ToolResponse.failed(
          call.callId(),
          "FAILED",
          new com.oncode.server.toolgateway.ToolError(
              "INTERNAL_ERROR",
              "SYSTEM",
              ex.getMessage() == null ? ex.getClass().getSimpleName() : ex.getMessage(),
              false),
          0);
    }
  }

  /**
   * 에이전트가 보낸 RESPONSE Envelope를 대기 중 호출에 연결한다. EVENT는 무시한다.
   *
   * @param incoming proto Envelope. RESPONSE가 아니면 false
   * @return 처리했으면 true
   */
  public boolean complete(Envelope incoming) {
    if (incoming == null || incoming.getMessageType() != MessageType.RESPONSE) {
      return false;
    }
    ToolResponsePayload payload = incoming.getToolResponse();
    if (payload == null || payload.getCallId().isBlank()) {
      return false;
    }
    CompletableFuture<ToolResponse> future = pending.remove(payload.getCallId());
    if (future == null) {
      return false;
    }
    future.complete(fromProto(payload));
    return true;
  }

  private static Envelope requestEnvelope(ToolCall call, String wireCallId) {
    int timeout = call.timeoutMs() == null ? 0 : call.timeoutMs();
    return Envelope.newBuilder()
        .setProtocol("oncode-tool/1.0")
        .setMessageType(MessageType.REQUEST)
        .setMessageId(ProtocolIds.message())
        .setTimestamp(Instant.now().toString())
        .setActor(
            Actor.newBuilder()
                .setType(actorType(call.actorType()))
                .setId(call.actorId() == null ? "" : call.actorId())
                .build())
        .setToolRequest(
            ToolRequestPayload.newBuilder()
                .setCallId(wireCallId)
                .setTool(call.tool())
                .setArgumentsJson(EventJson.write(call.arguments()))
                .setTimeoutMs(timeout)
                .build())
        .build();
  }

  private static ActorType actorType(String raw) {
    if (raw == null) {
      return ActorType.ACTOR_TYPE_UNSPECIFIED;
    }
    return switch (raw) {
      case "USER" -> ActorType.USER;
      case "AGENT" -> ActorType.AGENT;
      case "ORCHESTRATOR" -> ActorType.ORCHESTRATOR;
      case "LOCAL_AGENT" -> ActorType.LOCAL_AGENT;
      case "IDE" -> ActorType.IDE;
      case "SYSTEM" -> ActorType.SYSTEM;
      default -> ActorType.ACTOR_TYPE_UNSPECIFIED;
    };
  }

  private static ToolResponse fromProto(ToolResponsePayload payload) {
    String status = payload.getStatus().name();
    if ("TOOL_STATUS_UNSPECIFIED".equals(status) || status.isBlank()) {
      status = "FAILED";
    }
    if (!payload.hasError()) {
      String json = payload.getResultJson();
      return ToolResponse.success(
          payload.getCallId(), json == null || json.isBlank() ? "{}" : json, 0);
    }
    ToolError err = payload.getError();
    return ToolResponse.failed(
        payload.getCallId(),
        status,
        new com.oncode.server.toolgateway.ToolError(
            err.getCode(), err.getCategory(), err.getMessage(), err.getRetryable()),
        payload.hasMetrics() ? payload.getMetrics().getDurationMs() : 0);
  }
}
