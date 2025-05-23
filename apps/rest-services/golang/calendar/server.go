package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Server struct {
	name             string
	tracer           trace.Tracer
	rnd              *rand.Rand
	apiCounter       metric.Int64Counter
	latency          metric.Float64Histogram
	memoryGauge      metric.Int64ObservableGauge
	activeUsersGauge metric.Int64ObservableGauge
	activeUsersCount *atomic.Int64
}

func NewServer(name string, mp metric.MeterProvider, tp trace.TracerProvider) (*Server, error) {
	tracer := tp.Tracer(name)
	meter := mp.Meter(name)
	apiCounter, err := meter.Int64Counter(
		name+".api.counter",
		metric.WithDescription("Number of API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, err
	}
	histogram, err := meter.Float64Histogram(
		name+".task.duration",
		metric.WithDescription("The duration of task execution."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	memoryGauge, err := meter.Int64ObservableGauge(
		name+".memory.heap",
		metric.WithDescription(
			"Memory usage of the allocated heap objects.",
		),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			o.Observe(int64(m.HeapAlloc))
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	var activeUsersCount atomic.Int64
	activeUsersGauge, err := meter.Int64ObservableGauge(
		name+".active.users.gauge",
		metric.WithDescription(
			"active users gauge",
		),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(activeUsersCount.Load())
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Server{
		name:             name,
		tracer:           tracer,
		rnd:              rand.New(rand.NewSource(time.Now().Unix())),
		apiCounter:       apiCounter,
		latency:          histogram,
		memoryGauge:      memoryGauge,
		activeUsersGauge: activeUsersGauge,
		activeUsersCount: &activeUsersCount,
	}, nil
}

func (s *Server) getDate(ctx context.Context) string {
	_, span := s.tracer.Start(
		ctx,
		"getDate",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("peer.service", "random-date-service")),
	)
	defer span.End()
	logger.Info(
		"getting random date",
		zap.String("dd.trace_id", span.SpanContext().TraceID().String()),
		zap.String("dd.span_id", span.SpanContext().SpanID().String()),
		zap.Any("context", ctx),
	)
	dayOffset := rand.Intn(365)
	startDate := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.Local)
	day := startDate.AddDate(0, 0, dayOffset)

	d := day.Format(time.DateOnly)
	logger.Info(
		"random date",
		zap.String("date", d),
		zap.String("dd.trace_id", span.SpanContext().TraceID().String()),
		zap.String("dd.span_id", span.SpanContext().SpanID().String()),
		zap.Any("context", ctx),
	)
	return d
}

type response struct {
	Date  string `json:"date"`
	Error string `json:"error_message"`
}

func (s *Server) calendarHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()
	s.activeUsersCount.Add(1)
	defer func() {
		s.activeUsersCount.Add(-1)
		duration := time.Since(start)
		s.latency.Record(ctx, float64(duration))
	}()
	s.apiCounter.Add(r.Context(), 1)

	timer := time.NewTimer(time.Millisecond * time.Duration(s.rnd.Int63n(900)))

	defer func() {
		timer.Stop()
	}()

	select {
	case <-ctx.Done():
		resp := response{
			Error: fmt.Sprintf("time out %s", ctx.Err()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.Error("encoding resp", zap.Error(err))
		}
	case <-timer.C:
		dt := s.getDate(ctx)
		resp := response{
			Date: dt,
		}
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			logger.Error("encoding resp", zap.Error(err))
		}
	}
}
