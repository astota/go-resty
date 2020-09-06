package rest

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"time"

	"github.com/astota/go-logging"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics"
)

func getJaegerSampler(s tracerSampler, val string) jaeger.Sampler {
	switch s {
	case ConstantSampler:
		v := val == "true"
		return jaeger.NewConstSampler(v)
	case ProbabilisticSampler:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			f = 0.0
		}
		f = math.Max(math.Min(f, 1.0), 0.0)
		sampler, _ := jaeger.NewProbabilisticSampler(f)
		return sampler
	}

	return jaeger.NewConstSampler(true)
}

func initJaegerTracer(cfg tracerConfig) (opentracing.Tracer, io.Closer, error) {
	if cfg.hostPort == "" {
		cfg.hostPort = "6831"
	}

	// Create new remote reporter
	sender, err := jaeger.NewUDPTransport(fmt.Sprintf("%s:%s", cfg.hostName, cfg.hostPort), 1<<16)
	if err != nil {
		cfg.logger.Errorf("Error when initializing opentracing reporter: %s", err.Error())
		return defaultNoopTracer, noopCloser{}, err
	}
	reporter := jaeger.NewRemoteReporter(sender)

	// Configure tracer
	config := jaegercfg.Configuration{
		ServiceName: cfg.serviceName,
		Disabled:    false,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			BufferFlushInterval: 5 * time.Second,
		},
	}

	// Create new logger, with tracer field
	logger := cfg.logger.AddFields(logging.Fields{
		"tracer": "jaeger",
	})

	sampler := getJaegerSampler(cfg.sampler, cfg.samplerValue)

	// Create and return tracer
	return config.NewTracer(
		jaegercfg.Logger(logger),
		jaegercfg.Metrics(metrics.NullFactory),
		jaegercfg.Reporter(reporter),
		jaegercfg.Sampler(sampler),
		jaegercfg.Tag(environmentKey, cfg.tags[environmentKey]),
	)
}
