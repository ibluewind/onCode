package com.oncode.server.agentruntime;

import com.oncode.server.context.ContextService;
import com.oncode.server.inferencegateway.InferenceGateway;
import com.oncode.server.inferencegateway.InferenceRequest;
import com.oncode.server.inferencegateway.InferenceResponse;
import com.oncode.server.persistence.EntityIds;
import com.oncode.server.toolgateway.ToolCall;
import com.oncode.server.toolgateway.ToolGateway;
import com.oncode.server.toolgateway.ToolResponse;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import org.springframework.stereotype.Service;

/**
 * 프롬프트 구성, Inference Gateway 호출, 출력 검증, context.put.
 * Implementation만 propose_changes를 부르며 apply는 허용 목록에 없다.
 */
@Service
public class DefaultAgentRuntime implements AgentRuntime {

  static final String APPLY_TOOL = "workspace.apply_changes";
  private static final Pattern STRING_FIELD =
      Pattern.compile("\"([^\"]+)\"\\s*:\\s*\"((?:\\\\.|[^\"\\\\])*)\"");

  private final InferenceGateway inference;
  private final ToolGateway tools;
  private final ContextService contexts;

  /**
   * 게이트웨이와 컨텍스트 저장소를 받는다.
   *
   * @param inference 유일한 모델 진입점. null 불가
   * @param tools Local Agent 게이트웨이. null 불가
   * @param contexts context put/get. null 불가
   */
  public DefaultAgentRuntime(
      InferenceGateway inference, ToolGateway tools, ContextService contexts) {
    this.inference = inference;
    this.tools = tools;
    this.contexts = contexts;
  }

  /**
   * {@inheritDoc}
   */
  @Override
  public AgentResult run(AgentId agentId, AgentTurn turn) {
    Objects.requireNonNull(agentId, "agentId");
    Objects.requireNonNull(turn, "turn");
    if (turn.workItemId() == null || turn.workItemId().isBlank()) {
      throw new IllegalArgumentException("workItemId must not be blank");
    }
    if (turn.userMessage() == null || turn.userMessage().isBlank()) {
      throw new IllegalArgumentException("userMessage must not be blank");
    }
    AgentDefinition definition = AgentDefinitions.require(agentId);
    if (definition.allowedTools().contains(APPLY_TOOL)) {
      throw new IllegalStateException(agentId + " must not allow " + APPLY_TOOL);
    }
    String prompt = buildPrompt(turn);
    InferenceResponse model =
        inference.complete(
            new InferenceRequest(
                definition.modelProfile(),
                "Return JSON only. No hidden reasoning.",
                prompt,
                definition.outputSchemaName()));
    Map<String, String> output = stringFields(model.outputJson());
    return switch (agentId) {
      case SENIOR_DEVELOPER -> writeDesign(turn, definition, model.outputJson(), output);
      case IMPLEMENTATION -> writeProposal(turn, definition, output);
      case REVIEW -> writeReview(turn, definition, model.outputJson(), output);
    };
  }

  /**
   * {@inheritDoc}
   */
  @Override
  public List<AgentDefinition> definitions() {
    return AgentDefinitions.all();
  }

  /** context_ref만 프롬프트에 넣고, 본문은 런타임이 읽어 모델에 붙인다. 에이전트 간 전달은 ref다. */
  private String buildPrompt(AgentTurn turn) {
    StringBuilder prompt = new StringBuilder(turn.userMessage());
    List<String> refs = turn.contextRefs() == null ? List.of() : turn.contextRefs();
    for (String ref : refs) {
      prompt.append("\ncontext_ref=").append(ref);
      prompt.append("\n").append(contexts.get(ref));
    }
    return prompt.toString();
  }

  private AgentResult writeDesign(
      AgentTurn turn, AgentDefinition definition, String raw, Map<String, String> output) {
    requireField(output, "summary");
    String ref = contexts.put(turn.workItemId(), "DESIGN", raw);
    return new AgentResult(definition.id(), ref, definition.modelProfile());
  }

  private AgentResult writeProposal(
      AgentTurn turn, AgentDefinition definition, Map<String, String> output) {
    requireField(output, "path");
    requireField(output, "operation");
    requireField(output, "content");
    if (!definition.allowedTools().contains("workspace.propose_changes")) {
      throw new IllegalStateException("implementation cannot propose");
    }
    Map<String, Object> args = new LinkedHashMap<>();
    args.put("path", output.get("path"));
    args.put("operation", output.get("operation"));
    args.put("content", output.get("content"));
    ToolResponse proposed =
        tools.call(
            new ToolCall(
                EntityIds.call(),
                "workspace.propose_changes",
                args,
                5_000,
                "AGENT",
                "implementation-agent",
                turn.workflowId(),
                turn.workItemId()));
    if (!"SUCCESS".equals(proposed.status())) {
      throw new IllegalStateException(
          "propose_changes failed: "
              + (proposed.error() == null ? proposed.status() : proposed.error().code()));
    }
    String ref = contexts.put(turn.workItemId(), "CHANGE_SET", proposed.resultJson());
    return new AgentResult(definition.id(), ref, definition.modelProfile());
  }

  private AgentResult writeReview(
      AgentTurn turn, AgentDefinition definition, String raw, Map<String, String> output) {
    requireField(output, "verdict");
    String ref = contexts.put(turn.workItemId(), "REVIEW", raw);
    return new AgentResult(definition.id(), ref, definition.modelProfile());
  }

  private static Map<String, String> stringFields(String raw) {
    if (raw == null || raw.isBlank()) {
      throw new IllegalArgumentException("model output must be a JSON object");
    }
    Matcher matcher = STRING_FIELD.matcher(raw);
    Map<String, String> fields = new LinkedHashMap<>();
    while (matcher.find()) {
      fields.put(matcher.group(1), matcher.group(2).replace("\\\"", "\"").replace("\\\\", "\\"));
    }
    if (fields.isEmpty()) {
      throw new IllegalArgumentException("model output must be a JSON object");
    }
    return fields;
  }

  private static void requireField(Map<String, String> output, String field) {
    String value = output.get(field);
    if (value == null || value.isBlank()) {
      throw new IllegalArgumentException("missing output field: " + field);
    }
  }
}
