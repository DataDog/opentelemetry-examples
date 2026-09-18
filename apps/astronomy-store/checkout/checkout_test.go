package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/segmentio/kafka-go"
)

type recordingKafkaProducer struct {
	message kafka.Message
}

func (p *recordingKafkaProducer) WriteMessages(_ context.Context, messages ...kafka.Message) error {
	if len(messages) > 0 {
		p.message = messages[0]
	}
	return nil
}

func (p *recordingKafkaProducer) Close() error {
	return nil
}

func TestNewKafkaProducerAllowsTopicCreation(t *testing.T) {
	producer := newKafkaProducer("kafka:9092", "orders")
	defer producer.Close()

	if !producer.AllowAutoTopicCreation {
		t.Error("AllowAutoTopicCreation = false, want true")
	}
}

func TestPlaceOrderPublishesOrderToKafka(t *testing.T) {
	logger = slog.New(slog.NewTextHandler(io.Discard, nil))
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
	response := httptest.NewRecorder()

	server.placeOrder(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if producer.message.Topic != "" {
		t.Errorf("message topic = %q, want empty because the writer supplies it", producer.message.Topic)
	}
	if producer.message.Key == nil {
		t.Fatal("expected an order to be published")
	}

	var order OrderResult
	if err := json.Unmarshal(producer.message.Value, &order); err != nil {
		t.Fatalf("unmarshal Kafka message: %v", err)
	}
	if order.OrderID == "" {
		t.Error("published order is missing order_id")
	}
	if order.ShippingTrackingID == "" {
		t.Error("published order is missing shipping_tracking_id")
	}
}
