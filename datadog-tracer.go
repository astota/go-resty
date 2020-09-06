package rest

import (
	"fmt"
	"io"
	"math"
	"strconv"

	ddopentracer "github.com/foodiefm/opentracing/contrib/gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"github.com/opentracing/opentracing-go"
	dtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type datadogTracer struct {
	tracer opentracing.Tracer
}

func (t datadogTracer) Close() error {
	dtracer.Stop()
	return nil
}

func (t *datadogTracer) Init(cfg tracerConfig) (opentracing.Tracer, io.Closer, error) {
	host := cfg.hostName
	if cfg.hostPort != "" {
		host = fmt.Sprintf("%s:%s", host, cfg.hostPort)
	}

	sampler := t.getSampler(cfg.sampler, cfg.samplerValue)

	t.tracer = ddopentracer.New(cfg.serviceName,
		dtracer.WithAgentAddr(host),
		dtracer.WithServiceName(cfg.serviceName),
		dtracer.WithSampler(sampler),
		dtracer.WithGlobalTag(environmentKey, cfg.tags[environmentKey]),
	)

	return t.tracer, t, nil
}

func (t datadogTracer) getSampler(s tracerSampler, val string) dtracer.Sampler {
	switch s {
	case ConstantSampler:
		if val == "true" {
			return dtracer.NewRateSampler(1.0)
		}
		return dtracer.NewRateSampler(0.0)
	case ProbabilisticSampler:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			f = 0.0
		}
		f = math.Max(math.Min(f, 1.0), 0.0)
		return dtracer.NewRateSampler(f)
	}

	return dtracer.NewRateSampler(0.0)
}

func initDatadogTracer(cfg tracerConfig) (opentracing.Tracer, io.Closer, error) {
	tracer := datadogTracer{}
	return tracer.Init(cfg)
}
