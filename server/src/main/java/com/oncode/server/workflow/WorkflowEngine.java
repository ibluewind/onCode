package com.oncode.server.workflow;

import com.oncode.server.approval.ApprovalDecision;
import com.oncode.server.approval.ApprovalPort;
import com.oncode.server.approval.ApprovalRecord;
import com.oncode.server.approval.ApprovalResourceMismatchException;
import com.oncode.server.approval.ApprovalType;
import com.oncode.server.context.ContextService;
import com.oncode.server.persistence.EntityIds;
import com.oncode.server.persistence.WorkItemRecord;
import com.oncode.server.persistence.WorkItemRepository;
import com.oncode.server.persistence.WorkflowRecord;
import com.oncode.server.persistence.WorkflowRepository;
import com.oncode.server.persistence.WorkflowTransitionRecord;
import com.oncode.server.persistence.WorkflowTransitionRepository;
import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.Objects;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * 공식 워크플로 상태를 PostgreSQL에 영속한다. 전이만 수행하며 LLM 출력은 상태가 아니다.
 * 에이전트 본문·Tool Gateway 실호출·workspace apply는 하지 않는다.
 */
@Service
public class WorkflowEngine {

  static final String TYPE_IMPLEMENTATION = "IMPLEMENTATION";
  static final String TRIGGER_ENGINE = "ENGINE";
  static final String TRIGGER_SYSTEM = "SYSTEM";
  static final String TRIGGER_USER_DECISION = "USER_DECISION";
  static final String RESOURCE_TYPE_DESIGN_REF = "DESIGN_REF";
  static final String RESOURCE_TYPE_CHANGE_SET = "CHANGE_SET";

  private final WorkItemRepository workItems;
  private final WorkflowRepository workflows;
  private final WorkflowTransitionRepository transitions;
  private final ContextService contexts;
  private final ApprovalPort approvals;
  private final String leaseOwner;
  private final Duration leaseDuration;
  private final Duration approvalTtl;
  private final Clock clock;

  /**
   * 저장소·승인 포트·리스 설정을 받는다.
   *
   * @param workItems 워크 아이템 저장소. null 불가
   * @param workflows 워크플로 저장소. null 불가
   * @param transitions 전이 이력 저장소. null 불가
   * @param contexts 설계 stub 저장. null 불가
   * @param approvals 승인 포트. null 불가
   * @param leaseOwner 이 프로세스의 리스 이름. 비면 {@code oncode-server}
   * @param leaseDurationSeconds 리스 길이(초). 1 이상
   * @param approvalTtlSeconds 승인 유효 시간(초). 1 이상
   */
  @Autowired
  public WorkflowEngine(
      WorkItemRepository workItems,
      WorkflowRepository workflows,
      WorkflowTransitionRepository transitions,
      ContextService contexts,
      ApprovalPort approvals,
      @Value("${oncode.workflow.lease-owner:oncode-server}") String leaseOwner,
      @Value("${oncode.workflow.lease-duration-seconds:30}") long leaseDurationSeconds,
      @Value("${oncode.approval.ttl-seconds:3600}") long approvalTtlSeconds) {
    this(
        workItems,
        workflows,
        transitions,
        contexts,
        approvals,
        leaseOwner,
        Duration.ofSeconds(leaseDurationSeconds),
        Duration.ofSeconds(approvalTtlSeconds),
        Clock.systemUTC());
  }

  /**
   * 테스트에서 시계를 고정할 때 쓴다.
   *
   * @param workItems 워크 아이템 저장소
   * @param workflows 워크플로 저장소
   * @param transitions 전이 이력
   * @param contexts 컨텍스트 저장
   * @param approvals 승인 포트
   * @param leaseOwner 리스 소유자
   * @param leaseDuration 리스 길이. 양수
   * @param approvalTtl 승인 TTL. 양수
   * @param clock UTC 시계
   */
  WorkflowEngine(
      WorkItemRepository workItems,
      WorkflowRepository workflows,
      WorkflowTransitionRepository transitions,
      ContextService contexts,
      ApprovalPort approvals,
      String leaseOwner,
      Duration leaseDuration,
      Duration approvalTtl,
      Clock clock) {
    this.workItems = workItems;
    this.workflows = workflows;
    this.transitions = transitions;
    this.contexts = contexts;
    this.approvals = approvals;
    this.leaseOwner = leaseOwner;
    this.leaseDuration = leaseDuration;
    this.approvalTtl = approvalTtl;
    this.clock = clock;
  }

