package com.oncode.server.api;

import com.oncode.server.agentruntime.AgentWorkflowLoop;
import com.oncode.server.approval.ApprovalDecision;
import com.oncode.server.approval.ApprovalExpiredException;
import com.oncode.server.approval.ApprovalNotFoundException;
import com.oncode.server.approval.ApprovalPort;
import com.oncode.server.approval.ApprovalRecord;
import com.oncode.server.approval.ApprovalResourceMismatchException;
import com.oncode.server.approval.ApprovalReuseException;
import com.oncode.server.approval.ApprovalStatus;
import com.oncode.server.approval.ApprovalType;
import com.oncode.server.context.ContextService;
import com.oncode.server.persistence.WorkflowRecord;
import com.oncode.server.persistence.WorkflowRepository;
import com.oncode.server.workflow.IllegalWorkflowTransitionException;
import com.oncode.server.workflow.WorkflowEngine;
import com.oncode.server.workflow.WorkflowNotFoundException;
import com.oncode.server.workflow.WorkflowState;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.springframework.boot.json.JsonParserFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * IDE 승인 결정과 재연결 스냅샷. proto/gRPC 타입에 의존하지 않으며 apply를 호출하지 않는다.
 */
@Service
public class ApprovalIntake {

  static final String DEFAULT_WORKSPACE = "ws-local";
  static final String REF_WORKSPACE = "WORKSPACE";
  static final String REF_BIND = "APPROVAL_BIND";

  private final AgentWorkflowLoop loop;
  private final WorkflowEngine engine;
  private final WorkflowRepository workflows;
  private final ApprovalPort approvals;
  private final ContextService contexts;

  /**
   * 워크플로·승인·컨텍스트 포트를 받는다.
   *
   * @param loop 설계 승인 후 구현 루프. null 불가
   * @param engine 결정 전이. null 불가
   * @param workflows work item 조회. null 불가
   * @param approvals 대기 승인. null 불가
   * @param contexts 바인딩 스냅샷. null 불가
   */
  public ApprovalIntake(
      AgentWorkflowLoop loop,
      WorkflowEngine engine,
      WorkflowRepository workflows,
      ApprovalPort approvals,
      ContextService contexts) {
    this.loop = loop;
    this.engine = engine;
    this.workflows = workflows;
    this.approvals = approvals;
    this.contexts = contexts;
  }

  /**
   * 채팅으로 만든 work item에 workspace_id를 붙인다. 이미 있으면 덮어쓰지 않는다.
   *
   * @param workItemId WI-...
   * @param workspaceId 비면 {@link #DEFAULT_WORKSPACE}
   */
  @Transactional
  public void rememberWorkspace(String workItemId, String workspaceId) {
    String ref = workspaceRef(workItemId);
    if (contexts.exists(ref)) {
      return;
    }
    String id = (workspaceId == null || workspaceId.isBlank()) ? DEFAULT_WORKSPACE : workspaceId.trim();
    contexts.put(workItemId, REF_WORKSPACE, "{\"workspace_id\":\"" + escape(id) + "\"}");
  }

  /**
   * 현재 워크플로의 대기 승인 뷰. 없으면 null. CODE는 ProposedChanges만 담고 actual diff는 없다.
   *
   * @param row 현재 워크플로. null 불가
   */
  public ApprovalView viewFor(WorkflowRecord row) {
    if (row.currentState() == WorkflowState.WAITING_DESIGN_APPROVAL) {
      return designView(row);
    }
    if (row.currentState() == WorkflowState.WAITING_CODE_APPROVAL) {
      return codeView(row);
    }
    return null;
  }

