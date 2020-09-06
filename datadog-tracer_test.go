package rest

import (
	"reflect"
	"testing"

	dtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func TestGetSampler(t *testing.T) {
	tests := []struct {
		name    string
		sampler tracerSampler
		val     string
		want    dtracer.Sampler
	}{
		{"invalid sampler", tracerSampler(""), "", dtracer.NewRateSampler(0.0)},
		{"constant sampler, with default", ConstantSampler, "", dtracer.NewRateSampler(0.0)},
		{"constant sampler, with invalid", ConstantSampler, "invalid", dtracer.NewRateSampler(0.0)},
		{"constant sampler, with false", ConstantSampler, "false", dtracer.NewRateSampler(0.0)},
		{"constant sampler, with true", ConstantSampler, "true", dtracer.NewRateSampler(1.0)},
		{"probabilistic sampler, with default", ProbabilisticSampler, "", dtracer.NewRateSampler(0.0)},
		{"probabilistic sampler, with invalid value", ProbabilisticSampler, "invalid", dtracer.NewRateSampler(0.0)},
		{"probabilistic sampler, with 0.0 value", ProbabilisticSampler, "0.0", dtracer.NewRateSampler(0.0)},
		{"probabilistic sampler, with 0.5 value", ProbabilisticSampler, "0.5", dtracer.NewRateSampler(0.5)},
		{"probabilistic sampler, with -0.5 value", ProbabilisticSampler, "-0.5", dtracer.NewRateSampler(0.0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dd := datadogTracer{}
			if got := dd.getSampler(tt.sampler, tt.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getJaegerSampler() = %v, want %v", got, tt.want)
			}
		})
	}
}
