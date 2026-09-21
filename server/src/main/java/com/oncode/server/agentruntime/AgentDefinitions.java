package com.oncode.server.agentruntime;

import com.oncode.server.inferencegateway.ModelProfile;
import java.util.List;
import java.util.Map;

/**
 * Phase 2 최소 에이전트 정의. apply는 허용 목록에 없다.
 */
public final class AgentDefinitions {

  private static final Map<AgentId, AgentDefinition> BY_ID =
      Map.of(
          AgentId.SENIOR_DEVELOPER,
          new AgentDefinition(
              AgentId.SENIOR_DEVELOPER, ModelProfile.REASONING_HIGH, List.of(), "design"),
          AgentId.IMPLEMENTATION,
          new AgentDefinition(
              AgentId.IMPLEMENTATION,
              ModelProfile.CODING,
              List.of("workspace.read_file", "workspace.propose_changes"),
              "proposed_change"),
          AgentId.REVIEW,
          new AgentDefinition(
              AgentId.REVIEW, ModelProfile.REVIEW, List.of("workspace.read_file"), "review"));

  private AgentDefinitions() {}

  /**
   * 식별자로 정의를 돌려준다.
   *
   * @param id 에이전트
   * @return 고정 정의
   */
  public static AgentDefinition require(AgentId id) {
    AgentDefinition definition = BY_ID.get(id);
    if (definition == null) {
      throw new IllegalArgumentException("unknown agent: " + id);
    }
    return definition;
  }

  /**
   * 세 에이전트 정의를 돌려준다.
   *
   * @return Senior, Implementation, Review
   */
  public static List<AgentDefinition> all() {
    return List.of(
        require(AgentId.SENIOR_DEVELOPER),
        require(AgentId.IMPLEMENTATION),
        require(AgentId.REVIEW));
  }
}
