package com.oncode.sample;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;

class GreeterTest {
  @Test
  void greetsName() {
    assertEquals("Hello, onCode", Greeter.greet("onCode"));
  }

  @Test
  void greetsBlank() {
    assertEquals("Hello", Greeter.greet(" "));
  }
}
