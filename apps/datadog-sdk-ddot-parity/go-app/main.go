package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	sqltrace "github.com/DataDog/dd-trace-go/contrib/database/sql/v2"
	httptrace "github.com/DataDog/dd-trace-go/contrib/net/http/v2"
	ddotel "github.com/DataDog/dd-trace-go/v2/ddtrace/opentelemetry"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/DataDog/dd-trace-go/v2/profiler"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel"
	otelattr "go.opentelemetry.io/otel/attribute"
)

var (
	allocationHold [][]byte
	allocationMu   sync.Mutex
	db             *sql.DB
)

func main() {
	service := envOr("OTEL_SERVICE_NAME", envOr("DD_SERVICE", "otel-go-ddot-parity"))
	environment := envOr("DD_ENV", "validation")
	version := envOr("DD_VERSION", "dd-trace-go-2.10.1-run1")

	provider := ddotel.NewTracerProvider()
	defer func() { _ = provider.Shutdown() }()
	otel.SetTracerProvider(provider)

	if err := profiler.Start(
		profiler.WithService(service),
		profiler.WithEnv(environment),
		profiler.WithVersion(version),
		profiler.WithProfileTypes(
			profiler.CPUProfile,
			profiler.HeapProfile,
			profiler.GoroutineProfile,
			profiler.MutexProfile,
			profiler.BlockProfile,
		),
	); err != nil {
		log.Fatalf("start profiler: %v", err)
	}
	defer profiler.Stop()

	var err error
	db, err = openDatabase(service)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelDatabase()
	if err := waitForDatabase(databaseContext); err != nil {
		log.Fatalf("wait for database: %v", err)
	}

	mux := httptrace.NewServeMux(httptrace.WithService(service))
	mux.HandleFunc("/health", healthEndpoint)
	mux.HandleFunc("/span", spanEndpoint)
	mux.HandleFunc("/probe", probeEndpoint)
	mux.HandleFunc("/cpu", cpuEndpoint)
	mux.HandleFunc("/allocate", allocationEndpoint)
	mux.HandleFunc("/goroutines", goroutineEndpoint)
	mux.HandleFunc("/db/query", databaseQueryEndpoint)
	mux.HandleFunc("/db/slow", databaseSlowEndpoint)
	mux.HandleFunc("/db/error", databaseErrorEndpoint)

	log.Printf("listening on :5000 service=%s version=%s", service, version)
	log.Fatal(http.ListenAndServe(":5000", mux))
}

func openDatabase(service string) (*sql.DB, error) {
	sqltrace.Register(
		"postgres",
		&pq.Driver{},
		sqltrace.WithService(service+"-postgres"),
		sqltrace.WithDBMPropagation(tracer.DBMPropagationModeFull),
	)
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&application_name=%s",
		envOr("POSTGRES_USER", "validation"),
		envOr("POSTGRES_PASSWORD", "validation"),
		envOr("POSTGRES_HOST", "postgres"),
		envOr("POSTGRES_DB_PORT", "5432"),
		envOr("POSTGRES_DB", "validation"),
		service,
	)
	connection, err := sqltrace.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	connection.SetMaxOpenConns(8)
	connection.SetMaxIdleConns(4)
	return connection, nil
}

func healthEndpoint(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339Nano)})
}

func spanEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("otel-go-validation").Start(r.Context(), "manual.span")
	span.SetAttributes(otelattr.String("demo.endpoint", "/span"))
	time.Sleep(150 * time.Millisecond)
	span.End()
	writeJSON(w, http.StatusOK, map[string]any{"span": "ok", "trace_context": ctx != nil})
}

func dynamicProbeTarget(value int) int {
	normalized := value
	return normalized*2 + 1
}

func probeEndpoint(w http.ResponseWriter, r *http.Request) {
	value := intQuery(r, "value", 7, 1, 10000)
	writeJSON(w, http.StatusOK, map[string]any{"input": value, "result": dynamicProbeTarget(value)})
}

func cpuEndpoint(w http.ResponseWriter, r *http.Request) {
	seconds := floatQuery(r, "seconds", 1.5, 0.1, 10)
	deadline := time.Now().Add(time.Duration(seconds * float64(time.Second)))
	iterations := 0
	checksum := 0
	for time.Now().Before(deadline) {
		checksum = (checksum + (iterations*31)%9973) % 1_000_003
		iterations++
	}
	writeJSON(w, http.StatusOK, map[string]any{"seconds": seconds, "iterations": iterations, "checksum": checksum})
}

func allocationEndpoint(w http.ResponseWriter, r *http.Request) {
	megabytes := intQuery(r, "mb", 8, 1, 64)
	block := make([]byte, megabytes*1024*1024)
	for index := 0; index < len(block); index += 4096 {
		block[index] = byte(index)
	}
	allocationMu.Lock()
	allocationHold = append(allocationHold, block)
	if len(allocationHold) > 4 {
		allocationHold = allocationHold[1:]
	}
	retained := len(allocationHold)
	allocationMu.Unlock()
	runtime.GC()
	writeJSON(w, http.StatusOK, map[string]any{"allocated_mb": megabytes, "retained_blocks": retained})
}

func goroutineEndpoint(w http.ResponseWriter, r *http.Request) {
	count := intQuery(r, "count", 100, 1, 1000)
	var mutex sync.Mutex
	var wg sync.WaitGroup
	for index := 0; index < count; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mutex.Lock()
			time.Sleep(2 * time.Millisecond)
			mutex.Unlock()
		}()
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, map[string]any{"goroutines": count})
}

func databaseQueryEndpoint(w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(r.Context(), "SELECT status, count(*) FROM orders WHERE customer_id = $1 GROUP BY status", 42)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defer rows.Close()
	result := make([]map[string]any, 0, 4)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		result = append(result, map[string]any{"status": status, "count": count})
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": result})
}

func databaseSlowEndpoint(w http.ResponseWriter, r *http.Request) {
	var slept any
	var count int
	err := db.QueryRowContext(r.Context(), "SELECT pg_sleep(1.25), count(*) FROM orders").Scan(&slept, &count)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count, "slept_seconds": 1.25})
}

func databaseErrorEndpoint(w http.ResponseWriter, r *http.Request) {
	_, err := db.ExecContext(r.Context(), "SELECT * FROM validation_table_that_does_not_exist")
	if err == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "not raised"})
		return
	}
	log.Printf("intentional database validation error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "intentional database validation error"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func intQuery(r *http.Request, name string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil {
		value = fallback
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func floatQuery(r *http.Request, name string, fallback, minimum, maximum float64) float64 {
	value, err := strconv.ParseFloat(r.URL.Query().Get(name), 64)
	if err != nil {
		value = fallback
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func waitForDatabase(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
