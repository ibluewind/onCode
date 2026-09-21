package com.oncode.server;

import static org.assertj.core.api.Assertions.assertThat;

import com.oncode.server.api.grpc.AgentGrpcServer;
import com.oncode.server.support.PostgresSpringBootTest;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.ApplicationContext;

@PostgresSpringBootTest
class OnCodeServerApplicationTest {

  @Autowired private ApplicationContext context;
  @Autowired private AgentGrpcServer agentGrpcServer;

  /**
   * Flyway와 JDBC가 붙은 채로 컨텍스트가 뜨는지 확인한다.
   */
  @Test
  void contextLoads() {
    assertThat(context).isNotNull();
  }

  /**
   * Local Agent용 gRPC가 테스트에서 ephemeral 포트로 listen하는지 확인한다.
   */
  @Test
  void agentGrpcListensOnEphemeralPort() {
    assertThat(agentGrpcServer.getPort()).isGreaterThan(0);
  }

  /**
   * PHASE_02 모듈 설정 빈 8개가 등록됐는지 확인한다.
   */
  @Test
  void modularMonolithModulesAreRegistered() {
    for (Class<?> module : ServerModules.CONFIGURATIONS) {
      assertThat(context.getBean(module)).isNotNull();
    }
    assertThat(ServerModules.CONFIGURATIONS).hasSize(8);
  }
}
