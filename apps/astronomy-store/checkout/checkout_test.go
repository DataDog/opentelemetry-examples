package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type recordingKafkaProducer struct {
	message *sarama.ProducerMessage
}

func (p *recordingKafkaProducer) SendMessage(message *sarama.ProducerMessage) (int32, int64, error) {
	p.message = message
	return 0, 1, nil
}

func (p *recordingKafkaProducer) Close() error {
	return nil
}

func TestPlaceOrderPublishesOrderToKafka(t *testing.T) {
	logger = zap.NewNop()
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTextMapPropagator(previousPropagator)
	})

	producer := &recordingKafkaProducer{}
	server := checkoutServer{
		kafkaProducer: producer,
		kafkaTopic:    "orders",
	}
	request := httptest.NewRequest(http.MethodPost, "/checkout/place-order", bytes.NewBufferString(`{
		"user_id":"test-user",
		"user_currency":"USD",
		"address":{"street_address":"1 Galaxy Way","city":"Paris","country":"FR"},
		"email":"test@example.com",
		"credit_card":{
			"credit_card_number":"4111111111111111",
			"credit_card_cvv":123,
			"credit_card_expiration_year":2030,
			"credit_card_expiration_month":12
		}
	}`))
	request = request.WithContext(trace.ContextWithSpanContext(request.Context(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{1},
		TraceFlags: trace.FlagsSampled,
	})))
	response := httptest.NewRecorder()

	server.placeOrder(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if producer.message == nil {
		t.Fatal("expected an order to be published")
	}
	if producer.message.Topic != "orders" {
		t.Errorf("topic = %q, want %q", producer.message.Topic, "orders")
	}

	payload, err := producer.message.Value.Encode()
	if err != nil {
		t.Fatalf("encode Kafka message: %v", err)
	}
	var order OrderResult
	if err := json.Unmarshal(payload, &order); err != nil {
		t.Fatalf("unmarshal Kafka message: %v", err)
	}
	if order.OrderID == "" {
		t.Error("published order is missing order_id")
	}
	if order.ShippingTrackingID == "" {
		t.Error("published order is missing shipping_tracking_id")
	}

	for _, header := range producer.message.Headers {
		if string(header.Key) == "traceparent" {
			return
		}
	}
	t.Error("published order is missing traceparent header")
}
