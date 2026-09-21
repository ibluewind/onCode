package com.oncode.server.persistence;

import java.util.UUID;

/**
 * 영속 엔티티 식별자를 만든다. 프로토콜 ID 형식(WI-/WF-/TR-/APR-/CALL-/CHG-)만 맞추고
 * 전역 채번은 하지 않는다.
 */
public final class EntityIds {

  private EntityIds() {}

  /**
   * 워크 아이템 ID를 발급한다.
   *
   * @return {@code WI-}로 시작하는 12자리 hex
   */
  public static String workItem() {
    return "WI-" + suffix();
  }

  /**
   * 워크플로 ID를 발급한다.
   *
   * @return {@code WF-}로 시작하는 12자리 hex
   */
  public static String workflow() {
    return "WF-" + suffix();
  }

  /**
   * 전이 ID를 발급한다.
   *
   * @return {@code TR-}로 시작하는 12자리 hex
   */
  public static String transition() {
    return "TR-" + suffix();
  }

  /**
   * 승인 ID를 발급한다.
   *
   * @return {@code APR-}로 시작하는 12자리 hex
   */
  public static String approval() {
    return "APR-" + suffix();
  }

  /**
   * 도구 호출 ID를 발급한다.
   *
   * @return {@code CALL-}로 시작하는 12자리 hex
   */
  public static String call() {
    return "CALL-" + suffix();
  }

  /**
   * 변경 세트 ID를 발급한다.
   *
   * @return {@code CHG-}로 시작하는 12자리 hex
   */
  public static String changeSet() {
    return "CHG-" + suffix();
  }

  /** UUID에서 짧은 대문자 hex를 잘라낸다. */
  private static String suffix() {
    return UUID.randomUUID().toString().replace("-", "").substring(0, 12).toUpperCase();
  }
}
