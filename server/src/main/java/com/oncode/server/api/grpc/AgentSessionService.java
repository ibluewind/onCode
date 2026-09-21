package com.oncode.server.api.grpc;

import com.oncode.protocol.tool.v1.Envelope;
import com.oncode.protocol.tool.v1.EventPayload;
import com.oncode.protocol.tool.v1.LocalAgentSessionGrpc;
import com.oncode.protocol.tool.v1.MessageType;
import com.oncode.server.api.ApprovalIntake;
import com.oncode.server.api.ApprovalView;
import com.oncode.server.api.ChatIntake;
import com.oncode.server.api.ChatOutcome;
import com.oncode.server.api.ProgressEvent;
import com.oncode.server.api.Question;
import com.oncode.server.api.QuestionOption;
import com.oncode.server.api.RunResult;
import io.grpc.stub.StreamObserver;
import java.time.Instant;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import org.springframework.boot.json.JsonParserFactory;
import org.springframework.stereotype.Service;

/**
 * Local Agent outbound 스트림을 받는다. Envelope만 다루며 ChatIntake에 proto를 넘기지 않는다.
 * EVENT 처리는 수신 스레드 밖에서 돌린다. 같은 스트림의 TOOL RESPONSE가 막히지 않게 한다.
 */
@Service
public class AgentSessionService extends LocalAgentSessionGrpc.LocalAgentSessionImplBase {

  private final ChatIntake intake;
  private final ApprovalIntake approvals;
  private final GrpcToolSession tools;
  private final ConcurrentHashMap<String, Map<String, Object>> pendingChats =
      new ConcurrentHashMap<>();
  private final ExecutorService events = Executors.newCachedThreadPool();

  /**
   * 채팅·승인 도메인과 도구 세션을 받는다.
   *
   * @param intake proto를 모르는 채팅 접수. null 불가
   * @param approvals 결정·복원·diff 바인딩. null 불가
   * @param tools REQUEST/RESPONSE 상관. null 불가
   */
  public AgentSessionService(ChatIntake intake, ApprovalIntake approvals, GrpcToolSession tools) {
    this.intake = intake;
    this.approvals = approvals;
    this.tools = tools;
  }

  /**
   * 양방향 Envelope 스트림. Local Agent가 연다.
   *
   * @param responseObserver 서버→에이전트 송신
   */
  @Override
  public StreamObserver<Envelope> open(StreamObserver<Envelope> responseObserver) {
    tools.attach(responseObserver);
    return new StreamObserver<>() {
      @Override
      public void onNext(Envelope value) {
        if (tools.complete(value)) {
          return;
        }
        events.execute(() -> handle(value));
      }

      @Override
      public void onError(Throwable t) {
        tools.detach();
      }

      @Override
      public void onCompleted() {
        tools.detach();
        responseObserver.onCompleted();
      }
    };
  }

  /**
   * EVENT를 처리한다. RESPONSE는 {@link #open}에서 이미 상관했으므로 여기 두지 않는다.
   * 도구 왕복을 이 스레드에서 기다려도 gRPC 수신 스레드는 비어 있다.
   *
   * @param incoming 에이전트가 보낸 Envelope
   */
  private void handle(Envelope incoming) {
    try {
      handleEvent(incoming);
    } catch (RuntimeException ex) {
      tools.send(
          errorEvent(
              "INTERNAL",
              ex.getMessage() == null ? ex.getClass().getSimpleName() : ex.getMessage()));
    }
  }

  /**
   * EVENT 종류별로 채팅·승인 도메인에 넘긴다. proto 생성 타입은 이 메서드 안에서만 본다.
   *
   * @param incoming 에이전트가 보낸 Envelope. EVENT가 아니면 무시
   */
  private void handleEvent(Envelope incoming) {
    if (incoming.getMessageType() != MessageType.EVENT || !incoming.hasEventPayload()) {
      return;
    }
    EventPayload ev = incoming.getEventPayload();
    switch (ev.getEvent()) {
      case "chat.submit" -> onChat(ev.getDataJson());
      case "question.answer" -> onAnswer(ev.getDataJson());
      case "approval.decide" -> emitOutcome(Map.of(), approvals.decide(parseMap(ev.getDataJson())));
      case "approval.bind" -> emitOutcome(Map.of(), approvals.bindDiff(parseMap(ev.getDataJson())));
      case "apply.completed" ->
          emitOutcome(Map.of(), approvals.applyCompleted(parseMap(ev.getDataJson())));
      case "session.restore" ->
          emitOutcome(
              Map.of(),
              approvals.restore(ApprovalIntake.text(parseMap(ev.getDataJson()), "work_item_id")));
      default -> {
        // heartbeat / register
      }
    }
  }

  private void onChat(String dataJson) {
    Map<String, Object> data = parseMap(dataJson);
    emitOutcome(data, intake.submit(data));
  }

