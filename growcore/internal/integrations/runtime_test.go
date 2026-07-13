package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func TestOpenMeteoRuntimeNormalizesForecast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/forecast" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("latitude") != "50.0000" {
			t.Errorf("latitude = %s", r.URL.Query().Get("latitude"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hourly":{"time":["2026-07-13T12:00"],"temperature_2m":[24.5],"relative_humidity_2m":[61],"surface_pressure":[1008.2]}}`))
	}))
	defer server.Close()

	bundle := Bundle{Runtime: RuntimeSpec{Type: "builtin", Handler: "open-meteo"}}
	result, err := runOperation(context.Background(), bundle, map[string]string{"baseUrl": server.URL}, "weather.forecast", map[string]any{"latitude": 50.0, "longitude": 14.0, "pastDays": 4})
	if err != nil {
		t.Fatal(err)
	}
	encoded := result.(struct {
		Temp     []domain.SeriesPoint `json:"temp"`
		Humidity []domain.SeriesPoint `json:"humidity"`
		Pressure []domain.SeriesPoint `json:"pressure"`
	})
	if len(encoded.Temp) != 1 || encoded.Temp[0].Value != 24.5 || encoded.Humidity[0].Value != 61 || encoded.Pressure[0].Value != 1008.2 {
		t.Fatalf("unexpected forecast: %#v", result)
	}
}
