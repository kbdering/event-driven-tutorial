package tracing

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"github.com/kbdering/waiterservice/pkg/message"
	
)

var tp *sdktrace.TracerProvider
var t trace.Tracer
var propagator propagation.TextMapPropagator

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	return otlptracegrpc.New(ctx)
}

func newTraceProvider(exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("WaiterService"),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

func InitOpenTelemetry() {
	fmt.Printf("initializing OpenTelemetry")
	ctx := context.Background()

	exp, err := newExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp = newTraceProvider(exp)

	otel.SetTracerProvider(tp)

	// Finally, set the tracer that can be used for this package.
	t = tp.Tracer("WaiterService")
	propagator = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})

}

func closeOpenTelemetry() {
	_ = tp.Shutdown(context.Background())

}

func GetTracer() trace.Tracer {
	return t
}

func GetContextFromHeaders(m *message.Message) context.Context {
	return propagator.Extract(context.Background(), m.Headers)
}

func InjectContextToHeaders(m *message.Message, ctx context.Context) {
	propagator.Inject(ctx, m.Headers)
}
