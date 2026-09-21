package com.oncode.server.support;

import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.boot.testcontainers.service.connection.ServiceConnection;
import org.springframework.context.annotation.Bean;
import org.testcontainers.postgresql.PostgreSQLContainer;

/**
 * 통합 테스트용 PostgreSQL. 로컬 compose(5437)와 볼륨을 공유하지 않는다.
 */
@TestConfiguration(proxyBeanMethods = false)
public class PostgresContainerConfig {

  /**
   * Testcontainers Postgres를 띄우고 Spring DataSource에 연결한다.
   *
   * @return 테스트가 끝나면 중지되는 컨테이너
   */
  @Bean
  @ServiceConnection
  PostgreSQLContainer postgresContainer() {
    return new PostgreSQLContainer("postgres:16-alpine");
  }
}
