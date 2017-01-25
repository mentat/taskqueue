package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/juju/loggo"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type AsyncTask struct {
	URL        string `json:"url"`
	ETA        int64  `json:"eta"`
	Countdown  int    `json:"countdown"`
	MaxRetries int64  `json:"max_retries"`
	Payload    string `json:"payload"`
	Expires    int64  `json:"expires"`
	Queue      string `json:"queue"`
}

// BackendServer -
var BackendServer Backend

var logger = loggo.GetLogger("taskqueue")

func main() {
	logger.SetLogLevel(loggo.TRACE)
	logger.Infof("Starting up...")

	fileName := os.Getenv("TASKQUEUE_CONFIG_FILE")
	if fileName == "" {
		fileName = "/etc/taskqueue/taskqueue.ini"
	}

	flag.StringVar(&fileName, "config", fileName, "Configuration file.")
	flag.Parse()

	config, err := ParseConfigFile(fileName)

	if err != nil {
		logger.Errorf("The configuration file could not be parsed: %s", err)
		os.Exit(1)
	}

	BackendServer, err := GetBackend(config.ServerConnectString)

	err = BackendServer.Connect()
	if err != nil {
		logger.Errorf("Could not connect with backend server: %s", err)
		os.Exit(1)
	}

	errChan := make(chan error)

	// Spawn goroutines to handle each queue's operation.
	for i := range config.Queues {

		channel, err := BackendServer.GetChannel()
		if err != nil {
			logger.Errorf("Could not get channel: %s", err)
			os.Exit(1)
		}

		go readQueue(&config.Queues[i], channel, errChan)
	}

	r := mux.NewRouter()
	r.HandleFunc("/tasks/push/", CreatePushTask)
	r.HandleFunc("/tasks/push/{id}", GetPushTask).Methods("GET")
	r.HandleFunc("/tasks/push/{id}", ModifyPushTask).Methods("PUT")
	r.HandleFunc("/tasks/push/{id}", DeletePushTask).Methods("DELETE")
	r.Handle("/metrics", promhttp.Handler())

	http.ListenAndServe(":12345", r)
}