  /**
   * 요청을 받아 Work Item과 Workflow를 {@link WorkflowState#RECEIVED}로 만든다.
   *
   * @param originalRequest 사용자 원문. null/공백이면 예외
   * @return 생성된 워크플로 행
   */
  @Transactional
  public WorkflowRecord receive(String originalRequest) {
    if (originalRequest == null || originalRequest.isBlank()) {
      throw new IllegalArgumentException("originalRequest must not be blank");
    }
    Instant now = clock.instant();
    String workItemId = EntityIds.workItem();
    String workflowId = EntityIds.workflow();
    workItems.insert(
        new WorkItemRecord(
            workItemId,
            null,
            originalRequest.trim(),
            TYPE_IMPLEMENTATION,
            WorkItemStatus.OPEN,
            now,
            now));
    WorkflowRecord created =
        new WorkflowRecord(
            workflowId,
            workItemId,
            TYPE_IMPLEMENTATION,
            WorkflowState.RECEIVED,
            null,
            null,
            0,
            WorkflowStatus.RUNNING,
            0,
            leaseOwner,
            now.plus(leaseDuration),
            now,
            now);
    workflows.insert(created);
    insertTransition(
        workflowId, null, WorkflowState.RECEIVED, TRIGGER_SYSTEM, null, "received", now);
    return created;
  }

  /**
   * 허용된 한 단계만 전이한다. 승인 대기 상태는 이 메소드로 빠져나가지 못한다.
   *
   * @param workflowId WF-... . 없으면 {@link WorkflowNotFoundException}
   * @param to 도착 상태. null 불가
   * @param reason 이력에 남길 짧은 이유. null 가능
   * @return 갱신된 워크플로 행
   */
  @Transactional
  public WorkflowRecord transition(String workflowId, WorkflowState to, String reason) {
    WorkflowRecord current =
        workflows.findById(workflowId).orElseThrow(() -> new WorkflowNotFoundException(workflowId));
    if (current.currentState().isWaiting()) {
      throw new IllegalWorkflowTransitionException(
          current.currentState() + " requires an approval decision");
    }
    return applyTransition(current, to, TRIGGER_ENGINE, null, reason);
  }

  /**
   * 승인 결정을 반영한다. 소비와 전이를 한 트랜잭션에서 수행한다.
   * apply는 호출하지 않는다.
   *
   * @param approvalId APR-... . 사용 가능한 REQUESTED여야 한다
   * @param decision APPROVE 또는 REJECT
   * @param actor 결정 주체. null이면 {@code user}
   * @param reason 사유. null 가능
   * @return 전이된 워크플로
   */
  @Transactional
  public WorkflowRecord decide(
      String approvalId, ApprovalDecision decision, String actor, String reason) {
    Objects.requireNonNull(decision, "decision");
    ApprovalRecord pending = approvals.requireUsable(approvalId);
    WorkflowRecord current = require(pending.workflowId());
    WorkflowState target = targetAfterDecision(current, pending, decision);
    String who = (actor == null || actor.isBlank()) ? "user" : actor.trim();
    approvals.consume(approvalId, decision, who, reason);
    return applyTransition(current, target, TRIGGER_USER_DECISION, approvalId, reason);
  }

  /**
   * RECEIVED부터 선형 경로를 {@link WorkflowState#WAITING_DESIGN_APPROVAL}까지 돌린다.
   * 에이전트는 호출하지 않는다. 설계 stub과 DESIGN 승인을 남긴다.
   *
   * @param originalRequest 사용자 원문
   * @return 대기 상태에 들어간 워크플로
   */
  @Transactional
  public WorkflowRecord receiveAndDriveToWaitingDesignApproval(String originalRequest) {
    WorkflowRecord current = receive(originalRequest);
    while (current.currentState() != WorkflowState.WAITING_DESIGN_APPROVAL) {
      WorkflowState from = current.currentState();
      WorkflowState next =
          WorkflowTransitions.nextLinear(from)
              .orElseThrow(
                  () -> new IllegalWorkflowTransitionException("no linear next from " + from));
      current = transition(current.workflowId(), next, "linear-advance");
    }
    return current;
  }

  /**
   * 설계 승인 후 {@link WorkflowState#IMPLEMENTING}에서 코드 승인 대기까지 선형 스텁을 돌린다.
   *
   * @param workflowId 현재 IMPLEMENTING인 WF-...
   * @return WAITING_CODE_APPROVAL 행
   */
  @Transactional
  public WorkflowRecord driveToWaitingCodeApproval(String workflowId) {
    WorkflowRecord current = require(workflowId);
    if (current.currentState() != WorkflowState.IMPLEMENTING) {
      throw new IllegalWorkflowTransitionException(
          "driveToWaitingCodeApproval requires IMPLEMENTING, was " + current.currentState());
    }
    while (current.currentState() != WorkflowState.WAITING_CODE_APPROVAL) {
      WorkflowState from = current.currentState();
      WorkflowState next =
          WorkflowTransitions.nextLinear(from)
              .orElseThrow(
                  () -> new IllegalWorkflowTransitionException("no linear next from " + from));
      current = transition(current.workflowId(), next, "linear-advance");
    }
    return current;
  }

