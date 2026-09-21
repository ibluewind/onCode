package com.oncode.server.api;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HexFormat;

/**
 * 설계 본문·actual diff 바인딩용 SHA-256. 워크스페이스 파일을 읽지 않는다.
 */
public final class ContentHash {

  private ContentHash() {}

  /**
   * UTF-8 바이트의 {@code sha256:<hex>}를 만든다.
   *
   * @param text null이면 빈 문자열과 같다
   */
  public static String sha256(String text) {
    String raw = text == null ? "" : text;
    try {
      MessageDigest digest = MessageDigest.getInstance("SHA-256");
      return "sha256:" + HexFormat.of().formatHex(digest.digest(raw.getBytes(StandardCharsets.UTF_8)));
    } catch (NoSuchAlgorithmException ex) {
      throw new IllegalStateException("SHA-256 required", ex);
    }
  }
}
