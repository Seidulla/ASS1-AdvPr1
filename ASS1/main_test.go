package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

func TestCreateDevice(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(localhost:3306)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = CreateDevice(db, "phone", "Apple", "iPhone 13")
	if err != nil {
		t.Errorf("Error creating device: %s", err)
	}
}

func TestGetDeviceHandler(t *testing.T) {
	dsn := fmt.Sprintf("%s:%s@tcp(localhost:3306)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	req, err := http.NewRequest("GET", "/devices/21", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/devices/{id}", getDeviceHandler).Methods("GET")
	router.ServeHTTP(rr, req)

	expected := Device{
		ID:    21,
		Type1: "phone",
		Brand: "Apple",
		Model: "iPhone 13",
	}

	var device Device
	err = json.NewDecoder(rr.Body).Decode(&device)

	if device != expected {
		t.Errorf("Incorrect device. Expected: %+v, Got: %+v", expected, device)
	}
}
