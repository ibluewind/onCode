package com.oncode.server.toolgateway;

import com.oncode.server.api.grpc.GrpcToolSession;
import org.springframework.context.annotation.Primary;
import org.springframework.stereotype.Service;

/**
 * 에이전트 스트림이 열려 있으면 gRPC REQUEST/RESPONSE, 아니면 인메모리 하니스.
 * 게이트웨이 정책(apply 거부·timeout)은 밖에 있다.
 */
@Service
@Primary
public class RoutedLocalAgentChannel implements LocalAgentChannel {

  private final InMemoryLocalAgentChannel memory;
  private final GrpcToolSession session;

  /**
   * 하니스와 gRPC 세션을 받는다.
   *
   * @param memory 미연결 시 사용. null 불가
   * @param session 연결 세션. null 불가
   */
  public RoutedLocalAgentChannel(InMemoryLocalAgentChannel memory, GrpcToolSession session) {
    this.memory = memory;
    this.session = session;
  }

  /**
   * {@inheritDoc}
   */
  @Override
  public ToolResponse invoke(ToolCall call) {
    if (session.isOpen()) {
      return session.invoke(call);
    }
    return memory.invoke(call);
  }
}
