package dht

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter = otel.Meter("userspace.com.au/dhtsearch/dht")

var (
	netOctets  metric.Int64Counter
	netPackets metric.Int64Counter
)

var ktableCalls metric.Int64Counter

func init() {
	var err error
	netOctets, err = meter.Int64Counter(
		"network.octets",
		metric.WithDescription("The number of bytes."),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		panic(err)
	}
	netPackets, err = meter.Int64Counter(
		"bytes.received",
		metric.WithDescription("The number of packets."),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		panic(err)
	}
	ktableCalls, err = meter.Int64Counter(
		"ktable.calls",
		metric.WithDescription("Number of calls to ktable."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		panic(err)
	}
}

func configureMetrics(c *Client) error {
	var err error
	start := time.Now()
	if _, err = meter.Float64ObservableCounter(
		"uptime",
		metric.WithDescription("The running duration."),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(float64(time.Since(start).Seconds()))
			return nil
		}),
	); err != nil {
		panic(err)
	}

	_, err = meter.Int64ObservableUpDownCounter(
		"ktable.size",
		metric.WithDescription("The number of nodes in the ktable."),
		metric.WithUnit("{nodes}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(c.table.Count()))
			return nil
		}),
	)
	if err != nil {
		return err
	}
	return nil
}
