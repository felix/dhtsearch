package main

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func newMeterProvider(ctx context.Context) (func(context.Context) error, error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			attribute.String("service.name", "indexer"),
			//semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		return nil, err
	}
	exporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, err
	}
	// metricExporter, err := stdoutmetric.New()
	// if err != nil {
	// 	panic(err)
	// }

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(
			exporter,
			metric.WithInterval(30*time.Second),
		)),
	)
	otel.SetMeterProvider(mp)
	return mp.Shutdown, nil
}
