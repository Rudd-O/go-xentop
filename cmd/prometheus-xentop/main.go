package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/Rudd-O/prometheus-xentop/xenstat"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type knownMetric struct {
	Name string
	Type prometheus.ValueType
	Desc *prometheus.Desc
}

func knownMetrics() map[string]knownMetric {
	var m map[string]knownMetric
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
			"counter", "Count of out-of-request situations this domain has encountered", []string{"dom", "vbd"},
		},
		"vbd_read_requests_total": {
			"counter", "Count of read requests this domain has issued", []string{"dom", "vbd"},
		},
		"vbd_write_requests_total": {
			"counter", "Count of write requests this domain has issued", []string{"dom", "vbd"},
		},
		"vbd_read_bytes_total": {
			"counter", "Total bytes this domain has read from virtual block devices", []string{"dom", "vbd"},
		},
		"vbd_written_bytes_total": {
			"counter", "Total bytes this domain has written to from virtual block devices", []string{"dom", "vbd"},
		},
		"nic_count": {
			"gauge", "Count of virtual network devices assigned to this domain", []string{"dom"},
		},
		"net_transmit_bytes_total": {
			"counter", "Total bytes this domain has transmitted through virtual network devices", []string{"dom", "nic"},
		},
		"net_receive_bytes_total": {
			"counter", "Total bytes this domain has received through virtual network devices", []string{"dom", "nic"},
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
		m[metricName] = knownMetric{Name: metricName, Type: typ, Desc: desc}
	}
	return m
}

type XenCollector struct {
	x       *xenstat.XenStats
	metrics map[string]knownMetric
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
		if g.x, err = xenstat.NewXenStats(); err != nil {
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
	m := g.metrics
	for _, domain := range domaindata {
		ch <- f(m["cpu_seconds_total"].Desc, m["cpu_seconds_total"].Type, float64(domain.CPUSeconds), domain.Name)
		ch <- f(m["cpu_count"].Desc, m["cpu_count"].Type, float64(domain.NumVCPUs), domain.Name)
		ch <- f(m["memory_used_bytes"].Desc, m["memory_used_bytes"].Type, float64(domain.MemoryBytes), domain.Name)
		ch <- f(m["memory_maximum_bytes"].Desc, m["memory_maximum_bytes"].Type, float64(domain.MaxmemBytes), domain.Name)
		ch <- f(m["vbd_count"].Desc, m["vbd_count"].Type, float64(domain.NumVBDs), domain.Name)
		ch <- f(m["nic_count"].Desc, m["nic_count"].Type, float64(domain.NumNICs), domain.Name)
		for n, v := range domain.VBDs {
			ch <- f(m["vbd_out_of_requests_errors_total"].Desc, m["vbd_out_of_requests_errors_total"].Type, float64(v.OutOfRequests), domain.Name, fmt.Sprintf("%d", n))
			ch <- f(m["vbd_read_requests_total"].Desc, m["vbd_read_requests_total"].Type, float64(v.ReadRequests), domain.Name, fmt.Sprintf("%d", n))
			ch <- f(m["vbd_write_requests_total"].Desc, m["vbd_write_requests_total"].Type, float64(v.WriteRequests), domain.Name, fmt.Sprintf("%d", n))
			ch <- f(m["vbd_read_bytes_total"].Desc, m["vbd_read_bytes_total"].Type, float64(v.BytesRead), domain.Name, fmt.Sprintf("%d", n))
			ch <- f(m["vbd_written_bytes_total"].Desc, m["vbd_written_bytes_total"].Type, float64(v.BytesWritten), domain.Name, fmt.Sprintf("%d", n))
		}
		for n, v := range domain.NICs {
			ch <- f(m["net_transmit_bytes_total"].Desc, m["net_transmit_bytes_total"].Type, float64(v.BytesTransmitted), domain.Name, fmt.Sprintf("%d", n))
			ch <- f(m["net_receive_bytes_total"].Desc, m["net_receive_bytes_total"].Type, float64(v.BytesReceived), domain.Name, fmt.Sprintf("%d", n))
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
