package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func runTest(ctx context.Context, b Bundle, cfg map[string]string) error {
	if b.Runtime.Type == "builtin" && b.Runtime.Handler == "ollama" {
		_, err := ollamaRequest(ctx, cfg, http.MethodGet, "/api/version", nil)
		return err
	}
	if b.Runtime.Type == "builtin" && b.Runtime.Handler == "open-meteo" {
		_, err := openMeteoRequest(ctx, cfg, map[string]any{"latitude": 50.0755, "longitude": 14.4378, "pastDays": 0, "forecastDays": 1})
		return err
	}
	if b.Runtime.Type == "http" && b.Runtime.Test != nil {
		_, err := declarativeRequest(ctx, *b.Runtime.Test, cfg, map[string]any{})
		return err
	}
	return fmt.Errorf("bundle %s has no connection test", b.ID)
}

func runOperation(ctx context.Context, b Bundle, cfg map[string]string, cap string, input map[string]any) (any, error) {
	if b.Runtime.Type == "builtin" && b.Runtime.Handler == "ollama" {
		switch cap {
		case "ai.models":
			return ollamaRequest(ctx, cfg, http.MethodGet, "/api/tags", nil)
		case "ai.chat", "ai.vision":
			body := map[string]any{}
			for k, v := range input {
				body[k] = v
			}
			if _, ok := body["model"]; !ok {
				body["model"] = cfg["model"]
			}
			if _, ok := body["stream"]; !ok {
				body["stream"] = false
			}
			return ollamaRequest(ctx, cfg, http.MethodPost, "/api/chat", body)
		}
	}
	if b.Runtime.Type == "builtin" && b.Runtime.Handler == "open-meteo" && cap == "weather.forecast" {
		return openMeteoRequest(ctx, cfg, input)
	}
	if b.Runtime.Type == "http" {
		op, ok := b.Runtime.Operations[cap]
		if !ok {
			return nil, fmt.Errorf("no runtime operation for %s", cap)
		}
		return declarativeRequest(ctx, op, cfg, input)
	}
	return nil, fmt.Errorf("unsupported runtime %q", b.Runtime.Type)
}

func openMeteoRequest(ctx context.Context, cfg map[string]string, input map[string]any) (any, error) {
	lat, okLat := number(input["latitude"])
	lon, okLon := number(input["longitude"])
	if !okLat || !okLon || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return nil, fmt.Errorf("valid latitude and longitude are required")
	}
	pastDays, _ := number(input["pastDays"])
	forecastDays, ok := number(input["forecastDays"])
	if !ok || forecastDays < 1 {
		forecastDays = 2
	}
	base := strings.TrimRight(cfg["baseUrl"], "/")
	endpoint := base + "/v1/forecast?" + url.Values{
		"latitude":      {strconv.FormatFloat(lat, 'f', 4, 64)},
		"longitude":     {strconv.FormatFloat(lon, 'f', 4, 64)},
		"hourly":        {"temperature_2m,relative_humidity_2m,surface_pressure"},
		"past_days":     {strconv.Itoa(max(0, min(92, int(pastDays))))},
		"forecast_days": {strconv.Itoa(max(1, min(16, int(forecastDays))))},
		"timezone":      {"UTC"},
	}.Encode()
	raw, err := doJSON(ctx, http.MethodGet, endpoint, nil, func(*http.Request) {}, timeout(cfg["timeoutSeconds"]))
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var response struct {
		Hourly struct {
			Time     []string  `json:"time"`
			Temp     []float64 `json:"temperature_2m"`
			Humidity []float64 `json:"relative_humidity_2m"`
			Pressure []float64 `json:"surface_pressure"`
		} `json:"hourly"`
	}
	if err := json.Unmarshal(encoded, &response); err != nil {
		return nil, err
	}
	out := struct {
		Temp     []domain.SeriesPoint `json:"temp"`
		Humidity []domain.SeriesPoint `json:"humidity"`
		Pressure []domain.SeriesPoint `json:"pressure"`
	}{}
	for i, stamp := range response.Hourly.Time {
		t, err := time.Parse("2006-01-02T15:04", stamp)
		if err != nil {
			continue
		}
		if i < len(response.Hourly.Temp) && !math.IsNaN(response.Hourly.Temp[i]) {
			out.Temp = append(out.Temp, domain.SeriesPoint{Time: t, Value: response.Hourly.Temp[i]})
		}
		if i < len(response.Hourly.Humidity) && !math.IsNaN(response.Hourly.Humidity[i]) {
			out.Humidity = append(out.Humidity, domain.SeriesPoint{Time: t, Value: response.Hourly.Humidity[i]})
		}
		if i < len(response.Hourly.Pressure) && !math.IsNaN(response.Hourly.Pressure[i]) {
			out.Pressure = append(out.Pressure, domain.SeriesPoint{Time: t, Value: response.Hourly.Pressure[i]})
		}
	}
	return out, nil
}

