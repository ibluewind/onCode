package com.oncode.server.inferencegateway;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import org.junit.jupiter.api.Test;

/**
 * 모델 ID가 아니라 프로파일로만 완료되는지 검증한다.
 */
class ScriptedInferenceGatewayTest {

  /**
   * 프로파일마다 구조화 JSON을 돌려준다.
   */
  @Test
  void completesByModelProfile() {
    ScriptedInferenceGateway gateway = new ScriptedInferenceGateway();
    InferenceResponse design =
        gateway.complete(
            new InferenceRequest(ModelProfile.REASONING_HIGH, "", "lock account", "design"));
    assertThat(design.modelProfile()).isEqualTo(ModelProfile.REASONING_HIGH);
    assertThat(design.outputJson()).contains("summary");
  }

  /**
   * userPrompt가 비면 거부한다.
   */
  @Test
  void rejectsBlankPrompt() {
    ScriptedInferenceGateway gateway = new ScriptedInferenceGateway();
    assertThatThrownBy(
            () ->
                gateway.complete(
                    new InferenceRequest(ModelProfile.CODING, "", "  ", "proposed_change")))
        .isInstanceOf(IllegalArgumentException.class);
  }
}
