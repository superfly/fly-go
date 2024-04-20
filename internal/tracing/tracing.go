package tracing

import (
	"net/http"

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

func SpanContextFromHeaders(res *http.Response) trace.SpanContext {
	traceIDstr := res.Header.Get(HeaderFlyTraceId)
	spanIDstr := res.Header.Get(HeaderFlySpanId)

	traceID, _ := trace.TraceIDFromHex(traceIDstr)
	spanID, _ := trace.SpanIDFromHex(spanIDstr)

	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})
}