func number(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(v, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func ollamaRequest(ctx context.Context, cfg map[string]string, method, path string, body any) (any, error) {
	base := strings.TrimRight(cfg["baseUrl"], "/")
	if _, err := url.ParseRequestURI(base); err != nil {
		return nil, fmt.Errorf("invalid Ollama URL: %w", err)
	}
	return doJSON(ctx, method, base+path, body, func(req *http.Request) {
		if token := cfg["apiKey"]; token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}, timeout(cfg["timeoutSeconds"]))
}

func declarativeRequest(ctx context.Context, spec HTTPRequest, cfg map[string]string, input map[string]any) (any, error) {
	target := cfg[spec.URLField]
	if target == "" {
		return nil, fmt.Errorf("URL field %q is empty", spec.URLField)
	}
	body := expandValue(spec.Body, cfg, input)
	return doJSON(ctx, normalizedMethod(spec.Method), target, body, func(req *http.Request) {
		for k, v := range spec.Headers {
			expanded := expandString(v, cfg, input)
			if expanded != "" && !strings.HasSuffix(expanded, "Bearer ") {
				req.Header.Set(k, expanded)
			}
		}
	}, timeout(cfg["timeoutSeconds"]))
}

func doJSON(ctx context.Context, method, target string, body any, headers func(*http.Request), limit time.Duration) (any, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "GrowRig/1 integration-runtime")
	headers(req)
	client := &http.Client{Timeout: limit}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		message := strings.TrimSpace(string(raw))
		if len(message) > 300 {
			message = message[:300]
		}
		return nil, fmt.Errorf("remote service returned %s: %s", res.Status, message)
	}
	if len(raw) == 0 {
		return map[string]any{"ok": true, "status": res.StatusCode}, nil
	}
	var out any
	if json.Unmarshal(raw, &out) == nil {
		return out, nil
	}
	return map[string]any{"ok": true, "status": res.StatusCode, "body": string(raw)}, nil
}

func timeout(value string) time.Duration {
	if value == "" {
		return 15 * time.Second
	}
	var seconds int
	if _, err := fmt.Sscanf(value, "%d", &seconds); err != nil || seconds < 1 || seconds > 120 {
		return 15 * time.Second
	}
	return time.Duration(seconds) * time.Second
}
func expandString(value string, cfg map[string]string, input map[string]any) string {
	for k, v := range cfg {
		value = strings.ReplaceAll(value, "{{config."+k+"}}", v)
	}
	for k, v := range input {
		value = strings.ReplaceAll(value, "{{input."+k+"}}", fmt.Sprint(v))
	}
	return value
}
func expandValue(value any, cfg map[string]string, input map[string]any) any {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "{{input.") && strings.HasSuffix(v, "}}") {
			if raw, ok := input[strings.TrimSuffix(strings.TrimPrefix(v, "{{input."), "}}")]; ok {
				return raw
			}
		}
		return expandString(v, cfg, input)
	case map[string]any:
		out := map[string]any{}
		for k, item := range v {
			out[k] = expandValue(item, cfg, input)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = expandValue(item, cfg, input)
		}
		return out
	default:
		return value
	}
}