  /**
   * Local Agent apply 성공을 받아 COMPLETED로 닫는다. 서버는 디스크를 쓰지 않는다.
   *
   * @param data work_item_id, workspace_id, change_set_id(선택)
   */
  @Transactional
  public ChatOutcome applyCompleted(Map<String, Object> data) {
    String workItemId = text(data, "work_item_id");
    String workspaceId = text(data, "workspace_id");
    if (workItemId.isEmpty() || workspaceId.isEmpty()) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "work_item_id and workspace_id are required");
    }
    if (!workspaceId.equals(workspaceIdOf(workItemId))) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "workspace_id does not match");
    }
    WorkflowRecord row = workflows.findByWorkItemId(workItemId).orElse(null);
    if (row == null) {
      return new ChatOutcome.Rejected("NOT_FOUND", "work item not found");
    }
    if (row.currentState() == WorkflowState.COMPLETED) {
      return new ChatOutcome.Accepted(
          row.workItemId(),
          row.workflowId(),
          List.of(ChatIntake.progressOf(WorkflowState.COMPLETED)),
          null,
          List.of());
    }
    try {
      WorkflowRecord done = engine.completeAfterApply(row.workflowId(), "apply.completed");
      return new ChatOutcome.Accepted(
          done.workItemId(),
          done.workflowId(),
          List.of(ChatIntake.progressOf(done.currentState())),
          null,
          List.of());
    } catch (IllegalWorkflowTransitionException | WorkflowNotFoundException ex) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", ex.getMessage());
    }
  }

  /**
   * 재연결 시 진행 상태와 대기 승인을 다시 만든다.
   *
   * @param workItemId WI-...
   */
  @Transactional
  public ChatOutcome restore(String workItemId) {
    if (workItemId == null || workItemId.isBlank()) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "work_item_id is required");
    }
    WorkflowRecord row = workflows.findByWorkItemId(workItemId.trim()).orElse(null);
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
        List.of(ChatIntake.progressOf(row.currentState())),
        viewFor(row),
        List.of());
  }

  /**
   * Local Agent actual diff 해시를 승인 행에 묶는다. IDE가 보낸 값이 아니라 에이전트가 계산한 값이어야 한다.
   *
   * @param data approval_id, work_item_id, workspace_id, resource_id, diff_hash
   */
  @Transactional
  public ChatOutcome bindDiff(Map<String, Object> data) {
    String approvalId = text(data, "approval_id");
    String workItemId = text(data, "work_item_id");
    String workspaceId = text(data, "workspace_id");
    String resourceId = text(data, "resource_id");
    String diffHash = text(data, "diff_hash");
    if (approvalId.isEmpty()
        || workItemId.isEmpty()
        || workspaceId.isEmpty()
        || resourceId.isEmpty()
        || diffHash.isEmpty()) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "approval bind fields are required");
    }
    ApprovalRecord pending = approvals.findById(approvalId).orElse(null);
    if (pending == null
        || !workItemId.equals(pending.workItemId())
        || !resourceId.equals(pending.resourceId())
        || pending.approvalType() != ApprovalType.CODE_CHANGE
        || pending.status() != ApprovalStatus.REQUESTED
        || pending.consumed()) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "approval bind does not match CODE_CHANGE");
    }
    if (!workspaceId.equals(workspaceIdOf(workItemId))) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "workspace_id does not match");
    }
    String json =
        "{\"approval_id\":\""
            + escape(approvalId)
            + "\",\"workspace_id\":\""
            + escape(workspaceId)
            + "\",\"resource_id\":\""
            + escape(resourceId)
            + "\",\"diff_hash\":\""
            + escape(diffHash)
            + "\"}";
    contexts.put(workItemId, REF_BIND, json);
    return new ChatOutcome.Accepted(
        pending.workItemId(), pending.workflowId(), List.of(), null, List.of());
  }

  /**
   * 사용자 결정을 바인딩과 함께 소비한다. 거절은 REJECTED. 승인은 apply를 부르지 않는다.
   *
   * @param data approval_id, decision, work_item_id, workspace_id, resource_id, design_identity 또는 diff_hash
   */
  @Transactional
  public ChatOutcome decide(Map<String, Object> data) {
    String approvalId = text(data, "approval_id");
    String decisionRaw = text(data, "decision");
    String workItemId = text(data, "work_item_id");
    String workspaceId = text(data, "workspace_id");
    String resourceId = text(data, "resource_id");
    String reason = text(data, "reason");
    if (approvalId.isEmpty() || decisionRaw.isEmpty() || workItemId.isEmpty() || workspaceId.isEmpty()
        || resourceId.isEmpty()) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "approval decide fields are required");
    }
    ApprovalDecision decision;
    try {
      decision = ApprovalDecision.valueOf(decisionRaw);
    } catch (IllegalArgumentException ex) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "decision must be APPROVE or REJECT");
    }
    ApprovalRecord pending;
    try {
      pending = approvals.requireUsable(approvalId);
    } catch (ApprovalNotFoundException
        | ApprovalExpiredException
        | ApprovalReuseException ex) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", ex.getMessage());
    }
    if (!workItemId.equals(pending.workItemId()) || !resourceId.equals(pending.resourceId())) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "approval resource binding mismatch");
    }
    if (!workspaceId.equals(workspaceIdOf(workItemId))) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", "workspace_id does not match");
    }
    try {
      if (pending.approvalType() == ApprovalType.DESIGN) {
        String identity = text(data, "design_identity");
        ApprovalView expected = designView(engine.require(pending.workflowId()));
        if (expected == null || !identity.equals(expected.designIdentity())) {
          return new ChatOutcome.Rejected("INVALID_REQUEST", "design_identity does not match");
        }
      } else if (pending.approvalType() == ApprovalType.CODE_CHANGE) {
        String diffHash = text(data, "diff_hash");
        String bound = boundDiffHash(workItemId, approvalId, resourceId);
        if (bound == null || !bound.equals(diffHash)) {
          return new ChatOutcome.Rejected("INVALID_REQUEST", "diff_hash does not match Local Agent bind");
        }
      }
    } catch (RuntimeException ex) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", ex.getMessage());
    }
    WorkflowRecord next;
    try {
      next = engine.decide(approvalId, decision, "user", reason.isEmpty() ? null : reason);
    } catch (ApprovalResourceMismatchException
        | IllegalWorkflowTransitionException
        | ApprovalReuseException
        | ApprovalExpiredException
        | ApprovalNotFoundException ex) {
      return new ChatOutcome.Rejected("INVALID_REQUEST", ex.getMessage());
    }
    List<ProgressEvent> progress = new ArrayList<>();
    progress.add(ChatIntake.progressOf(next.currentState()));
    if (decision == ApprovalDecision.APPROVE && next.currentState() == WorkflowState.IMPLEMENTING) {
      next = loop.implementAndReview(next.workflowId());
      progress.add(ChatIntake.progressOf(WorkflowState.IMPLEMENTING));
      progress.add(ChatIntake.progressOf(WorkflowState.REVIEWING));
      progress.add(ChatIntake.progressOf(WorkflowState.PREPARING_CHANGE));
      progress.add(ChatIntake.progressOf(next.currentState()));
    }
    return new ChatOutcome.Accepted(
        next.workItemId(), next.workflowId(), progress, viewFor(next), List.of());
  }

  private ApprovalView designView(WorkflowRecord row) {
    ApprovalRecord pending =
        approvals.findRequested(row.workflowId(), ApprovalType.DESIGN).orElse(null);
    if (pending == null) {
      return null;
    }
    String payload = contexts.find(pending.resourceId()).orElse("");
    String summary = extractSummary(payload);
    return ApprovalView.design(
        pending.approvalId(),
        pending.workItemId(),
        pending.workflowId(),
        workspaceIdOf(row.workItemId()),
        pending.resourceType(),
        pending.resourceId(),
        ContentHash.sha256(payload),
        summary);
  }

  private ApprovalView codeView(WorkflowRecord row) {
    ApprovalRecord pending =
        approvals.findRequested(row.workflowId(), ApprovalType.CODE_CHANGE).orElse(null);
    if (pending == null) {
      return null;
    }
    String proposed =
        contexts
            .find("ctx://work-items/" + row.workItemId() + "/change_set/latest")
            .orElse("");
    return ApprovalView.codeProposed(
        pending.approvalId(),
        pending.workItemId(),
        pending.workflowId(),
        workspaceIdOf(row.workItemId()),
        pending.resourceType(),
        pending.resourceId(),
        proposed);
  }

  private String workspaceIdOf(String workItemId) {
    return contexts
        .find(workspaceRef(workItemId))
        .map(
            json -> {
              Map<String, Object> parsed = JsonParserFactory.getJsonParser().parseMap(json);
              Object id = parsed.get("workspace_id");
              if (id != null && !String.valueOf(id).isBlank()) {
                return String.valueOf(id).trim();
              }
              return DEFAULT_WORKSPACE;
            })
        .orElse(DEFAULT_WORKSPACE);
  }

  private String boundDiffHash(String workItemId, String approvalId, String resourceId) {
    return contexts
        .find(bindRef(workItemId))
        .map(
            json -> {
              Map<String, Object> parsed = JsonParserFactory.getJsonParser().parseMap(json);
              if (!approvalId.equals(String.valueOf(parsed.getOrDefault("approval_id", "")))) {
                return null;
              }
              if (!resourceId.equals(String.valueOf(parsed.getOrDefault("resource_id", "")))) {
                return null;
              }
              String hash = String.valueOf(parsed.getOrDefault("diff_hash", "")).trim();
              return hash.isEmpty() ? null : hash;
            })
        .orElse(null);
  }

  private static String workspaceRef(String workItemId) {
    return "ctx://work-items/" + workItemId + "/workspace/latest";
  }

  private static String bindRef(String workItemId) {
    return "ctx://work-items/" + workItemId + "/approval_bind/latest";
  }

  /**
   * JSON object에서 summary 문자열을 꺼낸다. 없으면 본문 앞부분을 쓴다.
   *
   * @param payload DESIGN context JSON
   */
  static String extractSummary(String payload) {
    if (payload == null || payload.isBlank()) {
      return "설계를 검토해 주세요.";
    }
    try {
      Map<String, Object> parsed = JsonParserFactory.getJsonParser().parseMap(payload);
      Object summary = parsed.get("summary");
      if (summary != null && !String.valueOf(summary).isBlank()) {
        return String.valueOf(summary).trim();
      }
    } catch (RuntimeException ignored) {
      // raw
    }
    return payload.length() > 200 ? payload.substring(0, 200) : payload;
  }

  /**
   * JSON object에서 문자열 필드를 꺼낸다. 없으면 빈 문자열.
   *
   * @param data 파싱된 맵. null 가능
   * @param key 필드 이름
   */
  public static String text(Map<String, Object> data, String key) {
    if (data == null || data.get(key) == null) {
      return "";
    }
    return String.valueOf(data.get(key)).trim();
  }

  private static String escape(String value) {
    return value.replace("\\", "\\\\").replace("\"", "\\\"");
  }
}
