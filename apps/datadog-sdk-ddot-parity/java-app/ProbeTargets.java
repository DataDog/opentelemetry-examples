public final class ProbeTargets {
  private ProbeTargets() {}

  private static int calculate(int value) {
    int normalized = Math.max(0, value);
    return normalized * 2 + 1;
  }

  public static int metricProbeTarget(int value) {
    return calculate(value);
  }

  public static int conditionalMetricProbeTarget(int value) {
    return calculate(value);
  }

  public static int spanProbeTarget(int value) {
    return calculate(value);
  }

  public static int spanTagProbeTarget(int value) {
    return calculate(value);
  }

  public static int logProbeTarget(int value) {
    return calculate(value);
  }
}
