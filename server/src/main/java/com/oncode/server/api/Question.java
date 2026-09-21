package com.oncode.server.api;

import java.util.List;

/**
 * 사용자에게 보여줄 질문. 워크플로 상태는 바꾸지 않는다.
 *
 * @param questionId 연결 단위 식별자
 * @param questionType SINGLE_SELECT, TEXT, CONFIRM
 * @param prompt 표시 문구. 비면 안 된다
 * @param options SINGLE_SELECT일 때만 항목. 그 외는 빈 목록
 */
public record Question(
    String questionId, String questionType, String prompt, List<QuestionOption> options) {

  public static final String SINGLE_SELECT = "SINGLE_SELECT";
  public static final String TEXT = "TEXT";
  public static final String CONFIRM = "CONFIRM";
}
