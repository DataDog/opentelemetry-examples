package main

import (
	"net/http/httptest"
	"testing"
)

func TestDynamicProbeTarget(t *testing.T) {
	if got, want := dynamicProbeTarget(10), 21; got != want {
		t.Fatalf("dynamicProbeTarget(10) = %d, want %d", got, want)
	}
}

func TestQueryBounds(t *testing.T) {
	request := httptest.NewRequest("GET", "/?value=500", nil)
	if got, want := intQuery(request, "value", 7, 1, 100), 100; got != want {
		t.Fatalf("intQuery upper bound = %d, want %d", got, want)
	}

	request = httptest.NewRequest("GET", "/?seconds=0.01", nil)
	if got, want := floatQuery(request, "seconds", 1.5, 0.1, 10), 0.1; got != want {
		t.Fatalf("floatQuery lower bound = %f, want %f", got, want)
	}
}
