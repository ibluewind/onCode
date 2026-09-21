package com.oncode.server.toolgateway;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.Map;
import java.util.concurrent.atomic.AtomicInteger;
import org.junit.jupiter.api.Test;

/**
 * timeout, AGENT apply 거부, 중복 call_id side-effect 보호를 검증한다.
 */
class DefaultToolGatewayTest {

  /**
   * Phase 1 ping이 하니스를 통해 성공한다.
   */
  @Test
  void pingUsesPhase1ToolName() {
    InMemoryLocalAgentChannel harness = new InMemoryLocalAgentChannel();
    DefaultToolGateway gateway = new DefaultToolGateway(harness, 1_000);
    ToolResponse response =
        gateway.call(
            new ToolCall(null, "system.ping", Map.of(), 500, "ORCHESTRATOR", "server", null, null));
    assertThat(response.status()).isEqualTo("SUCCESS");
    assertThat(response.callId()).startsWith("CALL-");
    assertThat(response.resultJson()).contains("oncode-tool/1.0");
    assertThat(harness.invokedTools()).containsExactly("system.ping");
  }

  /**
   * timeout_ms를 넘기면 structured TIMEOUT 오류가 난다.
   */
  @Test
  void timeoutReturnsStructuredError() {
    DefaultToolGateway gateway =
        new DefaultToolGateway(
            call -> {
              try {
                Thread.sleep(5_000);
              } catch (InterruptedException ignored) {
                Thread.currentThread().interrupt();
              }
              return ToolResponse.success(call.callId(), "{}", 0);
            },
            5_000);
    ToolResponse response =
        gateway.call(
            new ToolCall("CALL-TIMEOUT1", "system.ping", Map.of(), 50, "AGENT", "a", null, null));
    assertThat(response.status()).isEqualTo("TIMEOUT");
    assertThat(response.error().code()).isEqualTo("TIMEOUT");
    assertThat(response.error().retryable()).isTrue();
    assertThat(response.error().category()).isEqualTo("TIMEOUT");
  }

  /**
   * AGENT는 apply를 채널까지 보내지 못한다.
   */
  @Test
  void agentApplyIsDenied() {
    InMemoryLocalAgentChannel harness = new InMemoryLocalAgentChannel();
    DefaultToolGateway gateway = new DefaultToolGateway(harness, 1_000);
    ToolResponse response =
        gateway.call(
            new ToolCall(
                "CALL-APPLY1",
                "workspace.apply_changes",
                Map.of(),
                500,
                "AGENT",
                "implementation-agent",
                "WF-1",
                "WI-1"));
    assertThat(response.status()).isEqualTo("FAILED");
    assertThat(response.error().code()).isEqualTo("PERMISSION_DENIED");
    assertThat(harness.invokedTools()).isEmpty();
  }

  /**
   * 같은 call_id의 propose는 한 번만 실행한다.
   */
  @Test
  void duplicateSideEffectCallIsNotReplayed() {
    AtomicInteger invokes = new AtomicInteger();
    DefaultToolGateway gateway =
        new DefaultToolGateway(
            call -> {
              invokes.incrementAndGet();
              return ToolResponse.success(call.callId(), "{\"n\":" + invokes.get() + "}", 0);
            },
            1_000);
    ToolCall first =
        new ToolCall(
            "CALL-DUP1",
            "workspace.propose_changes",
            Map.of("path", "a.java"),
            500,
            "AGENT",
            "implementation-agent",
            null,
            null);
    ToolResponse one = gateway.call(first);
    ToolResponse two = gateway.call(first);
    assertThat(invokes.get()).isEqualTo(1);
    assertThat(two.resultJson()).isEqualTo(one.resultJson());
  }
}
