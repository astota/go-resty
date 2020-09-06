package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/astota/go-logging"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
)

func TestRequestTracer(t *testing.T) {
	teardownTest := setupTest(t)
	defer teardownTest(t)

	tracer := mocktracer.New()
	opentracing.SetGlobalTracer(
		tracer,
	)

	tests := []struct {
		name string
		item string
	}{
		{"no span", ""},
		{"defined root span", "tracing_test"},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			tracer.Reset()
			handler := func(c echo.Context) error {
				span, _ := opentracing.StartSpanFromContext(c.Request().Context(), "tracing_test_handler")
				defer span.Finish()

				if item := span.BaggageItem(tst.item); tst.item != "" && item != "tracer_test_item" {
					t.Errorf("%s: invalid tracer baggage item '%s'", tst.name, item)
				}

				c.String(http.StatusOK, "")
				return nil
			}
			req := initTracerTestRequest(t)

			var root *mocktracer.MockSpan
			if tst.item != "" {
				root = addTestSpan(tst.item, req)
			}

			resp := callTracerTestRequest(t, req, handler)
			if resp.Code != http.StatusOK {
				t.Errorf("%s: incorrect output status: %d", tst.name, resp.Code)
			}

			if root != nil {
				root.Finish()
			}
		})
	}
}

func TestInitGlobalTracer(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		error string
	}{
		{"no environment variable", map[string]string{}, ""},
		{"empty tracer type", map[string]string{"TRACER_SERVICE": ""}, "tracer type is empty"},
		{"tracer type datadog", map[string]string{"TRACER_SERVICE": "datadog"}, ""},
		{"tracer type jaeger", map[string]string{"TRACER_SERVICE": "jaeger"}, ""},
	}
	logger := logging.NewLogger()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Unsetenv("TRACER_SERVICE")
			os.Unsetenv("TRACER_HOST")
			os.Unsetenv("TRACER_SAMPLER")
			os.Unsetenv("TRACER_SAMPLER_VALUE")
			for k, v := range test.env {
				os.Setenv(k, v)
			}

			closer, err := InitGlobalTracer("test", logger)
			if err != nil || test.error != "" {
				if err != nil && !strings.Contains(err.Error(), test.error) {
					t.Errorf("incorrect error got: '%s', expected: '%s'", err.Error(), test.error)
				} else if err != nil && test.error == "" {
					t.Errorf("got error: '%s' even there should be", err.Error())
				}
			} else {
				span := opentracing.GlobalTracer().StartSpan("test span")
				defer span.Finish()
			}
			closer.Close()
		})
	}
}

// add baggage item to test span
func addTestSpan(item string, req *http.Request) *mocktracer.MockSpan {
	span := opentracing.StartSpan("mock_span")
	span.SetBaggageItem(item, "tracer_test_item")

	// add span to request
	opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header))

	return span.(*mocktracer.MockSpan)
}

// Calls test request
func callTracerTestRequest(t *testing.T, req *http.Request, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	app := echo.New()
	app.Logger.SetLevel(99)
	app.Use(RequestTracer())
	app.GET("/tracing", handler)

	resp := httptest.NewRecorder()
	app.ServeHTTP(resp, req)

	return resp
}

// Initialize tracer test request
func initTracerTestRequest(t *testing.T) *http.Request {
	t.Helper()
	logger := logging.NewLogger()
	req, _ := http.NewRequest("GET", "/tracing", nil)
	ctx := logging.SetLogger(context.Background(), logger)
	ctx = setDefaultContext(ctx, DefaultContext{
		OrganizationID: "123",
	})
	return req.WithContext(ctx)
}

func TestTracerConfig(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		expectError bool
	}{
		{"no tracer type config", map[string]string{}, false},
		{"tarcer type is empty", map[string]string{"TRACER_SERVICE": ""}, true},
		{"tracer type jaeger", map[string]string{"TRACER_SERVICE": "jaeger"}, false},
		{"tracer type datadog", map[string]string{"TRACER_SERVICE": "datadog"}, false},
		{"jaeger with hostname", map[string]string{"TRACER_SERVICE": "jaeger", "TRACER_HOST": "jaeger.svc"}, false},
		{"datadog with hostname", map[string]string{"TRACER_SERVICE": "datadog", "TRACER_HOST": "datadog.svc"}, false},
		{"jaeger with sampler", map[string]string{"TRACER_SERVICE": "jaeger", "TRACER_SAMPLER": "CONSTANT"}, false},
		{"datadog with sampler", map[string]string{"TRACER_SERVICE": "datadog", "TRACER_SAMPLER": "PROBABILISTIC"}, false},
		{"jaeger with invalid sampler", map[string]string{"TRACER_SERVICE": "jaeger", "TRACER_SAMPLER": ""}, true},
		{"datadog with invalid sampler", map[string]string{"TRACER_SERVICE": "datadog", "TRACER_SAMPLER": "invalid"}, true},
		{"jaeger with sampler and value", map[string]string{"TRACER_SERVICE": "jaeger", "TRACER_SAMPLER": "CONSTANT", "TRACER_SAMPLER_VALUE": "0.5"}, false},
		{"datadog with sampler and value", map[string]string{"TRACER_SERVICE": "datadog", "TRACER_SAMPLER": "PROBABILISTIC", "TRACER_SAMPLER_VALUE": "0.1"}, false},
		{"jaeger with env tag", map[string]string{"TRACER_SERVICE": "jaeger", "TRACER_ENVIRONMENT": "env"}, false},
		{"datadog with env", map[string]string{"TRACER_SERVICE": "datadog", "TRACER_ENVIRONMENT": "env"}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Unsetenv("TRACER_SERVICE")
			os.Unsetenv("TRACER_HOST")
			os.Unsetenv("TRACER_SAMPLER")
			os.Unsetenv("TRACER_SAMPLER_VALUE")
			for k, v := range test.env {
				os.Setenv(k, v)
			}

			cfg, err := getTracerConfig()
			if (err != nil) != test.expectError {
				t.Errorf("unexpected error value error exists: %v, expected: %v", err != nil, test.expectError)
			} else if test.expectError == false {
				if cfg.tracer != test.env["TRACER_SERVICE"] {
					t.Errorf("unexpected tracer, got: '%s', expected: '%s'", cfg.tracer, test.env["TRACER_SERVICE"])
				}
				host := "localhost"
				if v, exists := test.env["TRACER_HOST"]; exists {
					host = v
				}
				if cfg.hostName != host {
					t.Errorf("unexpected tracer hostname, got: '%s', expected: '%s'", cfg.hostName, host)
				}
				sampler := "CONSTANT"
				if v, exists := test.env["TRACER_SAMPLER"]; exists {
					sampler = v
				}
				if string(cfg.sampler) != sampler {
					t.Errorf("unexpected sampler, got: '%s', expected: '%s'", cfg.sampler, sampler)
				}

				env := "undefined"
				if v, exists := test.env["TRACER_ENVIRONMENT"]; exists && v != "" {
					env = v
				}
				if cfg.tags[environmentKey] != env {
					t.Errorf("unexcpected environment name, got: '%s', expected: '%s", cfg.tags[environmentKey], env)
				}
			}

		})
	}
}
