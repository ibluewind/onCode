package com.oncode.server.agentruntime;

import static org.assertj.core.api.Assertions.assertThat;

import com.oncode.server.context.ContextService;
import com.oncode.server.inferencegateway.InferenceGateway;
import com.oncode.server.persistence.EntityIds;
import com.oncode.server.persistence.WorkItemRecord;
import com.oncode.server.persistence.WorkItemRepository;
import com.oncode.server.support.PostgresSpringBootTest;
import com.oncode.server.toolgateway.InMemoryLocalAgentChannel;
import com.oncode.server.toolgateway.ToolCall;
import com.oncode.server.toolgateway.ToolGateway;
import com.oncode.server.workflow.WorkItemStatus;
import java.time.Instant;
import java.util.List;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.ApplicationContext;

/**
 * 세 에이전트가 존재하고, ref만 교환하며, apply를 부르지 않는지 검증한다.
 */
@PostgresSpringBootTest
class AgentRuntimeTest {

  @Autowired private AgentRuntime runtime;
  @Autowired private ContextService contexts;
  @Autowired private WorkItemRepository workItems;
  @Autowired private ToolGateway tools;
  @Autowired private InMemoryLocalAgentChannel harness;
  @Autowired private ApplicationContext applicationContext;

  /**
   * Senior / Implementation / Review 정의가 있고 apply가 허용 목록에 없다.
   */
  @Test
  void boundedAgentsExistWithoutApply() {
    assertThat(runtime.definitions()).extracting(AgentDefinition::id)
        .containsExactly(AgentId.SENIOR_DEVELOPER, AgentId.IMPLEMENTATION, AgentId.REVIEW);
    assertThat(runtime.definitions())
        .allSatisfy(definition -> assertThat(definition.allowedTools()).doesNotContain("workspace.apply_changes"));
  }

  /**
   * InferenceGateway 빈이 하나뿐이며 런타임이 그 타입만 주입받는다.
   */
  @Test
  void inferenceGatewayIsSoleModelEntry() {
    assertThat(applicationContext.getBeansOfType(InferenceGateway.class)).hasSize(1);
  }

  /**
   * Senior 결과는 context_ref이고, Implementation 턴에는 payload가 아니라 그 ref만 넘긴다.
   */
  @Test
  void agentsExchangeContextRefsNotPayloads() {
    String workItemId = insertWorkItem("설계 후 구현");
    AgentTurn seniorTurn = new AgentTurn(workItemId, "WF-TEST1", List.of(), "lock account");
    assertThat(seniorTurn.contextRefs()).isEmpty();
    AgentResult design = runtime.run(AgentId.SENIOR_DEVELOPER, seniorTurn);
    assertThat(design.contextRef()).startsWith("ctx://work-items/" + workItemId + "/design/latest");
    assertThat(contexts.get(design.contextRef())).contains("summary");

    AgentTurn implTurn =
        new AgentTurn(workItemId, "WF-TEST1", List.of(design.contextRef()), "implement design");
    assertThat(implTurn.contextRefs()).containsExactly(design.contextRef());
    AgentResult proposal = runtime.run(AgentId.IMPLEMENTATION, implTurn);
    assertThat(proposal.contextRef()).contains("/change_set/latest");
    assertThat(harness.invokedTools()).contains("workspace.propose_changes");
    assertThat(harness.invokedTools()).doesNotContain("workspace.apply_changes");
  }

  /**
   * 게이트웨이는 AGENT apply를 거절한다.
   */
  @Test
  void toolGatewayRejectsAgentApply() {
    var response =
        tools.call(
            new ToolCall(
                "CALL-AGENTAPPLY",
                "workspace.apply_changes",
                java.util.Map.of(),
                500,
                "AGENT",
                "implementation-agent",
                "WF-X",
                "WI-X"));
    assertThat(response.status()).isEqualTo("FAILED");
    assertThat(response.error().code()).isEqualTo("PERMISSION_DENIED");
  }

  private String insertWorkItem(String request) {
    Instant now = Instant.parse("2026-09-16T00:00:00Z");
    String id = EntityIds.workItem();
    workItems.insert(
        new WorkItemRecord(id, null, request, "IMPLEMENTATION", WorkItemStatus.OPEN, now, now));
    return id;
  }
}
