package rest

import (
	"github.com/opentracing/opentracing-go"
)

// NO-OP Closer implementation to allow closer that does not do anything.
type noopCloser struct{}

func (d noopCloser) Close() error {
	return nil
}

// NO-OP tracer openTracer to cases, where actual tracer is not defined.
var defaultNoopTracer opentracing.Tracer = opentracing.NoopTracer{}
