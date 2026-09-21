package com.oncode.server.inferencegateway;

import java.util.Objects;
import java.util.concurrent.ConcurrentHashMap;
import org.springframework.stereotype.Service;

/**
 * 실 vLLM 없이 프로파일별 JSON을 돌려준다. 프로덕션 라우팅은 이 클래스 뒤에 둔다.
 * 에이전트가 여기를 우회하지 못하게 {@link InferenceGateway}만 공개한다.
 */
@Service
public class ScriptedInferenceGateway implements InferenceGateway {

  private final ConcurrentHashMap<ModelProfile, String> scripts = new ConcurrentHashMap<>();

  /** 기본 스크립트를 채운다. */
  public ScriptedInferenceGateway() {
    scripts.put(
        ModelProfile.REASONING_HIGH,
        "{\"summary\":\"lock account after 5 failures\",\"status\":\"DRAFT\"}");
    scripts.put(
        ModelProfile.CODING,
        "{\"path\":\"src/AuthService.java\",\"operation\":\"MODIFY\",\"content\":\"class AuthService { /* lock */ }\"}");
    scripts.put(ModelProfile.REVIEW, "{\"verdict\":\"PASS\",\"findings\":[]}");
    scripts.put(ModelProfile.UTILITY, "{}");
  }

  /**
   * {@inheritDoc}
   */
  @Override
  public InferenceResponse complete(InferenceRequest request) {
    Objects.requireNonNull(request, "request");
    Objects.requireNonNull(request.modelProfile(), "modelProfile");
    if (request.userPrompt() == null || request.userPrompt().isBlank()) {
      throw new IllegalArgumentException("userPrompt must not be blank");
    }
    String json = scripts.get(request.modelProfile());
    if (json == null) {
      throw new IllegalStateException("no script for " + request.modelProfile());
    }
    return new InferenceResponse(request.modelProfile(), json);
  }
}
