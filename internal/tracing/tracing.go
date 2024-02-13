package tracing

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName       = "github.com/superfly/flyctl"
	HeaderFlyTraceId = "fly-trace-id"
	HeaderFlySpanId  = "fly-span-id"
)

func GetTracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

func RecordError(span trace.Span, err error, description string) {
	span.RecordError(err)
	span.SetStatus(codes.Error, description)
}