  /**
   * Local Agent apply 성공을 반영한다. APPLYING_CHANGE에서만 COMPLETED로 간다.
   * BUILDING/TESTING은 호출하지 않는다.
   *
   * @param workflowId WF-... . APPLYING_CHANGE여야 한다
   * @param reason 이력 사유. null 가능
   * @return COMPLETED 행
   */
  @Transactional
  public WorkflowRecord completeAfterApply(String workflowId, String reason) {
    WorkflowRecord current = require(workflowId);
    if (current.currentState() != WorkflowState.APPLYING_CHANGE) {
      throw new IllegalWorkflowTransitionException(
          "completeAfterApply requires APPLYING_CHANGE, was " + current.currentState());
    }
    WorkflowRecord done =
        applyTransition(
            current,
            WorkflowState.COMPLETED,
            TRIGGER_SYSTEM,
            null,
            reason == null || reason.isBlank() ? "local-agent-apply" : reason.trim());
    Instant now = clock.instant();
    workItems.updateStatus(current.workItemId(), WorkItemStatus.COMPLETED, now);
    return done;
  }

  /**
   * DB에서 워크플로를 다시 읽어 재시작 이후에도 상태가 남았는지 확인한다.
   * WAITING_*이면 워커 리스를 잡지 않는다.
   *
   * @param workflowId WF-...
   * @return DB의 현재 행
   */
  @Transactional
  public WorkflowRecord resumeAfterRestart(String workflowId) {
    WorkflowRecord loaded =
        workflows.findById(workflowId).orElseThrow(() -> new WorkflowNotFoundException(workflowId));
    if (loaded.currentState().isWaiting()) {
      workflows.clearLease(workflowId, clock.instant());
      return workflows
          .findById(workflowId)
          .orElseThrow(() -> new WorkflowNotFoundException(workflowId));
    }
    if (loaded.currentState().isTerminal()) {
      return loaded;
    }
    Instant now = clock.instant();
    boolean acquired =
        workflows.tryAcquireLease(workflowId, leaseOwner, now.plus(leaseDuration), now);
    if (!acquired) {
      throw new WorkflowConcurrentModificationException(workflowId, loaded.revision());
    }
    return workflows
        .findById(workflowId)
        .orElseThrow(() -> new WorkflowNotFoundException(workflowId));
  }

  /**
   * 만료된 리스를 이 노드가 회수한다. WAITING_*는 회수하지 않고 상태만 유지한다.
   *
   * @param workflowId WF-...
   * @return 리스를 가져왔으면 true. WAITING이거나 다른 노드가 갖고 있으면 false
   */
  @Transactional
  public boolean tryAcquireLease(String workflowId) {
    WorkflowRecord loaded =
        workflows.findById(workflowId).orElseThrow(() -> new WorkflowNotFoundException(workflowId));
    if (loaded.currentState().isWaiting()) {
      return false;
    }
    Instant now = clock.instant();
    return workflows.tryAcquireLease(workflowId, leaseOwner, now.plus(leaseDuration), now);
  }

  /**
   * 워크플로 행을 읽는다. 없으면 예외.
   *
   * @param workflowId WF-...
   * @return 현재 행
   */
  public WorkflowRecord require(String workflowId) {
    return workflows.findById(workflowId).orElseThrow(() -> new WorkflowNotFoundException(workflowId));
  }

  /**
   * 전이 이력을 시간순으로 돌려준다.
   *
   * @param workflowId WF-...
   * @return 빈 리스트 가능. null 아님
   */
  public List<WorkflowTransitionRecord> history(String workflowId) {
    Objects.requireNonNull(workflowId, "workflowId");
    return transitions.findByWorkflowId(workflowId);
  }

  /**
   * 설계 컨텍스트 ref를 만든다. {@link ContextService#put}과 같은 형식이다.
   *
   * @param workItemId WI-...
   * @return {@code ctx://work-items/{id}/design/latest}
   */
  public static String designResourceId(String workItemId) {
    return "ctx://work-items/" + workItemId + "/design/latest";
  }

  /**
   * 코드 변경 세트 stub 리소스 ID를 만든다. apply 대상이 아니다.
   *
   * @param workItemId WI-...
   * @return {@code cs://work-items/{id}/latest}
   */
  static String changeSetResourceId(String workItemId) {
    return "cs://work-items/" + workItemId + "/latest";
  }

