package com.oncode.server.toolgateway;

/**
 * ToolResponse payload에 대응하는 결과.
 *
 * @param callId 요청과 같은 CALL-...
 * @param status SUCCESS / FAILED / TIMEOUT
 * @param resultJson 성공 시 JSON 본문. 아니면 null
 * @param error 실패·타임아웃 시 구조화 오류. 성공이면 null
 * @param durationMs 소요 시간(ms). 0 이상
 */
public record ToolResponse(
    String callId, String status, String resultJson, ToolError error, long durationMs) {

  /**
   * 성공 응답을 만든다.
   *
   * @param callId CALL-...
   * @param resultJson JSON 본문
   * @param durationMs 소요 ms
   * @return status=SUCCESS
   */
  public static ToolResponse success(String callId, String resultJson, long durationMs) {
    return new ToolResponse(callId, "SUCCESS", resultJson, null, durationMs);
  }

  /**
   * 실패 또는 타임아웃 응답을 만든다.
   *
   * @param callId CALL-...
   * @param status FAILED 또는 TIMEOUT
   * @param error 구조화 오류. null 불가
   * @param durationMs 소요 ms
   * @return 오류 응답
   */
  public static ToolResponse failed(
      String callId, String status, ToolError error, long durationMs) {
    return new ToolResponse(callId, status, null, error, durationMs);
  }
}
