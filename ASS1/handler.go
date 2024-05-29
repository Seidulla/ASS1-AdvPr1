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
	"github.com/sirupsen/logrus"
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

func mainPageHandler(w http.ResponseWriter, r *http.Request) {
	devices, err := GetDevicesFromDB()
	if err != nil {
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}
	filter := r.URL.Query().Get("filter")
	sort := r.URL.Query().Get("sort")
	page := r.URL.Query().Get("page")

	limit := 10
	offset := 0

	if p, err := strconv.Atoi(page); err == nil && p > 1 {
		offset = (p - 1) * limit
	}

	log.WithFields(logrus.Fields{
		"action": "mainPageHandler",
		"method": r.Method,
		"path":   r.URL.Path,
	}).Info("Handling main page request")

	query := "SELECT id, type1, brand, model FROM electronic"
	if filter != "" {
		query += " WHERE brand LIKE '%" + filter + "%'"
	}
	if sort != "" {
		query += " ORDER BY " + sort
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	devices, err = GetDevicesFromDBWithPagination(query)
	if err != nil {
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("pages/index.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, devices)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
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
		panic(err.Error())
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

	http.Redirect(w, r, "/", http.StatusSeeOther)
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
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		panic(err.Error())
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

func GetDevice(db *sql.DB, id int) (*Device, error) {
	query := "SELECT * FROM electronic WHERE id = ?"
	row := db.QueryRow(query, id)

	device := &Device{}
	err := row.Scan(&device.ID, &device.Type1, &device.Brand, &device.Model)
	if err != nil {
		return nil, err
	}
	return device, nil
}

func updateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	vars := mux.Vars(r)
	idStr := vars["id"]

	deviceID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	var device Device
	err = json.NewDecoder(r.Body).Decode(&device)
	if err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		return
	}

	err = UpdateDevice(db, deviceID, device.Type1, device.Brand, device.Model)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Device updated successfully"})
}

func UpdateDevice(db *sql.DB, id int, type1, brand, model string) error {
	query := "UPDATE electronic SET type1 = ?, brand = ?, model = ? WHERE id = ?"
	_, err := db.Exec(query, type1, brand, model, id)
	if err != nil {
		return err
	}
	return nil
}

func deleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		panic(err.Error())
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
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Device deleted successfully"})
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
