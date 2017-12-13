package main

import (
	"net/http"
	"strconv"
	"github.com/ExpressenAB/bigip_exporter/config"
	"github.com/juju/loggo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/ExpressenAB/bigip_exporter/collector"
	"strings"
	"github.com/pr8kerl/f5er/f5"
)

var (
	logger = loggo.GetLogger("")
	configuration = config.GetConfig()
	collectors = createBigIPCollectors()
)

func createBigIPCollectors() map[string]*collector.BigipCollector{
	configuration := config.GetConfig()
	list := make(map[string]*collector.BigipCollector)
	for host, _ := range configuration.Lookup {
		bigipEndpoint := configuration.Lookup[host].Host + ":" + strconv.Itoa(configuration.Lookup[host].Port)
		var exporterPartitionsList []string
		if configuration.Exporter.Partitions != "" {
			exporterPartitionsList = strings.Split(configuration.Exporter.Partitions, ",")
		} else {
			exporterPartitionsList = nil
		}
		authMethod := f5.TOKEN
		if configuration.Lookup[host].BasicAuth {
			authMethod = f5.BASIC_AUTH
		}
		bigip := f5.New(bigipEndpoint,configuration.Lookup[host].Username,configuration.Lookup[host].Password,authMethod)
		list[host], _ = collector.NewBigipCollector(bigip, configuration.Exporter.Namespace, exporterPartitionsList)
	}
	return list
}

func listen(exporterBindAddress string, exporterBindPort int) {
	//http.Handle("/metrics", getTarget(prometheus.Handler()))
	http.HandleFunc("/metrics", getTarget)
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

//Wrapper around the handler.
//Get the key from the handler
//Return/Call the handler that is passed in as Parameter

func getTarget(w http.ResponseWriter, r *http.Request) {
	//return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Query().Get("target")) == 0 {
			logger.Errorf("Missing target")
			http.Error(w, "Missing target", http.StatusUnprocessableEntity)
			return // don't call original handler
		}else {
			target := r.URL.Query().Get("target")
			if val, ok := collectors[target]; ok {
				err := prometheus.Register(val)
				if err != nil {
					logger.Errorf("Error when registering collector for host: [%v]. Error: [%v]", target, err)
					logger.Errorf("Trying to unregister current collector")
					unregister := prometheus.Unregister(val)
					if !unregister {
						logger.Errorf("Failed to unregister for host [%v]", target)
					}
				}
				prometheus.Handler().ServeHTTP(w, r)
				prometheus.Unregister(val)
			} else {
				//Target not found
				logger.Errorf("Exporter does not have the configuration for target [%v]", r.URL.Query().Get("target"))
				http.Error(w, "Target not supported", http.StatusUnprocessableEntity)
			}
		}
	//})
}

func main() {
	logger.Debugf("Config: [%v]", configuration)
	listen(configuration.Exporter.BindAddress, configuration.Exporter.BindPort)
}
