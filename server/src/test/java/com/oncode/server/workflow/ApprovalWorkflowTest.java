package com.oncode.server.workflow;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.oncode.server.approval.ApprovalDecision;
import com.oncode.server.approval.ApprovalExpiredException;
import com.oncode.server.approval.ApprovalPort;
import com.oncode.server.approval.ApprovalRecord;
import com.oncode.server.approval.ApprovalResourceMismatchException;
import com.oncode.server.approval.ApprovalReuseException;
import com.oncode.server.approval.ApprovalType;
import com.oncode.server.persistence.WorkflowRecord;
import com.oncode.server.support.PostgresSpringBootTest;
import java.sql.Timestamp;
import java.time.Instant;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.simple.JdbcClient;

/**
 * 설계/코드 승인이 상태를 바꾸고, 거절·재사용·만료는 상태를 그대로 두는지 검증한다.
 */
@PostgresSpringBootTest
class ApprovalWorkflowTest {

  @Autowired private WorkflowEngine engine;
  @Autowired private ApprovalPort approvals;
  @Autowired private JdbcClient jdbc;

  /**
   * 설계 승인은 IMPLEMENTING으로 가고 구현을 시작한다.
   */
  @Test
  void designApproveMovesToImplementing() {
    WorkflowRecord waiting = engine.receiveAndDriveToWaitingDesignApproval("설계 승인");
    ApprovalRecord pending = requireDesign(waiting.workflowId());
    WorkflowRecord next =
        engine.decide(pending.approvalId(), ApprovalDecision.APPROVE, "dev", "ok");
    assertThat(next.currentState()).isEqualTo(WorkflowState.IMPLEMENTING);
    assertThat(next.status()).isEqualTo(WorkflowStatus.RUNNING);
    assertThat(engine.require(waiting.workflowId()).currentState())
        .isNotEqualTo(WorkflowState.REJECTED);
  }

  /**
   * 설계 거절은 구현 상태에 들어가지 않는다.
   */
  @Test
  void designRejectStopsImplementation() {
    WorkflowRecord waiting = engine.receiveAndDriveToWaitingDesignApproval("설계 거절");
    ApprovalRecord pending = requireDesign(waiting.workflowId());
    WorkflowRecord next =
        engine.decide(pending.approvalId(), ApprovalDecision.REJECT, "dev", "no");
    assertThat(next.currentState()).isEqualTo(WorkflowState.REJECTED);
    assertThat(next.status()).isEqualTo(WorkflowStatus.FAILED);
    assertThat(engine.history(waiting.workflowId()))
        .extracting(row -> row.toState())
        .doesNotContain(WorkflowState.IMPLEMENTING, WorkflowState.APPLYING_CHANGE);
  }

  /**
   * 일반 transition으로는 승인 대기를 빠져나가지 못한다.
   */
  @Test
  void waitingStateCannotTransitionWithoutDecision() {
    WorkflowRecord waiting = engine.receiveAndDriveToWaitingDesignApproval("무단 전이");
    assertThatThrownBy(
            () ->
                engine.transition(
                    waiting.workflowId(), WorkflowState.IMPLEMENTING, "skip-approval"))
        .isInstanceOf(IllegalWorkflowTransitionException.class);
    assertThat(engine.require(waiting.workflowId()).currentState())
        .isEqualTo(WorkflowState.WAITING_DESIGN_APPROVAL);
  }

  /**
   * 코드 승인 대기는 재시작 후에도 유지되고 리스를 잡지 않는다.
   */
  @Test
  void waitingCodeApprovalSurvivesRestartWithoutLease() {
    WorkflowRecord waiting = waitingCode("코드 승인 대기 재개");
    WorkflowRecord resumed = engine.resumeAfterRestart(waiting.workflowId());
    assertThat(resumed.currentState()).isEqualTo(WorkflowState.WAITING_CODE_APPROVAL);
    assertThat(resumed.status()).isEqualTo(WorkflowStatus.WAITING);
    assertThat(resumed.leaseOwner()).isNull();
    assertThat(engine.tryAcquireLease(waiting.workflowId())).isFalse();
  }

  /**
   * 코드 승인은 APPLYING_CHANGE로만 가고 Local Agent apply는 호출하지 않는다.
   */
  @Test
  void codeApproveMovesToApplyingChange() {
    WorkflowRecord waiting = waitingCode("코드 승인");
    ApprovalRecord pending = requireCode(waiting.workflowId());
    WorkflowRecord next =
        engine.decide(pending.approvalId(), ApprovalDecision.APPROVE, "dev", "ship");
    assertThat(next.currentState()).isEqualTo(WorkflowState.APPLYING_CHANGE);
    assertThat(next.status()).isEqualTo(WorkflowStatus.RUNNING);
  }

