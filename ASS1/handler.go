package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

type RequestBody struct {
	Message string `json:"message"`
}

type Response struct {
	Status string `json:"status"`

	Message string `json:"message"`
}

func mainPageHandler(w http.ResponseWriter, r *http.Request) {
	var fileName = "index.html"
	t, err := template.ParseFiles(fileName)
	if err != nil {
		fmt.Println("error parsing file", err)
		return
	}
	err = t.ExecuteTemplate(w, fileName, nil)
	if err != nil {
		fmt.Println("error executing template", err)
		return
	}
}

func handleJSONRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var requestBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid JSON Format", http.StatusBadRequest)
		return
	}
	if requestBody.Message == "" {
		http.Error(w, "Invalid JSON message", http.StatusBadRequest)
		return
	}

	fmt.Println("Recieved message: ", requestBody.Message)

	response := Response{
		Status:  "success",
		Message: "data successfully received  ",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

//if requestBody.Message == "" {
//    errorMessage := Response{
//        Status:  "400",
//        Message: "Invalid JSON message",
//    }
//    w.Header().Set("Content-Type", "application/json")
//    w.WriteHeader(http.StatusBadRequest)
//    json.NewEncoder(w).Encode(errorMessage)
//    return
//}
