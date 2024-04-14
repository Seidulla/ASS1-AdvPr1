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
	Type1 string `json:"type1"`
	Brand string `json:"brand"`
	Model string `json:"model"`
}

const (
	dbDriver = "mysql"
	dbUser   = "root"
	dbPass   = "aldi6on9"
	dbName   = "electronics"
)

func mainPageHandler(w http.ResponseWriter, r *http.Request) {
	// Запрос данных из базы данных
	devices, err := GetDevicesFromDB() // Предполагается, что у вас есть функция для получения всех устройств из базы данных
	if err != nil {
		// Обработка ошибки
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}
	// Получаем параметр фильтра из URL
	filter := r.URL.Query().Get("filter")
	sort := r.URL.Query().Get("sort")

	// SQL-запрос с учетом фильтра
	query := "SELECT id, type1, brand, model FROM electronic"
	if filter != "" {
		query += " WHERE brand LIKE '%" + filter + "%'"
	}
	if sort != "" {
		query += " ORDER BY " + sort
	}

	// Получаем устройства из базы данных с учетом фильтра
	devices, err = GetDevicesFromDBWithFilter(query)
	if err != nil {
		// Обработка ошибки
		http.Error(w, "Failed to fetch devices", http.StatusInternalServerError)
		return
	}

	// Загрузка HTML-шаблона
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		// Обработка ошибки
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	// Отображение HTML-страницы с данными
	err = tmpl.Execute(w, devices)
	if err != nil {
		// Обработка ошибки
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}

}

func GetDevicesFromDBWithFilter(query string) ([]Device, error) {
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Выполнение запроса к базе данных с учетом фильтра
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Перебор результатов запроса и создание списка устройств
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
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Выполнение запроса к базе данных
	rows, err := db.Query("SELECT id, type1, brand, model FROM electronic")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Перебор результатов запроса и создание списка устройств
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
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	// Получение данных из формы
	type1 := r.FormValue("type1")
	brand := r.FormValue("brand")
	model := r.FormValue("model")

	// Добавление нового устройства в базу данных
	err = CreateDevice(db, type1, brand, model)
	if err != nil {
		// Обработка ошибки
		http.Error(w, "Failed to create device", http.StatusInternalServerError)
		return
	}

	// Перенаправление на главную страницу или любую другую страницу
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
	err := row.Scan(&device.ID, &device.Type1, &device.Brand, &device.Model)
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
	UpdateDevice(db, deviceID, device.Type1, device.Brand, device.Model)
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