  private WorkflowRecord applyTransition(
      WorkflowRecord current,
      WorkflowState to,
      String triggerType,
      String triggerId,
      String reason) {
    if (!WorkflowTransitions.allows(current.currentState(), to)) {
      throw new IllegalWorkflowTransitionException(
          current.currentState() + " -> " + to + " is not allowed");
    }
    Instant now = clock.instant();
    WorkflowStatus status = statusOf(to);
    boolean release = to.isWaiting() || to.isTerminal();
    String owner = release ? null : leaseOwner;
    Instant until = release ? null : now.plus(leaseDuration);
    WorkflowRecord updated =
        new WorkflowRecord(
            current.workflowId(),
            current.workItemId(),
            current.workflowType(),
            to,
            current.currentState(),
            to.isWaiting() ? to : current.resumeState(),
            current.revision() + 1,
            status,
            current.retryCount(),
            owner,
            until,
            current.createdAt(),
            now);
    int rows = workflows.updateState(updated, current.revision());
    if (rows != 1) {
      throw new WorkflowConcurrentModificationException(current.workflowId(), current.revision());
    }
    insertTransition(
        current.workflowId(), current.currentState(), to, triggerType, triggerId, reason, now);
    if (to.isWaiting()) {
      openApproval(updated, now);
    }
    return updated;
  }

  /** WAITING 진입 시 해당 종류의 승인 요청을 남긴다. */
  private void openApproval(WorkflowRecord waiting, Instant now) {
    Instant expiresAt = now.plus(approvalTtl);
    if (waiting.currentState() == WorkflowState.WAITING_DESIGN_APPROVAL) {
      String ref = designResourceId(waiting.workItemId());
      if (!contexts.exists(ref)) {
        contexts.put(waiting.workItemId(), "DESIGN", "{\"status\":\"WAITING_APPROVAL\"}");
      }
      approvals.request(
          waiting.workItemId(),
          waiting.workflowId(),
          ApprovalType.DESIGN,
          RESOURCE_TYPE_DESIGN_REF,
          ref,
          expiresAt);
      return;
    }
    if (waiting.currentState() == WorkflowState.WAITING_CODE_APPROVAL) {
      approvals.request(
          waiting.workItemId(),
          waiting.workflowId(),
          ApprovalType.CODE_CHANGE,
          RESOURCE_TYPE_CHANGE_SET,
          changeSetResourceId(waiting.workItemId()),
          expiresAt);
    }
  }

  /** 승인 종류·리소스·현재 상태가 맞을 때만 도착 상태를 고른다. */
  private static WorkflowState targetAfterDecision(
      WorkflowRecord current, ApprovalRecord pending, ApprovalDecision decision) {
    if (pending.approvalType() == ApprovalType.DESIGN) {
      if (current.currentState() != WorkflowState.WAITING_DESIGN_APPROVAL
          || !designResourceId(current.workItemId()).equals(pending.resourceId())) {
        throw new ApprovalResourceMismatchException(
            "DESIGN approval does not match " + current.currentState());
      }
      return decision == ApprovalDecision.APPROVE
          ? WorkflowState.IMPLEMENTING
          : WorkflowState.REJECTED;
    }
    if (pending.approvalType() == ApprovalType.CODE_CHANGE) {
      if (current.currentState() != WorkflowState.WAITING_CODE_APPROVAL
          || !changeSetResourceId(current.workItemId()).equals(pending.resourceId())) {
        throw new ApprovalResourceMismatchException(
            "CODE_CHANGE approval does not match " + current.currentState());
      }
      return decision == ApprovalDecision.APPROVE
          ? WorkflowState.APPLYING_CHANGE
          : WorkflowState.REJECTED;
    }
    throw new ApprovalResourceMismatchException("unsupported approval type");
  }

  private static WorkflowStatus statusOf(WorkflowState to) {
    if (to == WorkflowState.COMPLETED) {
      return WorkflowStatus.COMPLETED;
    }
    if (to == WorkflowState.REJECTED) {
      return WorkflowStatus.FAILED;
    }
    if (to.isWaiting()) {
      return WorkflowStatus.WAITING;
    }
    return WorkflowStatus.RUNNING;
  }

  private void insertTransition(
      String workflowId,
      WorkflowState from,
      WorkflowState to,
      String triggerType,
      String triggerId,
      String reason,
      Instant now) {
    transitions.insert(
        new WorkflowTransitionRecord(
            EntityIds.transition(),
            workflowId,
            from,
            to,
            triggerType,
            triggerId,
            reason,
            now));
  }
}
