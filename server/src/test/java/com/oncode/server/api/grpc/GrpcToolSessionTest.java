package com.oncode.server.api.grpc;

import static org.assertj.core.api.Assertions.assertThat;

import com.oncode.protocol.tool.v1.Envelope;
import com.oncode.protocol.tool.v1.MessageType;
import com.oncode.protocol.tool.v1.ToolResponsePayload;
import com.oncode.protocol.tool.v1.ToolStatus;
import com.oncode.server.toolgateway.ToolCall;
import com.oncode.server.toolgateway.ToolResponse;
import io.grpc.stub.StreamObserver;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Executors;
import org.junit.jupiter.api.Test;

/**
 * 세션 attach/RESPONSE 상관과 미연결 실패를 검증한다.
 */
class GrpcToolSessionTest {

  /**
   * 스트림이 없으면 UNAVAILABLE이고 송신하지 않는다.
   */
  @Test
  void invokeWithoutSessionIsUnavailable() {
    GrpcToolSession session = new GrpcToolSession();
    ToolResponse response =
        session.invoke(
            new ToolCall("CALL-1", "system.ping", Map.of(), 500, "ORCHESTRATOR", "s", null, null));
    assertThat(response.status()).isEqualTo("FAILED");
    assertThat(response.error().code()).isEqualTo("UNAVAILABLE");
    assertThat(session.isOpen()).isFalse();
  }

  /**
   * REQUEST를 보낸 뒤 같은 와이어 call_id RESPONSE가 오면 게이트웨이 call_id로 돌려준다.
   * 응답을 다른 스레드에서 complete해야 한다. 수신 스레드에서 invoke를 기다리면 막힌다.
   */
  @Test
  void invokeCompletesWithMatchingResponse() throws Exception {
    GrpcToolSession session = new GrpcToolSession();
    List<Envelope> sent = new ArrayList<>();
    session.attach(
        new StreamObserver<>() {
          @Override
          public void onNext(Envelope value) {
            sent.add(value);
          }

          @Override
          public void onError(Throwable t) {}

          @Override
          public void onCompleted() {}
        });
    var exec = Executors.newSingleThreadExecutor();
    var future =
        exec.submit(
            () ->
                session.invoke(
                    new ToolCall(
                        "CALL-KEEP",
                        "system.ping",
                        Map.of(),
                        500,
                        "ORCHESTRATOR",
                        "s",
                        null,
                        null)));
    Envelope request = waitForRequest(sent);
    assertThat(request.getMessageType()).isEqualTo(MessageType.REQUEST);
    assertThat(request.getToolRequest().getTool()).isEqualTo("system.ping");
    String wireId = request.getToolRequest().getCallId();
    session.complete(
        Envelope.newBuilder()
            .setMessageType(MessageType.RESPONSE)
            .setToolResponse(
                ToolResponsePayload.newBuilder()
                    .setCallId(wireId)
                    .setStatus(ToolStatus.SUCCESS)
                    .setResultJson("{\"ok\":true}")
                    .build())
            .build());
    ToolResponse response = future.get();
    exec.shutdownNow();
    assertThat(response.callId()).isEqualTo("CALL-KEEP");
    assertThat(response.status()).isEqualTo("SUCCESS");
    assertThat(response.resultJson()).contains("ok");
  }

  private static Envelope waitForRequest(List<Envelope> sent) throws InterruptedException {
    for (int i = 0; i < 50; i++) {
      if (!sent.isEmpty()) {
        return sent.get(0);
      }
      Thread.sleep(20);
    }
    throw new AssertionError("request was not sent");
  }
}
