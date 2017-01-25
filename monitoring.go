package main

import "github.com/prometheus/client_golang/prometheus"

var (
	// Create a summary to track fictional interservice RPC latencies for three
	// distinct services with different latency distributions. These services are
	// differentiated via a "service" label.
	taskCounts = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "taskqueue_tasks",
			Help: "Tasks executed.",
		},
	)
)

/*

CounterOpts{
			Namespace:   opts.Namespace,
			Subsystem:   opts.Subsystem,
			Name:        "requests_total",
			Help:        "Total number of HTTP requests made.",
			ConstLabels: opts.ConstLabels,
		},
		instLabels,

*/
