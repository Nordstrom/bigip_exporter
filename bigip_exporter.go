package main

import (
	"net/http"
	"strconv"
	"github.com/juju/loggo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus"
	"fmt"
	"github.com/ExpressenAB/bigip_exporter/config"
)

var (
	logger = loggo.GetLogger("")
	configuration = config.GetConfig()
)


func listen(exporterBindAddress string, exporterBindPort int) {
	http.HandleFunc("/metrics", handler)
	http.HandleFunc("/health-check", HealthCheckHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>BIG-IP Exporter</title></head>
			<body>
			<h1>BIG-IP Exporter</h1>
			<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>`))
	})
	exporterBind := exporterBindAddress + ":" + strconv.Itoa(exporterBindPort)
	logger.Criticalf("Process failed: %s", http.ListenAndServe(exporterBind, nil))
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	logger.Tracef("health-check called")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"alive": true}`))
}

func handler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		return
	}
	moduleName := r.URL.Query().Get("module")
	if moduleName == "" {
		logger.Debugf("No module entered. Defaulting to [test_env]")
		moduleName = "test_env"
	}
	collector, ok := configuration.CreateBigipCollector(target,moduleName)
	if !ok {
		http.Error(w, fmt.Sprintf("[%s] module not found in the config.", moduleName), 400)
		return
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w,r)
}

func main() {
	logger.Debugf("Config: [%v]", configuration.String())
	listen(configuration.Exporter.BindAddress, configuration.Exporter.BindPort)
}
