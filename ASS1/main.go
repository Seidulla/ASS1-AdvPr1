package main

import (
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

func main() {
	err := checkDBConnection()
	if err != nil {
		fmt.Println("Failed to connect to the database:", err)
		return
	}

	r := mux.NewRouter()
	r.HandleFunc("/", mainPageHandler)
	r.HandleFunc("/json", handleJSONRequest)
	r.HandleFunc("/device", createDeviceHandler).Methods("POST")
	r.HandleFunc("/device/{id}", getDeviceHandler).Methods("GET")
	r.HandleFunc("/device/{id}", updateDeviceHandler).Methods("PUT")
	r.HandleFunc("/device/{id}", deleteDeviceHandler).Methods("DELETE")

	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", r)
}
