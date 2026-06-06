package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	ConnectionsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "im_connections_total",
		Help: "Current number of WebSocket connections",
	})

	MessagesSentTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "im_messages_sent_total",
		Help: "Total number of messages sent",
	})

	MessagesPushTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "im_messages_push_total",
		Help: "Total number of message pushes",
	})
)

func init() {
	prometheus.MustRegister(ConnectionsTotal)
	prometheus.MustRegister(MessagesSentTotal)
	prometheus.MustRegister(MessagesPushTotal)
}
