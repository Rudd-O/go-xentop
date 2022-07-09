package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type knownMetric struct {
	Name string
	Type prometheus.ValueType
	Desc *prometheus.Desc
}

func knownMetrics() []knownMetric {
	var m []knownMetric
	for metricName, metric := range map[string]struct {
		Type        string
		Description string
		Labels      []string
	}{
		"cpu_seconds_total": {
			"counter", "Total number of seconds spent across all CPUs executing in this domain", []string{"dom"},
		},
		"cpu_count": {
			"gauge", "Count of virtual CPUs assigned to this domain", []string{"dom"},
		},
		"memory_used_bytes": {
			"gauge", "Memory used by this domain", []string{"dom"},
		},
		"memory_maximum_bytes": {
			"gauge", "Maximum memory this domain is allowed to allocate, assuming availability", []string{"dom"},
		},
		"vbd_count": {
			"gauge", "Count of virtual block devices assigned to this domain", []string{"dom"},
		},
		"vbd_out_of_requests_errors_total": {
			"counter", "Count of out-of-request situations this domain has encountered", []string{"dom"},
		},
		"vbd_read_requests_total": {
			"counter", "Count of read requests this domain has issued", []string{"dom"},
		},
		"vbd_write_requests_total": {
			"counter", "Count of write requests this domain has issued", []string{"dom"},
		},
		"vbd_read_bytes_total": {
			"counter", "Total bytes this domain has read from virtual block devices", []string{"dom"},
		},
		"vbd_written_bytes_total": {
			"counter", "Total bytes this domain has written to from virtual block devices", []string{"dom"},
		},
		"net_transmit_bytes_total": {
			"counter", "Total bytes this domain has transmitted through virtual network devices", []string{"dom"},
		},
		"net_receive_bytes_total": {
			"counter", "Total bytes this domain has received through virtual network devices", []string{"dom"},
		},
	} {
		fullName := prometheus.BuildFQName("xen", "", metricName)
		var typ prometheus.ValueType
		switch metric.Type {
		case "gauge":
			typ = prometheus.GaugeValue
		case "counter":
			typ = prometheus.CounterValue
		}
		desc := prometheus.NewDesc(
			fullName,
			metric.Description,
			metric.Labels, nil,
		)
		m = append(m, knownMetric{Name: metricName, Type: typ, Desc: desc})
	}
	return m
}

type XenCollector struct {
	x       *XenStats
	metrics []knownMetric
}

func NewXenCollector() *XenCollector {
	return &XenCollector{nil, knownMetrics()}
}

func (g *XenCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range g.metrics {
		ch <- metric.Desc
	}
}

func (g *XenCollector) Collect(ch chan<- prometheus.Metric) {
	var err error
	if g.x == nil {
		if g.x, err = NewXenStats(); err != nil {
			log.Printf("Error collecting metrics: %s", err)
			return
		}
	}

	domaindata, err := g.x.Poll()
	if err != nil {
		g.x.Close()
		g.x = nil
		log.Printf("Error collecting metrics: %s", err)
		return
	}

	f := prometheus.MustNewConstMetric

	var val float64
	for _, domain := range domaindata {
		for _, metric := range g.metrics {
			switch metric.Name {
			case "cpu_seconds_total":
				val = float64(domain.CPU)
			case "cpu_count":
				val = float64(domain.NumVCPUs)
			case "memory_used_bytes":
				val = float64(domain.Memory)
			case "memory_maximum_bytes":
				val = float64(domain.Maxmem)
			case "vbd_count":
				val = float64(domain.VBDCount)
			case "vbd_out_of_requests_errors_total":
				val = float64(domain.VBD_OutOfRequests)
			case "vbd_read_requests_total":
				val = float64(domain.VBD_ReadRequests)
			case "vbd_write_requests_total":
				val = float64(domain.VBD_ReadRequests)
			case "vbd_read_bytes_total":
				val = float64(domain.VBD_BytesRead)
			case "vbd_written_bytes_total":
				val = float64(domain.VBD_BytesWritten)
			case "net_transmit_bytes_total":
				val = float64(domain.NIC_BytesTransmitted)
			case "net_receive_bytes_total":
				val = float64(domain.NIC_BytesReceived)
			default:
				log.Printf("unknown metric %s", metric.Name)
			}
			ch <- f(metric.Desc, metric.Type, val, domain.Name)
		}
	}
}

func main() {
	addr := flag.String("bind", ":8080", "The address to bind to")
	flag.Parse()

	prometheus.Register(NewXenCollector())

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		h := promhttp.HandlerFor(prometheus.Gatherers{
			prometheus.DefaultGatherer,
		}, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})
	log.Printf("Starting server on address %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
