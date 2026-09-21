package com.oncode.server.api;

import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.UUID;
import org.springframework.boot.json.JsonParserFactory;

/**
 * chat.submit JSON에서 소스 본문 첨부를 걷어낸다 (SPEC-07 §14).
 * 허용 필드는 message, work_item_id, 경로와 line range, cursor 좌표다.
 */
public final class IdeContextGuard {

  private static final Set<String> FORBIDDEN =
      Set.of("content", "source", "selected_text", "selection_text", "file_content", "source_text");

  private IdeContextGuard() {}

  /**
   * JSON object를 맵으로 파싱한다. 깨진 JSON이면 예외다.
   *
   * @param json object 문자열. 비면 안 된다
   */
  public static Map<String, Object> parseObject(String json) {
    if (json == null || json.isBlank()) {
      throw new IllegalArgumentException("json is blank");
    }
    return JsonParserFactory.getJsonParser().parseMap(json);
  }

  /**
   * 금지 키가 있으면 메시지를 담은 예외를 던진다.
   *
   * @param node JSON 트리. null이면 통과
   */
  public static void rejectSourceBody(Object node) {
    rejectSourceBody(node, "");
  }

  @SuppressWarnings("unchecked")
  private static void rejectSourceBody(Object node, String path) {
    if (node instanceof Map<?, ?> map) {
      for (Map.Entry<?, ?> e : map.entrySet()) {
        String key = String.valueOf(e.getKey()).toLowerCase(Locale.ROOT);
        if (FORBIDDEN.contains(key)) {
          throw new IllegalArgumentException("source body field '" + e.getKey() + "' is not allowed");
        }
        rejectSourceBody(e.getValue(), path + "." + e.getKey());
      }
    } else if (node instanceof List<?> list) {
      int i = 0;
      for (Object child : list) {
        rejectSourceBody(child, path + "[" + i++ + "]");
      }
    }
  }

  /**
   * ide_context.current_file 문자열을 꺼낸다. 없으면 null.
   *
   * @param data chat.submit data 맵
   */
  @SuppressWarnings("unchecked")
  public static String currentFile(Map<String, Object> data) {
    Object ctx = data.get("ide_context");
    if (!(ctx instanceof Map<?, ?> map)) {
      return null;
    }
    Object file = map.get("current_file");
    if (file == null) {
      return null;
    }
    String path = String.valueOf(file).trim();
    if (path.isEmpty() || path.contains("\n")) {
      return null;
    }
    return path;
  }

  /**
   * 새 질문 id를 만든다. protocol ID 종류를 추가하지 않는다.
   */
  public static String newQuestionId() {
    return "q-" + UUID.randomUUID();
  }

  /**
   * 슬라이스 2에서 쓰는 질문 템플릿. 세 타입을 모두 만들 수 있다.
   *
   * @param type SINGLE_SELECT / TEXT / CONFIRM
   */
  public static Question typedQuestion(String type) {
    return switch (type) {
      case Question.SINGLE_SELECT ->
          new Question(
              newQuestionId(),
              Question.SINGLE_SELECT,
              "이 요청을 어떻게 처리할까요?",
              List.of(
                  new QuestionOption("IMPLEMENT", "구현"),
                  new QuestionOption("REVIEW", "검토"),
                  new QuestionOption("EXPLAIN", "설명")));
      case Question.CONFIRM ->
          new Question(
              newQuestionId(),
              Question.CONFIRM,
              "이 작업을 이어서 진행할까요?",
              List.of());
      default ->
          new Question(
              newQuestionId(),
              Question.TEXT,
              "작업할 파일 경로를 알려주세요.",
              List.of());
    };
  }

  /**
   * 테스트에서 쓰기 쉽게 복사본 목록을 만든다.
   */
  public static List<String> supportedQuestionTypes() {
    List<String> types = new ArrayList<>();
    types.add(Question.SINGLE_SELECT);
    types.add(Question.TEXT);
    types.add(Question.CONFIRM);
    return types;
  }
}
