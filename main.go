package main

import (
	"flag"
	"net/http"

	"github.com/godbus/dbus/v5"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var addr = flag.String("listen-address", ":15694", "The address to listen on for HTTP requests.")

var dbusAddr = flag.String("dbus-addr", "", "Address of the DBus daemon to connect to.")

func main() {
	flag.Parse()

	var err error
	var dbusConn *dbus.Conn
	if dbusAddr != nil && *dbusAddr != "" {
		glog.Infof("connecting to dbus daemon at %s", *dbusAddr)
		dbusConn, err = dbus.Connect(*dbusAddr)
	} else {
		glog.Infof("connecting to the system dbus daemon")
		dbusConn, err = dbus.ConnectSystemBus()
	}
	if err != nil {
		glog.Fatal("failed connecting to dbus: ", err)
	}
	defer dbusConn.Close()

	sdConn := NewConnection(dbusConn)
	sdCollector := NewCollector(sdConn)

	// Create non-global registry.
	reg := prometheus.NewRegistry()

	// Add go runtime metrics and process collectors.
	reg.MustRegister(
		collectors.NewBuildInfoCollector(),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		sdCollector,
	)

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))

	glog.Infof("started systemd_networkd_exporter and listening at %s", *addr)
	glog.Fatal(http.ListenAndServe(*addr, nil))
}
