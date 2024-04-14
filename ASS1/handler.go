package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

type RequestBody struct {
	Message string `json:"message"`
}

type Response struct {
	Status string `json:"status"`

	Message string `json:"message"`
}

type Device struct {
	ID    int    `json:"id"`
	type1 string `json:"type1"`
	brand string `json:"brand"`
	model string `json:"model"`
}

const (
	dbDriver = "mysql"
	dbUser   = "root"
	dbPass   = ""
	dbName   = "electronics"
)

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

func createDeviceHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	var device Device
	json.NewDecoder(r.Body).Decode(&device)

	CreateDevice(db, device.type1, device.brand, device.model)
	if err != nil {
		http.Error(w, "Failed to create", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "Device created successfully")
}

func CreateDevice(db *sql.DB, type1, brand, model string) error {
	query := "INSERT INTO electronic (type1, brand, model) VALUES (?, ?, ?)"
	_, err := db.Exec(query, type1, brand, model)
	if err != nil {
		return err
	}
	return nil
}

func getDeviceHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// Get the 'id' parameter from the URL
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Convert 'id' to an integer
	deviceID, err := strconv.Atoi(idStr)

	// Call the GetUser function to fetch the user data from the database
	user, err := GetDevice(db, deviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// Convert the user object to JSON and send it in the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func GetDevice(db *sql.DB, id int) (*Device, error) {
	query := "SELECT * FROM electronic WHERE id = ?"
	row := db.QueryRow(query, id)

	device := &Device{}
	err := row.Scan(&device.ID, &device.type1, &device.brand, &device.model)
	if err != nil {
		return nil, err
	}
	return device, nil
}

func updateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// Get the 'id' parameter from the URL
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Convert 'id' to an integer
	deviceID, err := strconv.Atoi(idStr)

	var device Device
	err = json.NewDecoder(r.Body).Decode(&device)

	// Call the GetUser function to fetch the user data from the database
	UpdateDevice(db, deviceID, device.type1, device.brand, device.model)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	fmt.Fprintln(w, "Device updated successfully")
}

func UpdateDevice(db *sql.DB, id int, type1, brand, model string) error {
	query := "UPDATE electronic SET type = ?, brand = ?, model = ? WHERE id = ?"
	_, err := db.Exec(query, type1, brand, model, id)
	if err != nil {
		return err
	}
	return nil
}

func deleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	// Get the 'id' parameter from the URL
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Convert 'id' to an integer
	deviceID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid 'id' parameter", http.StatusBadRequest)
		return
	}

	user := DeleteDevice(db, deviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	fmt.Fprintln(w, "Device deleted successfully")

	// Convert the user object to JSON and send it in the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
func DeleteDevice(db *sql.DB, id int) error {
	query := "DELETE FROM electronic WHERE id = ?"
	_, err := db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
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

func checkDBConnection() error {
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	// Попробуйте выполнить простой запрос, например, выборку одной строки
	var result string
	err = db.QueryRow("SELECT 'Connected to database'").Scan(&result)
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
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
