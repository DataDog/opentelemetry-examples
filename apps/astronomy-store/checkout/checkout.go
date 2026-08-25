package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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

type checkoutServer struct{}

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
	logger.Info("PlaceOrder",
		zap.String("user_id", req.UserID),
		zap.String("user_currency", req.UserCurrency),
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
	logger.Info("order placed",
		zap.String("demo.order.id", orderID),
		zap.String("demo.shipping.tracking.id", shippingTrackingID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(PlaceOrderResponse{Order: result}); err != nil {
		logger.Error("failed to encode response", zap.Error(err))
	}
}

func (cs *checkoutServer) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
