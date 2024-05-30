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
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Device struct {
	ID    int    `json:"id"`
	Type1 string `json:"type1"`
	Brand string `json:"brand"`
	Model string `json:"model"`
}

func GetDevicesFromDBWithPagination(query string) ([]Device, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var device Device
		if err := rows.Scan(&device.ID, &device.Type1, &device.Brand, &device.Model); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func GetDevicesFromDB() ([]Device, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, type1, brand, model FROM electronic")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var device Device
		if err := rows.Scan(&device.ID, &device.Type1, &device.Brand, &device.Model); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func createDeviceHandler(w http.ResponseWriter, r *http.Request) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	type1 := r.FormValue("type1")
	brand := r.FormValue("brand")
	model := r.FormValue("model")

	err = CreateDevice(db, type1, brand, model)
	if err != nil {
		http.Error(w, "Failed to create device", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func CreateDevice(db *sql.DB, type1, brand, model string) error {
	query := "INSERT INTO electronic (type1, brand, model) VALUES (?, ?, ?)"
	_, err := db.Exec(query, type1, brand, model)
	return err
}

func getDeviceHandler(w http.ResponseWriter, r *http.Request) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vars := mux.Vars(r)
	idStr := vars["id"]
	deviceID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	device, err := GetDevice(db, deviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("edit_device.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, device)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
func GetDevice(db *sql.DB, id int) (*Device, error) {
	query := "SELECT id, type1, brand, model FROM electronic WHERE id = ?"
	row := db.QueryRow(query, id)

	device := &Device{}
	err := row.Scan(&device.ID, &device.Type1, &device.Brand, &device.Model)
	return device, err
}

func updateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vars := mux.Vars(r)
	idStr := vars["id"]
	deviceID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	type1 := r.FormValue("type1")
	brand := r.FormValue("brand")
	model := r.FormValue("model")

	err = UpdateDevice(db, deviceID, type1, brand, model)
	if err != nil {
		http.Error(w, "Failed to update device", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func UpdateDevice(db *sql.DB, id int, type1, brand, model string) error {
	query := "UPDATE electronic SET type1 = ?, brand = ?, model = ? WHERE id = ?"
	_, err := db.Exec(query, type1, brand, model, id)
	return err
}

func deleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vars := mux.Vars(r)
	idStr := vars["id"]
	deviceID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	err = DeleteDevice(db, deviceID)
	if err != nil {
		http.Error(w, "Failed to delete device", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func DeleteDevice(db *sql.DB, id int) error {
	log.Printf("Deleting device with ID %d\n", id) // Log the ID being deleted
	query := "DELETE FROM electronic WHERE id = ?"
	result, err := db.Exec(query, id)
	if err != nil {
		log.Printf("Error deleting device: %v\n", err) // Log the deletion error
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Rows affected after deletion: %d\n", rowsAffected) // Log the number of rows affected

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
		errorMessage := Response{
			Status:  "400",
			Message: "Invalid JSON message",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorMessage)
		return
	}

	fmt.Println("Recieved message: ", requestBody.Message)

	response := Response{
		Status:  "success",
		Message: "data successfully received",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
