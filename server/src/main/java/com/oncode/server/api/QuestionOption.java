package com.oncode.server.api;

/**
 * 질문 선택지. TEXT/CONFIRM에서는 쓰지 않는다.
 *
 * @param id 답변으로 돌아올 값
 * @param label IDE에 보일 문구
 */
public record QuestionOption(String id, String label) {}
