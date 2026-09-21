package com.oncode.server.support;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.context.annotation.Import;

/**
 * PostgreSQL Testcontainers가 붙은 서버 통합 테스트.
 */
@Target(ElementType.TYPE)
@Retention(RetentionPolicy.RUNTIME)
@SpringBootTest(properties = "oncode.grpc.port=0")
@Import(PostgresContainerConfig.class)
public @interface PostgresSpringBootTest {}
