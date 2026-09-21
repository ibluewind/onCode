package com.oncode.server.toolgateway;

import com.oncode.server.persistence.EntityIds;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

/**
 * ToolRequest 매핑, call_id 상관, timeout, 중복 side-effect 보호.
 * 워크스페이스 파일은 열지 않는다.
 */
@Service
public class DefaultToolGateway implements ToolGateway {

  static final String TOOL_APPLY = "workspace.apply_changes";
  static final String ACTOR_AGENT = "AGENT";
  private static final Set<String> SIDE_EFFECT_TOOLS =
      Set.of(
          "workspace.propose_changes",
          "workspace.apply_changes",
          "build.run",
          "test.run");

  private final LocalAgentChannel channel;
  private final int defaultTimeoutMs;
  private final ConcurrentHashMap<String, ToolResponse> sideEffectResults =
      new ConcurrentHashMap<>();
  private final ExecutorService executor = Executors.newCachedThreadPool();

  /**
   * 채널과 기본 timeout을 받는다.
   *
   * @param channel Local Agent 채널. null 불가
   * @param defaultTimeoutMs 기본 제한(ms). 1 이상
   */
  public DefaultToolGateway(
      LocalAgentChannel channel,
      @Value("${oncode.tool-gateway.default-timeout-ms:5000}") int defaultTimeoutMs) {
    this.channel = channel;
    this.defaultTimeoutMs = defaultTimeoutMs;
  }

  /**
   * {@inheritDoc}
   */
  @Override
  public ToolResponse call(ToolCall call) {
    Objects.requireNonNull(call, "call");
    if (call.tool() == null || call.tool().isBlank()) {
      throw new IllegalArgumentException("tool must not be blank");
    }
    String callId =
        (call.callId() == null || call.callId().isBlank()) ? EntityIds.call() : call.callId();
    ToolCall normalized =
        new ToolCall(
            callId,
            call.tool(),
            call.arguments() == null ? Map.of() : call.arguments(),
            call.timeoutMs(),
            call.actorType(),
            call.actorId(),
            call.workflowId(),
            call.workItemId());
    if (ACTOR_AGENT.equals(normalized.actorType()) && TOOL_APPLY.equals(normalized.tool())) {
      return ToolResponse.failed(
          callId,
          "FAILED",
          new ToolError(
              "PERMISSION_DENIED",
              "PERMISSION",
              "agents must not call workspace.apply_changes",
              false),
          0);
    }
    if (SIDE_EFFECT_TOOLS.contains(normalized.tool())) {
      ToolResponse cached = sideEffectResults.get(callId);
      if (cached != null) {
        return cached;
      }
    }
    int timeout =
        normalized.timeoutMs() == null || normalized.timeoutMs() < 1
            ? defaultTimeoutMs
            : normalized.timeoutMs();
    long start = System.nanoTime();
    var future = executor.submit(() -> channel.invoke(normalized));
    try {
      ToolResponse response = future.get(timeout, TimeUnit.MILLISECONDS);
      ToolResponse correlated =
          new ToolResponse(
              callId,
              response.status(),
              response.resultJson(),
              response.error(),
              durationMs(start));
      rememberSideEffect(normalized.tool(), callId, correlated);
      return correlated;
    } catch (TimeoutException timeoutEx) {
      future.cancel(true);
      ToolResponse timedOut =
          ToolResponse.failed(
              callId,
              "TIMEOUT",
              new ToolError("TIMEOUT", "TIMEOUT", "tool call exceeded timeout_ms=" + timeout, true),
              durationMs(start));
      rememberSideEffect(normalized.tool(), callId, timedOut);
      return timedOut;
    } catch (InterruptedException interrupted) {
      Thread.currentThread().interrupt();
      return ToolResponse.failed(
          callId,
          "FAILED",
          new ToolError("INTERNAL_ERROR", "SYSTEM", "tool call interrupted", false),
          durationMs(start));
    } catch (ExecutionException execution) {
      Throwable cause = execution.getCause() == null ? execution : execution.getCause();
      return ToolResponse.failed(
          callId,
          "FAILED",
          new ToolError("INTERNAL_ERROR", "SYSTEM", cause.getMessage(), false),
          durationMs(start));
    }
  }

  private void rememberSideEffect(String tool, String callId, ToolResponse response) {
    if (SIDE_EFFECT_TOOLS.contains(tool)) {
      sideEffectResults.putIfAbsent(callId, response);
    }
  }

  private static long durationMs(long startNanos) {
    return Math.max(0, (System.nanoTime() - startNanos) / 1_000_000);
  }
}
