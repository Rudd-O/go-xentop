go-xentop
=========

Wraps [xentop](https://wiki.xenproject.org/wiki/Xentop(1)).  Documentation is
on [godoc](https://godoc.org/github.com/Rudd-O/go-xentop).

Forked from bwesterb.

Xen prometheus exporter
-----------------------

As an example, the `prometheus-xentop` folder contains a webserver that exposes
the xentop data to [Prometheus](https://prometheus.io).

    $ go get github.com/Rudd-O/go-xentop/prometheus-xentop
    $ $GOROOT/bin/prometheus-xentop -bind 0:8080
