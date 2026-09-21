package com.oncode.server.toolgateway;

import static org.assertj.core.api.Assertions.assertThat;

import com.oncode.server.api.grpc.GrpcToolSession;
import java.util.Map;
import org.junit.jupiter.api.Test;

/**
 * 미연결이면 하니스, 연결이면 gRPC 세션으로 가는지 본다.
 */
class RoutedLocalAgentChannelTest {

  /**
   * 세션이 닫혀 있으면 인메모리 ping이 기록된다.
   */
  @Test
  void closedSessionUsesMemoryHarness() {
    InMemoryLocalAgentChannel memory = new InMemoryLocalAgentChannel();
    RoutedLocalAgentChannel routed = new RoutedLocalAgentChannel(memory, new GrpcToolSession());
    ToolResponse response =
        routed.invoke(
            new ToolCall(null, "system.ping", Map.of(), 500, "ORCHESTRATOR", "s", null, null));
    assertThat(response.status()).isEqualTo("SUCCESS");
    assertThat(memory.invokedTools()).containsExactly("system.ping");
  }
}
