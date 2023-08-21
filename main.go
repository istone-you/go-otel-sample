package main

import (
        "context"
        "fmt"
        "log"
        "os"
        "os/signal"
        "time"

        "go.opentelemetry.io/otel"
        "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
        "go.opentelemetry.io/otel/propagation"
        "go.opentelemetry.io/otel/sdk/resource"
        sdktrace "go.opentelemetry.io/otel/sdk/trace"
        semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func initProvider() (func(context.Context) error, error) {
        ctx := context.Background()

        res, err := resource.New(ctx,
                resource.WithAttributes(
                        semconv.ServiceName("SampleService"),
                ),
        )
        if err != nil {
                return nil, fmt.Errorf("failed to create resource: %w", err)
        }

        traceExporter, err := otlptracehttp.New(ctx,
                otlptracehttp.WithEndpoint("Endpoint"),
                otlptracehttp.WithInsecure(),
        )
        if err != nil {
                log.Printf("failed to create HTTP trace exporter: %s", err)
        }

        bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
        tracerProvider := sdktrace.NewTracerProvider(
                sdktrace.WithSampler(sdktrace.AlwaysSample()),
                sdktrace.WithResource(res),
                sdktrace.WithSpanProcessor(bsp),
        )
        otel.SetTracerProvider(tracerProvider)

        otel.SetTextMapPropagator(propagation.TraceContext{})

        return tracerProvider.Shutdown, nil
}

func main() {
        log.Printf("Waiting for connection...")

        ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
        defer cancel()

        shutdown, err := initProvider()
        if err != nil {
                log.Fatal(err)
        }
        defer func() {
                if err := shutdown(ctx); err != nil {
                        log.Fatal("failed to shutdown TracerProvider: %w", err)
                }
        }()

        tracer := otel.Tracer("SampleTracer")

        ctx, span := tracer.Start(
                ctx,
                "SampleCollectorExporter",)
        defer span.End()

        for i := 0; i < 10; i++ {
                _, iSpan := tracer.Start(ctx, fmt.Sprintf("Sample-%d", i))
                log.Printf("Doing really hard work (%d / 10)\n", i+1)

                <-time.After(time.Second)
                iSpan.End()
        }

        log.Printf("Done!")
}