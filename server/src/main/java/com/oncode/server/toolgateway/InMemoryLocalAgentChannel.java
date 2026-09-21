package com.oncode.server.toolgateway;

import com.oncode.server.persistence.EntityIds;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;
import org.springframework.stereotype.Service;

/**
 * IDE가 없을 때 쓰는 Phase 1 도구 하니스. 디스크를 건드리지 않고 메모리 파일만 다룬다.
 * gRPC 전송(ADR-003)은 이 클래스가 아니다.
 */
@Service
public class InMemoryLocalAgentChannel implements LocalAgentChannel {

  static final String PROTOCOL_VERSION = "oncode-tool/1.0";

  private final ConcurrentHashMap<String, String> files = new ConcurrentHashMap<>();
  private final CopyOnWriteArrayList<String> invokedTools = new CopyOnWriteArrayList<>();

  /** 기본 mock 워크스페이스 파일을 넣는다. */
  public InMemoryLocalAgentChannel() {
    files.put("src/AuthService.java", "class AuthService {}");
  }

  /**
   * 게이트웨이가 실제로 채널에 넘긴 도구명을 시간순으로 돌려준다.
   *
   * @return 빈 리스트 가능
   */
  public List<String> invokedTools() {
    return List.copyOf(invokedTools);
  }

  /**
   * {@inheritDoc}
   */
  @Override
  public ToolResponse invoke(ToolCall call) {
    invokedTools.add(call.tool());
    return switch (call.tool()) {
      case "system.ping" -> ping(call.callId());
      case "system.get_capabilities" -> capabilities(call.callId());
      case "workspace.read_file" -> readFile(call);
      case "workspace.propose_changes" -> propose(call);
      case "workspace.apply_changes" -> apply(call);
      default ->
          ToolResponse.failed(
              call.callId(),
              "FAILED",
              new ToolError(
                  "TOOL_NOT_SUPPORTED", "TOOL", "tool not supported: " + call.tool(), false),
              0);
    };
  }

  private static ToolResponse ping(String callId) {
    return ToolResponse.success(
        callId,
        "{\"agent_id\":\"local-harness\",\"version\":\"0.1.0\",\"protocol_version\":\""
            + PROTOCOL_VERSION
            + "\"}",
        0);
  }

  private static ToolResponse capabilities(String callId) {
    return ToolResponse.success(
        callId,
        "{\"protocol_version\":\""
            + PROTOCOL_VERSION
            + "\",\"supported_tools\":[\"system.ping\",\"system.get_capabilities\",\"workspace.read_file\",\"workspace.propose_changes\",\"workspace.apply_changes\"]}",
        0);
  }

  private ToolResponse readFile(ToolCall call) {
    Object pathObj = call.arguments() == null ? null : call.arguments().get("path");
    String path = pathObj == null ? "" : pathObj.toString();
    String content = files.get(path);
    if (content == null) {
      return ToolResponse.failed(
          call.callId(),
          "FAILED",
          new ToolError("WORKSPACE_FILE_NOT_FOUND", "WORKSPACE", "file not found: " + path, false),
          0);
    }
    return ToolResponse.success(
        call.callId(),
        "{\"path\":\"" + escape(path) + "\",\"content\":\"" + escape(content) + "\"}",
        0);
  }

  private ToolResponse propose(ToolCall call) {
    Map<String, Object> args = call.arguments() == null ? Map.of() : call.arguments();
    String changeSetId = EntityIds.changeSet();
    String path = String.valueOf(args.getOrDefault("path", "src/AuthService.java"));
    String operation = String.valueOf(args.getOrDefault("operation", "MODIFY"));
    String content = String.valueOf(args.getOrDefault("content", ""));
    return ToolResponse.success(
        call.callId(),
        "{\"change_set_id\":\""
            + escape(changeSetId)
            + "\",\"changes\":[{\"path\":\""
            + escape(path)
            + "\",\"operation\":\""
            + escape(operation)
            + "\",\"content\":\""
            + escape(content)
            + "\"}]}",
        0);
  }

  private static ToolResponse apply(ToolCall call) {
    return ToolResponse.success(call.callId(), "{\"applied\":true}", 0);
  }

  private static String escape(String value) {
    return value.replace("\\", "\\\\").replace("\"", "\\\"");
  }
}
