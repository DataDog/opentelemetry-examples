package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// defaultShippingCostUnits/Nanos is a stand-in shipping quote of 8.99: this demo
// does not run a separate cart/shipping service, so checkout charges a flat rate.
const (
	defaultShippingCostUnits = 8
	defaultShippingCostNanos = 990000000
)

// The request/response shapes below mirror the CheckoutService.PlaceOrder RPC
// from the OpenTelemetry Demo protobuf definitions:
// https://github.com/open-telemetry/opentelemetry-demo/blob/main/pb/demo.proto

type Address struct {
	StreetAddress string `json:"street_address"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	ZipCode       string `json:"zip_code"`
}

type CreditCardInfo struct {
	CreditCardNumber          string `json:"credit_card_number"`
	CreditCardCvv             int32  `json:"credit_card_cvv"`
	CreditCardExpirationYear  int32  `json:"credit_card_expiration_year"`
	CreditCardExpirationMonth int32  `json:"credit_card_expiration_month"`
}

type Money struct {
	CurrencyCode string `json:"currency_code"`
	Units        int64  `json:"units"`
	Nanos        int32  `json:"nanos"`
}

type PlaceOrderRequest struct {
	UserID       string         `json:"user_id"`
	UserCurrency string         `json:"user_currency"`
	Address      Address        `json:"address"`
	Email        string         `json:"email"`
	CreditCard   CreditCardInfo `json:"credit_card"`
}

type OrderResult struct {
	OrderID            string  `json:"order_id"`
	ShippingTrackingID string  `json:"shipping_tracking_id"`
	ShippingCost       Money   `json:"shipping_cost"`
	ShippingAddress    Address `json:"shipping_address"`
}

type PlaceOrderResponse struct {
	Order OrderResult `json:"order"`
}

type kafkaProducer interface {
	WriteMessages(ctx context.Context, messages ...kafka.Message) error
	Close() error
}

type checkoutServer struct {
	kafkaProducer kafkaProducer
	kafkaTopic    string
}

func (cs *checkoutServer) placeOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.String("user.id", req.UserID),
		attribute.String("demo.user_context.selected_currency", req.UserCurrency),
		attribute.String("user.email", req.Email),
		attribute.Int("demo.payment.card_cvv", int(req.CreditCard.CreditCardCvv)),
	)
	logger.InfoContext(ctx, "PlaceOrder",
		slog.String("user.id", req.UserID),
		slog.String("user.currency", req.UserCurrency),
	)

	orderID := uuid.NewString()
	shippingTrackingID := uuid.NewString()

	result := OrderResult{
		OrderID:            orderID,
		ShippingTrackingID: shippingTrackingID,
		ShippingCost: Money{
			CurrencyCode: req.UserCurrency,
			Units:        defaultShippingCostUnits,
			Nanos:        defaultShippingCostNanos,
		},
		ShippingAddress: req.Address,
	}

	span.SetAttributes(
		attribute.String("demo.order.id", orderID),
		attribute.String("demo.shipping.tracking.id", shippingTrackingID),
	)
	logger.InfoContext(ctx, "order placed",
		slog.String("demo.order.id", orderID),
		slog.String("demo.shipping.tracking.id", shippingTrackingID),
	)

	if err := cs.publishOrder(ctx, result); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to publish order to Kafka")
		logger.ErrorContext(ctx, "failed to publish order to Kafka",
			slog.String("demo.order.id", orderID),
			slog.Any("error", err),
		)
		http.Error(w, "failed to place order", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(PlaceOrderResponse{Order: result}); err != nil {
		logger.ErrorContext(ctx, "failed to encode response", slog.Any("error", err))
	}
}

func (cs *checkoutServer) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func newKafkaProducer(broker, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(broker),
		Topic:                  topic,
		RequiredAcks:           kafka.RequireAll,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
}

func (cs *checkoutServer) publishOrder(ctx context.Context, order OrderResult) error {
	payload, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}

	message := kafka.Message{
		Key:   []byte(order.OrderID),
		Value: payload,
	}

	err = cs.kafkaProducer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("send Kafka message: %w", err)
	}
	return nil
}
