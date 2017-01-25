package main

import "net/http"

func decodeForm(req *http.Request) (*AsyncTask, error) {
	return nil, nil
}

func decodeJSON(req *http.Request) (*AsyncTask, error) {
	return nil, nil
}

// CreatePushTask -
func CreatePushTask(http.ResponseWriter, *http.Request) {
	/*
	   Create a new Push Task.
	   Not needed at the moment for the requirements, but added as a
	   placeholder.
	*/
}

// ModifyPushTask -
func ModifyPushTask(http.ResponseWriter, *http.Request) {
	/*
		Modify a Push task.
		Not needed at the moment for the requirements, but added as a
		placeholder
	*/
}

// DeletePushTask -
func DeletePushTask(http.ResponseWriter, *http.Request) {
	/*
		Delete a Push task.
		Not needed at the moment for the requirements, but added as a
		placeholder
	*/
}

// GetPushTask -
func GetPushTask(http.ResponseWriter, *http.Request) {
	/*
		Get a Push task.
		Not needed at the moment for the requirements, but added as a
		placeholder
	*/
}