  private void onAnswer(String dataJson) {
    Map<String, Object> data = parseMap(dataJson);
    String questionId = String.valueOf(data.getOrDefault("question_id", ""));
    String answer = String.valueOf(data.getOrDefault("answer", "")).trim();
    Map<String, Object> original = pendingChats.remove(questionId);
    if (original == null || answer.isEmpty()) {
      return;
    }
    Map<String, Object> merged = new HashMap<>(original);
    Map<String, Object> ctx = new HashMap<>();
    ctx.put("current_file", answer);
    merged.put("ide_context", ctx);
    emitOutcome(merged, intake.submit(merged));
  }

  private void emitOutcome(Map<String, Object> original, ChatOutcome outcome) {
    switch (outcome) {
      case ChatOutcome.Rejected rejected ->
          tools.send(errorEvent(rejected.code(), rejected.message()));
      case ChatOutcome.NeedQuestion need -> {
        pendingChats.put(need.question().questionId(), original == null ? Map.of() : original);
        tools.send(questionEvent(need.question()));
      }
      case ChatOutcome.Accepted accepted -> {
        for (ProgressEvent ev : accepted.progress()) {
          tools.send(progressEvent(accepted.workItemId(), accepted.workflowId(), ev));
        }
        if (accepted.results() != null) {
          for (RunResult result : accepted.results()) {
            tools.send(runResultEvent(accepted.workItemId(), accepted.workflowId(), result));
          }
        }
        emitPending(accepted);
      }
    }
  }

  private void emitPending(ChatOutcome.Accepted accepted) {
    ApprovalView pending = accepted.pending();
    if (pending == null) {
      return;
    }
    String event = ApprovalView.DESIGN.equals(pending.kind()) ? "design.review" : "change.proposed";
    tools.send(eventEnvelope(event, wire(pending), pending.workItemId(), pending.workflowId()));
  }

  private static Map<String, Object> wire(ApprovalView view) {
    Map<String, Object> data = new LinkedHashMap<>();
    data.put("kind", view.kind());
    data.put("approval_id", view.approvalId());
    data.put("work_item_id", view.workItemId());
    data.put("workflow_id", view.workflowId());
    data.put("workspace_id", view.workspaceId());
    data.put("resource_type", view.resourceType());
    data.put("resource_id", view.resourceId());
    data.put("can_apply", false);
    if (view.designIdentity() != null) {
      data.put("design_identity", view.designIdentity());
    }
    if (view.designSummary() != null) {
      data.put("design_summary", view.designSummary());
    }
    if (view.proposedChangesJson() != null) {
      data.put("proposed_changes_json", view.proposedChangesJson());
    }
    return data;
  }

  private Envelope progressEvent(String workItemId, String workflowId, ProgressEvent ev) {
    Map<String, Object> data = new LinkedHashMap<>();
    data.put("stage", ev.stage());
    data.put("status", ev.status());
    data.put("message", ev.message());
    return eventEnvelope("workflow.progress", data, workItemId, workflowId);
  }

  private Envelope runResultEvent(String workItemId, String workflowId, RunResult result) {
    String event = RunResult.TEST.equals(result.kind()) ? "test.result" : "build.result";
    Map<String, Object> data = new LinkedHashMap<>();
    data.put("kind", result.kind());
    data.put("status", result.status());
    data.put("summary", result.summary());
    return eventEnvelope(event, data, workItemId, workflowId);
  }

  private Envelope questionEvent(Question q) {
    List<Map<String, Object>> options = new ArrayList<>();
    for (QuestionOption opt : q.options()) {
      Map<String, Object> row = new LinkedHashMap<>();
      row.put("id", opt.id());
      row.put("label", opt.label());
      options.add(row);
    }
    Map<String, Object> data = new LinkedHashMap<>();
    data.put("question_id", q.questionId());
    data.put("question_type", q.questionType());
    data.put("prompt", q.prompt());
    data.put("options", options);
    return eventEnvelope("question.ask", data, "", "");
  }

  private Envelope errorEvent(String code, String message) {
    Map<String, Object> data = new LinkedHashMap<>();
    data.put("code", code);
    data.put("message", message);
    return eventEnvelope("error", data, "", "");
  }

  private Envelope eventEnvelope(
      String event, Map<String, Object> data, String workItemId, String workflowId) {
    return Envelope.newBuilder()
        .setProtocol("oncode-tool/1.0")
        .setMessageType(MessageType.EVENT)
        .setMessageId("MSG-" + UUID.randomUUID())
        .setTimestamp(Instant.now().toString())
        .setWorkItemId(workItemId == null ? "" : workItemId)
        .setWorkflowId(workflowId == null ? "" : workflowId)
        .setEventPayload(
            EventPayload.newBuilder().setEvent(event).setDataJson(EventJson.write(data)).build())
        .build();
  }

  private static Map<String, Object> parseMap(String dataJson) {
    if (dataJson == null || dataJson.isBlank()) {
      return Map.of();
    }
    return JsonParserFactory.getJsonParser().parseMap(dataJson);
  }
}
