package com.oncode.sample;

public final class Greeter {
  private Greeter() {}

  public static String greet(String name) {
    if (name == null || name.isBlank()) {
      return "Hello";
    }
    return "Hello, " + name;
  }
}
