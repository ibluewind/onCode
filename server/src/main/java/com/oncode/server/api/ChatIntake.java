package com.oncode.server.api;

import com.oncode.server.agentruntime.AgentWorkflowLoop;
import com.oncode.server.persistence.WorkflowRecord;
import com.oncode.server.persistence.WorkflowRepository;
import com.oncode.server.workflow.WorkflowEngine;
import com.oncode.server.workflow.WorkflowNotFoundException;
import com.oncode.server.workflow.WorkflowState;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * IDE 채팅을 work item으로 만들고 진행 이벤트를 만든다.
 * 소스 본문은 받지 않으며, proto/gRPC 타입에 의존하지 않는다.
 */
@Service
public class ChatIntake {

  private final AgentWorkflowLoop loop;
  private final WorkflowEngine engine;
  private final WorkflowRepository workflows;
  private final ApprovalIntake approvals;

  /**
   * 워크플로 루프와 조회 포트를 받는다.
   *
   * @param loop 설계 대기까지 구동. null 불가
   * @param engine 재개 조회. null 불가
   * @param workflows work item → workflow. null 불가
   * @param approvals 대기 승인 뷰. null 불가
   */
  public ChatIntake(
      AgentWorkflowLoop loop,
      WorkflowEngine engine,
      WorkflowRepository workflows,
      ApprovalIntake approvals) {
    this.loop = loop;
    this.engine = engine;
    this.workflows = workflows;
    this.approvals = approvals;
  }

  /**
   * chat.submit data JSON을 접수한다.
   *
   * @param dataJson object JSON. 비면 거부
   * @return 생성/재개/질문/거부
   */
  @Transactional
  public ChatOutcome submit(String dataJson) {
    Map<String, Object> data;
    try {
      data = IdeContextGuard.parseObject(dataJson);
    } catch (RuntimeException ex) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", ex.getMessage());
    }
    return submit(data);
  }

  /**
   * 이미 파싱된 chat.submit 맵을 접수한다. gRPC 레이어가 질문 답변을 합친 뒤 다시 넣을 때 쓴다.
   *
   * @param data message 필수. null 불가
   */
  @Transactional
  public ChatOutcome submit(Map<String, Object> data) {
    try {
      IdeContextGuard.rejectSourceBody(data);
    } catch (RuntimeException ex) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", ex.getMessage());
    }
    Object messageObj = data.get("message");
    String message = messageObj == null ? "" : String.valueOf(messageObj).trim();
    if (message.isEmpty()) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "message is required");
    }
    Object resumeId = data.get("work_item_id");
    if (resumeId != null && !String.valueOf(resumeId).isBlank()) {
      return resume(String.valueOf(resumeId).trim());
    }
    String currentFile = IdeContextGuard.currentFile(data);
    if (currentFile == null) {
      return new ChatOutcome.NeedQuestion(IdeContextGuard.typedQuestion(Question.TEXT));
    }
    WorkflowRecord waiting = loop.startDesignApproval(message);
    approvals.rememberWorkspace(waiting.workItemId(), ApprovalIntake.text(data, "workspace_id"));
    return new ChatOutcome.Accepted(
        waiting.workItemId(),
        waiting.workflowId(),
        progressThroughDesign(waiting),
        approvals.viewFor(waiting),
        List.of());
  }

  /**
   * 기존 work item의 현재 상태를 진행 이벤트 하나로 돌려준다.
   *
   * @param workItemId WI-...
   */
  private ChatOutcome resume(String workItemId) {
    WorkflowRecord row =
        workflows
            .findByWorkItemId(workItemId)
            .orElse(null);
    if (row == null) {
      return new ChatOutcome.Rejected("NOT_FOUND", "work item not found");
    }
    try {
      row = engine.resumeAfterRestart(row.workflowId());
    } catch (WorkflowNotFoundException ex) {
      return new ChatOutcome.Rejected("NOT_FOUND", ex.getMessage());
    }
    return new ChatOutcome.Accepted(
        row.workItemId(),
        row.workflowId(),
        List.of(progressOf(row.currentState())),
        approvals.viewFor(row),
        List.of());
  }

  private static List<ProgressEvent> progressThroughDesign(WorkflowRecord waiting) {
    List<ProgressEvent> events = new ArrayList<>();
    events.add(progressOf(WorkflowState.RECEIVED));
    events.add(progressOf(WorkflowState.CLASSIFYING));
    events.add(progressOf(WorkflowState.DISCOVERING_CONTEXT));
    events.add(progressOf(WorkflowState.DESIGNING));
    events.add(progressOf(waiting.currentState()));
    return events;
  }

  /**
   * 상태 이름을 사용자 문구로 바꾼다. Agent CoT는 넣지 않는다.
   *
   * @param state 공식 상태
   */
  public static ProgressEvent progressOf(WorkflowState state) {
    String status =
        state.isWaiting()
            ? "WAITING"
            : (state.isCompleted() ? "COMPLETED" : (state.isTerminal() ? "FAILED" : "RUNNING"));
    String message =
        switch (state) {
          case RECEIVED -> "요청을 받았습니다.";
          case CLASSIFYING -> "요청을 분류하고 있습니다.";
          case DISCOVERING_CONTEXT -> "관련 맥락을 찾고 있습니다.";
          case DESIGNING -> "구현 설계를 작성하고 있습니다.";
          case WAITING_DESIGN_APPROVAL -> "설계 승인을 기다리고 있습니다.";
          case IMPLEMENTING -> "코드를 제안하고 있습니다.";
          case REVIEWING -> "변경을 검토하고 있습니다.";
          case PREPARING_CHANGE -> "변경 세트를 준비하고 있습니다.";
          case WAITING_CODE_APPROVAL -> "코드 승인을 기다리고 있습니다.";
          case APPLYING_CHANGE -> "승인된 변경을 적용하는 단계입니다.";
          case COMPLETED -> "변경 적용이 완료되었습니다.";
          case REJECTED -> "요청이 거절되었습니다.";
        };
    return new ProgressEvent(state.name(), status, message);
  }
}
