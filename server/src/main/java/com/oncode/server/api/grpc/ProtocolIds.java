package com.oncode.server.api.grpc;

import java.security.SecureRandom;

/**
 * gRPC Envelope에 넣는 프로토콜 ID. UUID v7이라서 Local Agent {@code ids.Validate}를 통과한다.
 * 워크플로 영속 ID({@code EntityIds} 12 hex)를 바꾸지 않는다.
 */
final class ProtocolIds {

  private static final SecureRandom RANDOM = new SecureRandom();

  private ProtocolIds() {}

  /**
   * MSG- 접두 UUID v7을 발급한다.
   *
   * @return 비어 있지 않음
   */
  static String message() {
    return "MSG-" + uuidV7();
  }

  /**
   * CALL- 접두 UUID v7을 발급한다. 게이트웨이 CALL-12hex와 다를 수 있다.
   *
   * @return 비어 있지 않음
   */
  static String call() {
    return "CALL-" + uuidV7();
  }

  /**
   * RFC 4122 variant의 UUIDv7 문자열을 만든다.
   *
   * @return 36자 hex+하이픈
   */
  static String uuidV7() {
    byte[] bytes = new byte[16];
    RANDOM.nextBytes(bytes);
    long ms = System.currentTimeMillis();
    bytes[0] = (byte) (ms >>> 40);
    bytes[1] = (byte) (ms >>> 32);
    bytes[2] = (byte) (ms >>> 24);
    bytes[3] = (byte) (ms >>> 16);
    bytes[4] = (byte) (ms >>> 8);
    bytes[5] = (byte) ms;
    bytes[6] = (byte) ((bytes[6] & 0x0f) | 0x70);
    bytes[8] = (byte) ((bytes[8] & 0x3f) | 0x80);
    return formatUuid(bytes);
  }

  private static String formatUuid(byte[] bytes) {
    String hex = hex(bytes);
    return hex.substring(0, 8)
        + "-"
        + hex.substring(8, 12)
        + "-"
        + hex.substring(12, 16)
        + "-"
        + hex.substring(16, 20)
        + "-"
        + hex.substring(20, 32);
  }

  private static String hex(byte[] bytes) {
    StringBuilder out = new StringBuilder(bytes.length * 2);
    for (byte b : bytes) {
      out.append(Character.forDigit((b >> 4) & 0xf, 16));
      out.append(Character.forDigit(b & 0xf, 16));
    }
    return out.toString();
  }
}
