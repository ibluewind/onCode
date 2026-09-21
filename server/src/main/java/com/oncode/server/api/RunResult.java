package com.oncode.server.api;

/**
 * Local Agent가 보낸 빌드/테스트 요약. 없으면 표시하지 않는다.
 *
 * @param kind BUILD 또는 TEST
 * @param status SUCCESS / FAILED / UNKNOWN
 * @param summary 짧은 결과 문구. CoT 아님
 */
public record RunResult(String kind, String status, String summary) {

  public static final String BUILD = "BUILD";
  public static final String TEST = "TEST";
}
