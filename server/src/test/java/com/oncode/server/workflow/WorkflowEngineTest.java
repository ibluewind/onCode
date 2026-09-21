package com.oncode.server.workflow;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.oncode.server.context.ContextNotFoundException;
import com.oncode.server.context.ContextService;
import com.oncode.server.persistence.WorkflowRecord;
import com.oncode.server.support.PostgresSpringBootTest;
import java.sql.Timestamp;
import java.time.Instant;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.simple.JdbcClient;

/**
 * PostgreSQL에 워크플로·컨텍스트가 남는지, WAITING 재시작과 리스 회수가 되는지 검증한다.
 */
@PostgresSpringBootTest
class WorkflowEngineTest {

  @Autowired private WorkflowEngine engine;
  @Autowired private ContextService contexts;
  @Autowired private JdbcClient jdbc;

  /**
   * RECEIVED부터 WAITING_DESIGN_APPROVAL까지 전이가 모두 남는지를 본다.
   */
  @Test
  void persistsLinearTransitionsToWaitingDesignApproval() {
    WorkflowRecord done =
        engine.receiveAndDriveToWaitingDesignApproval("로그인 실패 5회 시 계정을 잠가줘.");
    assertThat(done.currentState()).isEqualTo(WorkflowState.WAITING_DESIGN_APPROVAL);
    assertThat(done.status()).isEqualTo(WorkflowStatus.WAITING);
    assertThat(done.leaseOwner()).isNull();
    assertThat(engine.history(done.workflowId()))
        .extracting(row -> row.toState())
        .containsExactly(
            WorkflowState.RECEIVED,
            WorkflowState.CLASSIFYING,
            WorkflowState.DISCOVERING_CONTEXT,
            WorkflowState.DESIGNING,
            WorkflowState.WAITING_DESIGN_APPROVAL);
  }

  /**
   * 그래프에 없는 전이는 상태를 바꾸지 않는다.
   */
  @Test
  void rejectsIllegalTransition() {
    WorkflowRecord received = engine.receive("상태를 건너뛰지 말 것");
    assertThatThrownBy(
            () -> engine.transition(received.workflowId(), WorkflowState.DESIGNING, "skip"))
        .isInstanceOf(IllegalWorkflowTransitionException.class);
    assertThat(engine.require(received.workflowId()).currentState())
        .isEqualTo(WorkflowState.RECEIVED);
  }

  /**
   * WAITING_*는 프로세스 재시작(DB에서 다시 읽기) 후에도 유지되고 워커 리스를 잡지 않는다.
   */
  @Test
  void waitingDesignApprovalSurvivesRestartWithoutLease() {
    WorkflowRecord waiting =
        engine.receiveAndDriveToWaitingDesignApproval("설계 승인 대기 재개");
    WorkflowRecord resumed = engine.resumeAfterRestart(waiting.workflowId());
    assertThat(resumed.currentState()).isEqualTo(WorkflowState.WAITING_DESIGN_APPROVAL);
    assertThat(resumed.status()).isEqualTo(WorkflowStatus.WAITING);
    assertThat(resumed.leaseOwner()).isNull();
    assertThat(engine.tryAcquireLease(waiting.workflowId())).isFalse();
  }

  /**
   * RUNNING 중 만료된 리스는 다른 실행이 회수할 수 있다.
   */
  @Test
  void expiredRunningLeaseCanBeReacquired() {
    WorkflowRecord received = engine.receive("리스 회수");
    Instant past = Instant.parse("2020-01-01T00:00:00Z");
    jdbc.sql(
            """
            UPDATE oncode.workflow
            SET lease_owner = 'dead-node', lease_until = :past
            WHERE workflow_id = :id
            """)
        .param("past", Timestamp.from(past))
        .param("id", received.workflowId())
        .update();
    assertThat(engine.tryAcquireLease(received.workflowId())).isTrue();
    assertThat(engine.require(received.workflowId()).leaseOwner()).isEqualTo("oncode-server");
  }

  /**
   * context.put으로 저장한 본문을 context.get으로 같은 ref에서 읽는다.
   */
  @Test
  void contextPutAndGetRoundTrip() {
    WorkflowRecord received = engine.receive("컨텍스트 라운드트립");
    String ref =
        contexts.put(received.workItemId(), "DESIGN", "{\"summary\":\"lock account\"}");
    assertThat(ref).startsWith("ctx://work-items/" + received.workItemId() + "/design/latest");
    assertThat(contexts.get(ref)).contains("lock account");
    assertThatThrownBy(() -> contexts.get("ctx://work-items/WI-MISSING/design/latest"))
        .isInstanceOf(ContextNotFoundException.class);
  }
}
