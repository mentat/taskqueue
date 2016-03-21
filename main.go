package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type AsyncTask struct {
	URL        string `json:"url"`
	ETA        int    `json:"eta"`
	Countdown  int    `json:"countdown"`
	MaxRetries int    `json:"max_retries"`
	Payload    string `json:"payload"`
	Expires    int    `json:"expires"`
	Queue      string `json:"queue"`
}

var RabbitServer *AMQP

func main() {

	fileName := os.Getenv("TASKQUEUE_CONFIG_FILE")
	if fileName == "" {
		fileName = "taskqueue.ini"
	}

	config, err := ParseConfigFile(fileName)

	if err != nil {
		fmt.Println("The configuration file could not be parsed:", err)
		os.Exit(1)
	}

	RabbitServer := NewAMQP(config.AmqpServer)
	err = RabbitServer.Connect()
	if err != nil {
		fmt.Println("Could not connect with AMQP server:", err)
		os.Exit(1)
	}

	errChan := make(chan error)

	// Spawn goroutines to handle each queue's operation.
	for i := range config.Queues {

		delivery, err := RabbitServer.ConsumeQueue(config.Queues[i].Name)
		if err != nil {
			fmt.Println("Could not read queue:", config.Queues[i].Name, err)
			os.Exit(1)
		}

		go readQueue(&config.Queues[i], delivery, errChan)
	}

	r := mux.NewRouter()
	r.HandleFunc("/tasks/push/", CreatePushTask)
	r.HandleFunc("/tasks/push/{id}", GetPushTask).Methods("GET")
	r.HandleFunc("/tasks/push/{id}", ModifyPushTask).Methods("PUT")
	r.HandleFunc("/tasks/push/{id}", DeletePushTask).Methods("DELETE")

	http.ListenAndServe(":12345", r)
}

func decodeForm(req *http.Request) (*AsyncTask, error) {
	return nil, nil
}

func decodeJSON(req *http.Request) (*AsyncTask, error) {
	return nil, nil
}

func CreatePushTask(http.ResponseWriter, *http.Request) {
	/*
	   Create a new Push Task.
	*/
}

func ModifyPushTask(http.ResponseWriter, *http.Request) {
	/* Modify a Push task. */
}

func DeletePushTask(http.ResponseWriter, *http.Request) {
	/* Delete a Push task. */
}

func GetPushTask(http.ResponseWriter, *http.Request) {
	/* Get a Push task. */
}
