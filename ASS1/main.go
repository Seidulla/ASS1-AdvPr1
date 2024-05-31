package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"gopkg.in/gomail.v2"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var limiter = rate.NewLimiter(1, 10)
var log = logrus.New()

var cartStorage = struct {
	sync.RWMutex
	carts map[string][]Device
}{carts: make(map[string][]Device)}

func main() {
	log.SetFormatter(&logrus.JSONFormatter{})
	logFile, err := os.OpenFile("application.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Error("Failed to open log file: ", err)
		return
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	r := mux.NewRouter()
	r.Use(methodOverrideMiddleware)
	r.HandleFunc("/", limitHandler(mainPageHandler)).Methods("GET")
	r.HandleFunc("/json", limitHandler(handleJSONRequest)).Methods("POST")
	r.HandleFunc("/buy", buyHandler).Methods("POST")
	r.HandleFunc("/buy1", buyHandler1).Methods("POST")
	r.HandleFunc("/payment", paymentHandler).Methods("GET")
	r.HandleFunc("/process-payment", processPaymentHandler).Methods("POST")
	r.HandleFunc("/payment-success", paymentSuccessHandler).Methods("GET")
	r.HandleFunc("/register", registerHandler).Methods("GET", "POST")
	r.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	r.HandleFunc("/confirm", confirmHandler).Methods("GET")
	r.HandleFunc("/user", authMiddleware(userProfileHandler)).Methods("GET")
	r.HandleFunc("/admin", authMiddleware(adminMiddleware(adminProfileHandler))).Methods("GET")
	r.HandleFunc("/change-password", authMiddleware(changePasswordHandler)).Methods("POST")
	r.HandleFunc("/change-email", authMiddleware(changeEmailHandler)).Methods("POST")
	r.HandleFunc("/logout", logoutHandler).Methods("GET")

	// Admin routes for device management
	r.HandleFunc("/device", authMiddleware(adminMiddleware(createDeviceHandler))).Methods("POST")
	r.HandleFunc("/device/{id}", authMiddleware(adminMiddleware(getDeviceHandler))).Methods("GET")
	r.HandleFunc("/device/{id}", authMiddleware(adminMiddleware(updateDeviceHandler))).Methods("POST", "PUT")
	r.HandleFunc("/device/{id}", authMiddleware(adminMiddleware(deleteDeviceHandler))).Methods("POST", "DELETE")

	r.HandleFunc("/admin/roles", authMiddleware(adminMiddleware(createRoleHandler))).Methods("POST")
	r.HandleFunc("/admin/roles/update", authMiddleware(adminMiddleware(updateRoleHandler))).Methods("POST")
	r.HandleFunc("/admin/roles/delete", authMiddleware(adminMiddleware(deleteRoleHandler))).Methods("POST")
	r.HandleFunc("/admin/send-email", authMiddleware(adminMiddleware(sendEmailHandler))).Methods("POST")

	log.Info("Server listening on port 8080")
	http.ListenAndServe(":8080", r)
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

	// Check if user is logged in
	isLoggedIn := false
	if cookie, err := r.Cookie("token"); err == nil && cookie.Value != "" {
		isLoggedIn = true
	}

	data := struct {
		Devices    []Device
		IsLoggedIn bool
	}{
		Devices:    devices,
		IsLoggedIn: isLoggedIn,
	}

	tmpl, err := template.ParseFiles("pages/index.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
func limitHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}
}
func buyHandler1(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromRequest(r)
	if userID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	deviceIDStr := r.FormValue("device_id")
	deviceID, err := strconv.Atoi(deviceIDStr)
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	device, err := GetDeviceByID(deviceID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	cartStorage.Lock()
	cartStorage.carts[userID] = append(cartStorage.carts[userID], device)
	cartStorage.Unlock()

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "token",
		Value:  "",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func insertTransaction(db *sql.DB, customerID, status string) error {
	query := "INSERT INTO transactions1 (customer_id, status) VALUES (?, ?)"
	_, err := db.Exec(query, customerID, status)
	return err
}

func buyHandler(w http.ResponseWriter, r *http.Request) {
	customerID := getUserIDFromRequest(r)
	if customerID == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	err = insertTransaction(db, customerID, "pending")
	if err != nil {
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/payment", http.StatusSeeOther)
}

func GetDeviceByID(id int) (Device, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return Device{}, err
	}
	defer db.Close()

	var device Device
	query := "SELECT id, type1, brand, model FROM electronic WHERE id = ?"
	err = db.QueryRow(query, id).Scan(&device.ID, &device.Type1, &device.Brand, &device.Model)
	if err != nil {
		return Device{}, err
	}

	return device, nil
}

type ReceiptData struct {
	CompanyName       string
	TransactionNumber string
	DateTime          string
	CustomerName      string
	PaymentMethod     string
	Items             []Item
	GrandTotal        string
}

type Item struct {
	Name      string
	UnitPrice string
	Quantity  int
	Total     string
}

func generateReceiptPDF(data ReceiptData) ([]byte, error) {
	tmpl, err := template.ParseFiles("pages/receipt_template.html")
	if err != nil {
		return nil, err
	}

	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, data); err != nil {
		return nil, err
	}

	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	pdfg.AddPage(wkhtmltopdf.NewPageReader(bytes.NewReader(tpl.Bytes())))
	pdfg.MarginTop.Set(10)
	pdfg.MarginRight.Set(10)
	pdfg.MarginBottom.Set(10)
	pdfg.MarginLeft.Set(10)
	pdfg.Dpi.Set(300)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.Grayscale.Set(false)
	pdfg.NoCollate.Set(false)

	err = pdfg.Create()
	if err != nil {
		return nil, err
	}

	return pdfg.Bytes(), nil
}

func processPaymentHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	cardNumber := r.FormValue("cardNumber")
	expirationDate := r.FormValue("expirationDate")
	cvv := r.FormValue("cvv")
	name := r.FormValue("name")
	address := r.FormValue("address")
	log.Println(cardNumber, expirationDate, cvv, name, address)

	// Simulate payment processing
	paymentSuccessful := true // For testing, always assume the payment is successful

	if paymentSuccessful {
		// Simulate transaction data
		receiptData := ReceiptData{
			CompanyName:       "Your Company",
			TransactionNumber: "1234567890",
			DateTime:          time.Now().Format("2006-01-02 15:04:05"),
			CustomerName:      name,
			PaymentMethod:     "Credit Card",
			Items: []Item{
				{Name: "Item 1", UnitPrice: "$10.00", Quantity: 1, Total: "$10.00"},
				{Name: "Item 2", UnitPrice: "$20.00", Quantity: 2, Total: "$40.00"},
			},
			GrandTotal: "$50.00",
		}

		pdfBytes, err := generateReceiptPDF(receiptData)
		if err != nil {
			log.Printf("Failed to generate PDF: %v", err)
			http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
			return
		}

		// Assuming the client email is available

		clientEmail := "craldiyar@gmail.com"
		err = sendEmailWithAttachment(clientEmail, pdfBytes)
		if err != nil {
			log.Printf("Failed to send email: %v", err)
			http.Error(w, "Failed to send email", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/payment-success", http.StatusSeeOther)
	} else {
		http.Error(w, "Payment failed", http.StatusPaymentRequired)
	}
}

func getEmailByID(db *sql.DB, userID int) (string, error) {
	var email string
	query := "SELECT email FROM users WHERE id = ?"
	err := db.QueryRow(query, userID).Scan(&email)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no user found with id %d", userID)
		}
		return "", err
	}
	return email, nil
}

func sendEmailWithAttachment(to string, pdfBytes []byte) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "sagidolla04@internet.ru")
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Your Receipt")
	m.SetBody("text/plain", "Thank you for your purchase. Please find your receipt attached.")

	m.Attach("receipt.pdf", gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(pdfBytes)
		return err
	}))

	d := gomail.NewDialer("smtp.mail.ru", 587, "sagidolla04@internet.ru", "HBthmWAUUQ4JS7AvsK5v")

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func updateTransactionStatus(db *sql.DB, transactionID int, status string) error {
	query := "UPDATE transactions1 SET status = ? WHERE customer_id = ?"
	_, err := db.Exec(query, status, transactionID)
	if err != nil {
		log.Printf("Error executing query: %v", err)
	}
	return err
}

func getTransactionIDFromSession(r *http.Request) int {
	// For demonstration, returning a static transaction ID
	// In real implementation, retrieve it from session or request context
	return 21
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("pages/payment.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

func paymentSuccessHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("pages/payment_success.html")
	if err != nil {
		log.Printf("Failed to load template: %v", err)
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Printf("Failed to render template: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
