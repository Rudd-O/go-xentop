# Prometheus Xen top / statistics

Wraps [xentop](https://wiki.xenproject.org/wiki/Xentop(1)).  Documentation is
on [godoc](https://godoc.org/github.com/Rudd-O/prometheus-xentop/xenstat).

Forked from bwesterb then fully rewritten to use the C API of xenstats instead
of following the command-line utility line by line — for higher precision
collection of data.

## In the box: also a Xen statistics prometheus exporter

The `cmd/prometheus-xentop` folder contains a webserver that exposes
the Xen statistics data to [Prometheus](https://prometheus.io).

```
cd this/folder
make bin/prometheus-xentop
```

You can also build an RPM (just make sure to build it in a machine that
has the same Xen `libxenstat` and `glibc` libraries as the one you will be
running this program on):

```
make rpm
```

For users of Qubes OS, a great way of getting a compatible build environment
is to use https://github.com/Rudd-O/qubes-dom0-container-images .
