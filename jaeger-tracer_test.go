package rest

import (
	"reflect"
	"testing"

	"github.com/uber/jaeger-client-go"
)

func TestGetJaegerSampler(t *testing.T) {
	zeroProb, _ := jaeger.NewProbabilisticSampler(0.0)
	halfProb, _ := jaeger.NewProbabilisticSampler(0.5)

	tests := []struct {
		name    string
		sampler tracerSampler
		val     string
		want    jaeger.Sampler
	}{
		{"invalid sampler", tracerSampler(""), "", jaeger.NewConstSampler(true)},
		{"constant sampler, with default", ConstantSampler, "", jaeger.NewConstSampler(false)},
		{"constant sampler, with invalid", ConstantSampler, "invalid", jaeger.NewConstSampler(false)},
		{"constant sampler, with false", ConstantSampler, "false", jaeger.NewConstSampler(false)},
		{"constant sampler, with true", ConstantSampler, "true", jaeger.NewConstSampler(true)},
		{"probabilistic sampler, with default", ProbabilisticSampler, "", zeroProb},
		{"probabilistic sampler, with invalid value", ProbabilisticSampler, "invalid", zeroProb},
		{"probabilistic sampler, with 0.0 value", ProbabilisticSampler, "0.0", zeroProb},
		{"probabilistic sampler, with 0.5 value", ProbabilisticSampler, "0.5", halfProb},
		{"probabilistic sampler, with -0.5 value", ProbabilisticSampler, "-0.5", zeroProb},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getJaegerSampler(tt.sampler, tt.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getJaegerSampler() = %v, want %v", got, tt.want)
			}
		})
	}
}
