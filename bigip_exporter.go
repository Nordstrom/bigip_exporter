package main

import (
	"net/http"
	"strconv"
	"github.com/juju/loggo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus"
	"./config"
	"fmt"
)

var (
	logger = loggo.GetLogger("")
	configuration = config.GetConfig()
	//collectors = createBigIPCollectors()
)

//func createBigIPCollectors() map[string]*collector.BigipCollector{
//	list := make(map[string]*collector.BigipCollector)
//	for host, _ := range configuration.Lookup {
//		bigipEndpoint := configuration.Lookup[host].Host + ":" + strconv.Itoa(configuration.Lookup[host].Port)
//		var exporterPartitionsList []string
//		if configuration.Exporter.Partitions != "" {
//			exporterPartitionsList = strings.Split(configuration.Exporter.Partitions, ",")
//		} else {
//			exporterPartitionsList = nil
//		}
//		authMethod := f5.TOKEN
//		if configuration.Lookup[host].BasicAuth {
//			authMethod = f5.BASIC_AUTH
//		}
//		bigip := f5.New(bigipEndpoint,configuration.Lookup[host].Username,configuration.Lookup[host].Password,authMethod)
//		list[host], _ = collector.NewBigipCollector(bigip, configuration.Exporter.Namespace, exporterPartitionsList)
//	}
//	debugStatement := ""
//	for key, value := range list {
//		debugStatement += fmt.Sprintf("Key [%s], Value [%s]", key, value)
//	}
//	logger.Debugf("List of collectors: [%v]", debugStatement)
//	return list
//}

func listen(exporterBindAddress string, exporterBindPort int) {
	http.HandleFunc("/metrics", handler)
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
	logger.Debugf("Config: [%v]", configuration)
	listen(configuration.Exporter.BindAddress, configuration.Exporter.BindPort)
}
