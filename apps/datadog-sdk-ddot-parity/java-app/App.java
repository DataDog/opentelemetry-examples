import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.StatusCode;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.context.Scope;
import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.Executors;

public final class App {
  private static final Tracer TRACER =
      GlobalOpenTelemetry.get().getTracer("otel-java-validation", "1.0.0");
  private static final List<byte[]> ALLOCATION_HOLD = new ArrayList<>();
  private static final String SERVICE =
      System.getenv().getOrDefault("DD_SERVICE", "otel-java-ddot-parity");

  private App() {}

  public static void main(String[] args) throws Exception {
    Class.forName("org.postgresql.Driver");
    HttpServer server = HttpServer.create(new InetSocketAddress(5000), 0);
    server.setExecutor(Executors.newFixedThreadPool(8));
    server.createContext("/health", exchange -> respond(exchange, 200, "{\"status\":\"ok\"}"));
    server.createContext("/span", App::span);
    server.createContext("/probe/metric", exchange -> probe(exchange, "metric"));
    server.createContext("/probe/condition", exchange -> probe(exchange, "condition"));
    server.createContext("/probe/span", exchange -> probe(exchange, "span"));
    server.createContext("/probe/span-tag", exchange -> probe(exchange, "span-tag"));
    server.createContext("/probe/log", exchange -> probe(exchange, "log"));
    server.createContext("/cpu", App::cpu);
    server.createContext("/allocate", App::allocate);
    server.createContext("/exceptions", App::exceptions);
    server.createContext("/db/query", App::databaseQuery);
    server.createContext("/db/slow", App::databaseSlow);
    server.createContext("/db/error", App::databaseError);
    server.start();
    System.out.println("java validation service listening on 5000 at " + Instant.now());
  }

  private static Span startHttpSpan(String route) {
    Span span = TRACER.spanBuilder("validation.http " + route).startSpan();
    span.setAttribute("http.request.method", "GET");
    span.setAttribute("http.route", route);
    span.setAttribute("url.path", route);
    span.setAttribute("server.address", "otel-java-validation");
    return span;
  }

  private static void span(HttpExchange exchange) throws IOException {
    Span span = startHttpSpan("/span");
    try (Scope ignored = span.makeCurrent()) {
      span.setAttribute("validation.otel.api", true);
      busyWork(350);
      respond(exchange, 200, "{\"span\":\"ok\"}");
    } finally {
      span.end();
    }
  }

  private static void probe(HttpExchange exchange, String scenario) throws IOException {
    int value = intQuery(exchange, "value", 7);
    Span span = startHttpSpan("/probe/" + scenario);
    try (Scope ignored = span.makeCurrent()) {
      int result;
      switch (scenario) {
        case "metric" -> result = ProbeTargets.metricProbeTarget(value);
        case "condition" -> result = ProbeTargets.conditionalMetricProbeTarget(value);
        case "span" -> result = ProbeTargets.spanProbeTarget(value);
        case "span-tag" -> result = ProbeTargets.spanTagProbeTarget(value);
        case "log" -> result = ProbeTargets.logProbeTarget(value);
        default -> throw new IllegalArgumentException("unknown scenario " + scenario);
      }
      respond(exchange, 200, "{\"scenario\":\"" + scenario + "\",\"input\":" + value
          + ",\"result\":" + result + "}");
    } finally {
      span.end();
    }
  }

  private static void cpu(HttpExchange exchange) throws IOException {
    long milliseconds = Math.max(100, Math.min(10_000, intQuery(exchange, "milliseconds", 1500)));
    long iterations = busyWork(milliseconds);
    respond(exchange, 200, "{\"milliseconds\":" + milliseconds + ",\"iterations\":" + iterations + "}");
  }

  private static void allocate(HttpExchange exchange) throws IOException {
    int megabytes = Math.max(1, Math.min(64, intQuery(exchange, "mb", 8)));
    synchronized (ALLOCATION_HOLD) {
      ALLOCATION_HOLD.add(new byte[megabytes * 1024 * 1024]);
      if (ALLOCATION_HOLD.size() > 4) {
        ALLOCATION_HOLD.remove(0);
      }
    }
    respond(exchange, 200, "{\"allocated_mb\":" + megabytes + "}");
  }

  private static void exceptions(HttpExchange exchange) throws IOException {
    int count = Math.max(1, Math.min(5000, intQuery(exchange, "count", 250)));
    for (int index = 0; index < count; index++) {
      try {
        throw new IllegalArgumentException("intentional-profile-exception-" + (index % 5));
      } catch (IllegalArgumentException ignored) {
        // Caught intentionally so the profiler can attribute exception samples.
      }
    }
    respond(exchange, 200, "{\"caught\":" + count + "}");
  }

