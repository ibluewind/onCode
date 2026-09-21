package com.oncode.server.api.grpc;

import io.grpc.Server;
import io.grpc.netty.shaded.io.grpc.netty.NettyServerBuilder;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.util.concurrent.TimeUnit;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

/**
 * Local Agent가 붙는 gRPC 서버. HTTP 포트와 분리한다. WebSocket 전송은 두지 않는다.
 */
@Component
public class AgentGrpcServer implements org.springframework.context.SmartLifecycle {

  private final Server server;
  private volatile boolean running;

  /**
   * loopback에만 바인딩한다. port 0이면 운영체제가 고른다.
   *
   * @param service Envelope 스트림 구현
   * @param port listen 포트. 0–65535
   */
  public AgentGrpcServer(
      AgentSessionService service, @Value("${oncode.grpc.port:9443}") int port) {
    this.server =
        NettyServerBuilder.forAddress(new InetSocketAddress("127.0.0.1", port))
            .addService(service)
            .build();
  }

  /**
   * 실제 listen 포트. start 전에는 -1일 수 있다.
   */
  public int getPort() {
    return server.getPort();
  }

  @Override
  public void start() {
    try {
      server.start();
      running = true;
    } catch (IOException e) {
      throw new IllegalStateException("gRPC listen failed", e);
    }
  }

  @Override
  public void stop() {
    server.shutdown();
    try {
      server.awaitTermination(2, TimeUnit.SECONDS);
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
    }
    running = false;
  }

  @Override
  public boolean isRunning() {
    return running;
  }
}
