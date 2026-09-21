package com.oncode.server.api.grpc;

import java.util.List;
import java.util.Map;

/**
 * Envelope data_json용 최소 JSON 직렬화. proto/Jackson에 의존하지 않는다.
 */
final class EventJson {

  private EventJson() {}

  /**
   * 맵을 object JSON으로 쓴다. 순환 참조는 없다고 가정한다.
   *
   * @param data null이면 {@code {}}
   */
  static String write(Map<String, ?> data) {
    if (data == null) {
      return "{}";
    }
    StringBuilder out = new StringBuilder();
    writeValue(out, data);
    return out.toString();
  }

  private static void writeValue(StringBuilder out, Object value) {
    switch (value) {
      case null -> out.append("null");
      case String s -> out.append(quote(s));
      case Boolean b -> out.append(b);
      case Number n -> out.append(n);
      case Map<?, ?> map -> {
        out.append('{');
        boolean first = true;
        for (Map.Entry<?, ?> e : map.entrySet()) {
          if (!first) {
            out.append(',');
          }
          first = false;
          out.append(quote(String.valueOf(e.getKey()))).append(':');
          writeValue(out, e.getValue());
        }
        out.append('}');
      }
      case List<?> list -> {
        out.append('[');
        boolean first = true;
        for (Object item : list) {
          if (!first) {
            out.append(',');
          }
          first = false;
          writeValue(out, item);
        }
        out.append(']');
      }
      default -> out.append(quote(String.valueOf(value)));
    }
  }

  private static String quote(String value) {
    return "\"" + value.replace("\\", "\\\\").replace("\"", "\\\"") + "\"";
  }
}