  private static void databaseQuery(HttpExchange exchange) throws IOException {
    Span span = startDatabaseSpan("SELECT", "orders");
    try (Scope ignored = span.makeCurrent();
        Connection connection = databaseConnection();
        PreparedStatement statement = connection.prepareStatement(
            "SELECT status, count(*) FROM orders WHERE customer_id = ? GROUP BY status")) {
      statement.setInt(1, 42);
      int rows = 0;
      try (ResultSet results = statement.executeQuery()) {
        while (results.next()) {
          rows++;
        }
      }
      respond(exchange, 200, "{\"rows\":" + rows + "}");
    } catch (SQLException error) {
      recordFailure(span, error);
      respond(exchange, 500, "{\"error\":\"SQLException\"}");
    } finally {
      span.end();
    }
  }

  private static void databaseSlow(HttpExchange exchange) throws IOException {
    Span span = startDatabaseSpan("SELECT", "orders");
    try (Scope ignored = span.makeCurrent();
        Connection connection = databaseConnection();
        Statement statement = connection.createStatement();
        ResultSet results = statement.executeQuery("SELECT pg_sleep(1.25), count(*) FROM orders")) {
      results.next();
      respond(exchange, 200, "{\"count\":" + results.getLong(2) + ",\"slept_seconds\":1.25}");
    } catch (SQLException error) {
      recordFailure(span, error);
      respond(exchange, 500, "{\"error\":\"SQLException\"}");
    } finally {
      span.end();
    }
  }

  private static void databaseError(HttpExchange exchange) throws IOException {
    Span span = startDatabaseSpan("SELECT", "validation_table_that_does_not_exist");
    try (Scope ignored = span.makeCurrent();
        Connection connection = databaseConnection();
        Statement statement = connection.createStatement()) {
      statement.executeQuery("SELECT * FROM validation_table_that_does_not_exist");
      respond(exchange, 500, "{\"error\":\"not raised\"}");
    } catch (SQLException error) {
      recordFailure(span, error);
      respond(exchange, 500, "{\"error\":\"SQLException\"}");
    } finally {
      span.end();
    }
  }

  private static Span startDatabaseSpan(String operation, String target) {
    Span span = TRACER.spanBuilder("validation.jdbc." + operation.toLowerCase()).startSpan();
    span.setAttribute("db.system.name", "postgresql");
    span.setAttribute("db.namespace", "validation");
    span.setAttribute("db.operation.name", operation);
    span.setAttribute("db.collection.name", target);
    span.setAttribute("server.address", System.getenv().getOrDefault("POSTGRES_HOST", "postgres"));
    span.setAttribute("server.port", 5432L);
    return span;
  }

  private static Connection databaseConnection() throws SQLException {
    String host = System.getenv().getOrDefault("POSTGRES_HOST", "postgres");
    String port = System.getenv().getOrDefault("POSTGRES_DB_PORT", "5432");
    String database = System.getenv().getOrDefault("POSTGRES_DB", "validation");
    String user = System.getenv().getOrDefault("POSTGRES_USER", "validation");
    String password = System.getenv().getOrDefault("POSTGRES_PASSWORD", "validation");
    String url = "jdbc:postgresql://" + host + ":" + port + "/" + database
        + "?ApplicationName=" + SERVICE;
    return DriverManager.getConnection(url, user, password);
  }

  private static void recordFailure(Span span, Exception error) {
    span.recordException(error);
    span.setStatus(StatusCode.ERROR, error.getMessage());
  }

  private static long busyWork(long milliseconds) {
    long deadline = System.nanoTime() + milliseconds * 1_000_000L;
    long iterations = 0;
    long checksum = 0;
    while (System.nanoTime() < deadline) {
      checksum = (checksum + (iterations * 31) % 9973) % 1_000_003;
      iterations++;
    }
    return iterations + checksum;
  }

  private static int intQuery(HttpExchange exchange, String key, int fallback) {
    String query = exchange.getRequestURI().getRawQuery();
    if (query == null) {
      return fallback;
    }
    for (String pair : query.split("&")) {
      String[] parts = pair.split("=", 2);
      if (parts.length == 2 && parts[0].equals(key)) {
        try {
          return Integer.parseInt(parts[1]);
        } catch (NumberFormatException ignored) {
          return fallback;
        }
      }
    }
    return fallback;
  }

  private static void respond(HttpExchange exchange, int status, String body) throws IOException {
    byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
    exchange.getResponseHeaders().set("Content-Type", "application/json");
    exchange.sendResponseHeaders(status, bytes.length);
    exchange.getResponseBody().write(bytes);
    exchange.close();
  }
}
