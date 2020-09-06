package rest

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/astota/go-logging"
	middleware "github.com/foodiefm/opentracing/contrib/github.com/labstack/echo"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
)

const (
	environmentKey = "environment"
)

// RequestTracer creates OpenTracing span to incoming requests
func RequestTracer() echo.MiddlewareFunc {
	inject := func(ctx context.Context, span opentracing.Span) context.Context {
		if fctx, err := GetDefaultContext(ctx); err == nil {
			span.SetTag("http.BMG-Organization-Id", fctx.OrganizationID)
			span.SetTag("http.BMG-Request-Id", fctx.RequestID)
		}

		return ctx
	}
	return middleware.RequestTracer(inject)
}

// InitGlobalTracer initialises global OpenTracing tracer.
// New span can be then created using OpenTracing API.
func InitGlobalTracer(name string, logger logging.Logger) (io.Closer, error) {

	cfg, err := getTracerConfig()
	if err != nil {
		return noopCloser{}, err
	}
	cfg.logger = logger
	cfg.serviceName = name

	var closer io.Closer
	var tracer opentracing.Tracer

	switch cfg.tracer {
	case "datadog":
		tracer, closer, err = initDatadogTracer(cfg)
	case "jaeger":
		tracer, closer, err = initJaegerTracer(cfg)
	default:
		tracer = opentracing.NoopTracer{}
		closer = noopCloser{}
		err = nil
	}

	if err != nil {
		return closer, err
	}

	opentracing.SetGlobalTracer(tracer)

	return closer, nil
}

type tracerSampler string

func (ts *tracerSampler) UnmarshalText(bs []byte) error {
	text := tracerSampler(bs)
	switch text {
	case ProbabilisticSampler, ConstantSampler:
		*ts = text
		return nil
	}

	return fmt.Errorf("Invalid tracer type")
}

const (
	ConstantSampler      tracerSampler = "CONSTANT"
	ProbabilisticSampler tracerSampler = "PROBABILISTIC"
)

type tracerConfig struct {
	serviceName  string            // Name of service
	tracer       string            // Tracer service
	hostName     string            // Host name of the tracing agent
	hostPort     string            // Port number of the tracing agent
	sampler      tracerSampler     // Sampler type for tracer
	samplerValue string            // configuration parameter for sampler
	logger       logging.Logger    // Logger to use log tracer errors etc.
	tags         map[string]string // Tags that will be injected to every trace
}

func getTracerConfig() (tracerConfig, error) {
	cfg := tracerConfig{
		hostName:     "localhost",
		sampler:      ConstantSampler,
		samplerValue: "true",
		tags:         map[string]string{environmentKey: "undefined"},
	}

	if service, exists := os.LookupEnv("TRACER_SERVICE"); exists && service != "" {
		cfg.tracer = service
	} else if !exists {
		// No tracer configured
		return cfg, nil
	} else {
		return cfg, fmt.Errorf("tracer type is empty")
	}

	if hostname, exists := os.LookupEnv("TRACER_HOST"); exists && hostname != "" {
		cfg.hostName = hostname
	}

	if sampler, exists := os.LookupEnv("TRACER_SAMPLER"); exists {
		if err := cfg.sampler.UnmarshalText([]byte(sampler)); err != nil {
			return cfg, err
		}
		cfg.samplerValue, _ = os.LookupEnv("TRACER_SAMPLER_VALUE")
	}

	if env, exists := os.LookupEnv("TRACER_ENVIRONMENT"); exists && env != "" {
		cfg.tags[environmentKey] = env
	}

	return cfg, nil
}
