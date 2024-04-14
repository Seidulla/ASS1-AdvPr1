package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", mainPageHandler)
	http.HandleFunc("/json", handleJSONRequest)
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