  /**
   * apply 완료 통지 후 COMPLETED로 닫힌다.
   */
  @Test
  void completeAfterApplyClosesWorkflow() {
    WorkflowRecord waiting = waitingCode("apply 완료");
    ApprovalRecord pending = requireCode(waiting.workflowId());
    WorkflowRecord applying =
        engine.decide(pending.approvalId(), ApprovalDecision.APPROVE, "dev", "ship");
    WorkflowRecord done = engine.completeAfterApply(applying.workflowId(), "test");
    assertThat(done.currentState()).isEqualTo(WorkflowState.COMPLETED);
    assertThat(done.status()).isEqualTo(WorkflowStatus.COMPLETED);
    assertThat(done.leaseOwner()).isNull();
  }

  /**
   * 코드 거절은 APPLYING_CHANGE에 들어가지 않는다.
   */
  @Test
  void codeRejectDoesNotApply() {
    WorkflowRecord waiting = waitingCode("코드 거절");
    ApprovalRecord pending = requireCode(waiting.workflowId());
    WorkflowRecord next =
        engine.decide(pending.approvalId(), ApprovalDecision.REJECT, "dev", "redo");
    assertThat(next.currentState()).isEqualTo(WorkflowState.REJECTED);
    assertThat(engine.require(waiting.workflowId()).currentState())
        .isNotEqualTo(WorkflowState.APPLYING_CHANGE);
  }

  /**
   * 한 번 쓴 승인은 다시 쓸 수 없고 상태는 그대로다.
   */
  @Test
  void consumedApprovalCannotBeReused() {
    WorkflowRecord waiting = engine.receiveAndDriveToWaitingDesignApproval("재사용 금지");
    ApprovalRecord pending = requireDesign(waiting.workflowId());
    engine.decide(pending.approvalId(), ApprovalDecision.APPROVE, "dev", "once");
    assertThatThrownBy(
            () -> engine.decide(pending.approvalId(), ApprovalDecision.APPROVE, "dev", "twice"))
        .isInstanceOf(ApprovalReuseException.class);
    assertThat(engine.require(waiting.workflowId()).currentState())
        .isEqualTo(WorkflowState.IMPLEMENTING);
  }

  /**
   * 만료된 승인은 거부되고 대기 상태가 유지된다.
   */
  @Test
  void expiredApprovalIsRejected() {
    WorkflowRecord waiting = engine.receiveAndDriveToWaitingDesignApproval("만료");
    ApprovalRecord pending = requireDesign(waiting.workflowId());
    Instant past = Instant.parse("2020-01-01T00:00:00Z");
    jdbc.sql(
            """
            UPDATE oncode.approval
            SET expires_at = :past
            WHERE approval_id = :id
            """)
        .param("past", Timestamp.from(past))
        .param("id", pending.approvalId())
        .update();
    assertThatThrownBy(
            () -> engine.decide(pending.approvalId(), ApprovalDecision.APPROVE, "dev", "late"))
        .isInstanceOf(ApprovalExpiredException.class);
    assertThat(engine.require(waiting.workflowId()).currentState())
        .isEqualTo(WorkflowState.WAITING_DESIGN_APPROVAL);
  }

  /**
   * 코드 승인을 설계 대기에서 쓰면 불일치로 거부한다.
   */
  @Test
  void codeApprovalDoesNotSatisfyDesignWaiting() {
    WorkflowRecord waiting = engine.receiveAndDriveToWaitingDesignApproval("타입 불일치");
    ApprovalRecord bogus =
        approvals.request(
            waiting.workItemId(),
            waiting.workflowId(),
            ApprovalType.CODE_CHANGE,
            WorkflowEngine.RESOURCE_TYPE_CHANGE_SET,
            WorkflowEngine.changeSetResourceId(waiting.workItemId()),
            Instant.parse("2099-01-01T00:00:00Z"));
    assertThatThrownBy(
            () -> engine.decide(bogus.approvalId(), ApprovalDecision.APPROVE, "dev", "wrong"))
        .isInstanceOf(ApprovalResourceMismatchException.class);
    assertThat(engine.require(waiting.workflowId()).currentState())
        .isEqualTo(WorkflowState.WAITING_DESIGN_APPROVAL);
  }

  private WorkflowRecord waitingCode(String request) {
    WorkflowRecord designWaiting = engine.receiveAndDriveToWaitingDesignApproval(request);
    ApprovalRecord pending = requireDesign(designWaiting.workflowId());
    WorkflowRecord implementing =
        engine.decide(pending.approvalId(), ApprovalDecision.APPROVE, "dev", "design-ok");
    return engine.driveToWaitingCodeApproval(implementing.workflowId());
  }

  private ApprovalRecord requireDesign(String workflowId) {
    return approvals
        .findRequested(workflowId, ApprovalType.DESIGN)
        .orElseThrow(() -> new AssertionError("missing DESIGN approval"));
  }

  private ApprovalRecord requireCode(String workflowId) {
    return approvals
        .findRequested(workflowId, ApprovalType.CODE_CHANGE)
        .orElseThrow(() -> new AssertionError("missing CODE_CHANGE approval"));
  }
}
