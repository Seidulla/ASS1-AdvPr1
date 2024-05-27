package main

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/go-gomail/gomail"
	"golang.org/x/crypto/bcrypt"
)

func changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		currentPassword := r.FormValue("current-password")
		newPassword := r.FormValue("new-password")

		userID := getUserIDFromRequest(r)
		if userID == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if !validateCurrentPassword(userID, currentPassword) {
			http.Error(w, "Invalid current password", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		if err := updatePasswordInDatabase(userID, hashedPassword); err != nil {
			http.Error(w, "Failed to update password", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/user", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func validateCurrentPassword(userID string, currentPassword string) bool {

	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		log.Error("Failed to open database connection: ", err)
		return false
	}
	defer db.Close()

	var hashedPassword string
	err = db.QueryRow("SELECT password FROM users WHERE id = ?", userID).Scan(&hashedPassword)
	if err != nil {
		log.Error("Failed to retrieve hashed password from database: ", err)
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(currentPassword))
	if err != nil {
		log.Println("Invalid password")
		return false
	}

	return true
}

func updatePasswordInDatabase(userID string, hashedPassword []byte) error {

	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		log.Error("Failed to open database connection: ", err)
		return err
	}
	defer db.Close()

	query := "UPDATE users SET password = ? WHERE id = ?"
	_, err = db.Exec(query, hashedPassword, userID)
	if err != nil {
		log.Error("Failed to update password in database: ", err)
		return err
	}

	return nil
}

func changeEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		newEmail := r.FormValue("new-email")

		userID := getUserIDFromRequest(r)
		if userID == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if err := updateEmailInDatabase(userID, newEmail); err != nil {
			http.Error(w, "Failed to update email", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/user", http.StatusSeeOther)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func updateEmailInDatabase(userID string, newEmail string) error {

	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		log.Error("Failed to open database connection: ", err)
		return err
	}
	defer db.Close()

	query := "UPDATE users SET email = ? WHERE id = ?"
	_, err = db.Exec(query, newEmail, userID)
	if err != nil {
		log.Error("Failed to update email in database: ", err)
		return err
	}

	return nil
}

func sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	discount := r.FormValue("discount")

	userEmails, err := getAllUserEmails()
	if err != nil {
		log.Println("Failed to fetch user emails: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	for _, email := range userEmails {
		err := sendEmail(email, discount)
		if err != nil {
			log.Println("Failed to send email to ", email, ": ", err)
		}
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func sendEmail(email, discount string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "sagidolla04@internet.ru") //
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Special Discount Offer")
	m.SetBody("text/html", fmt.Sprintf("Dear user,<br><br>We are pleased to offer you a special discount: %s.<br><br>Best regards,<br>Device Shop", discount))

	d := gomail.NewDialer("smtp.mail.ru", 587, "esagidolla04@internet.ru", "HBthmWAUUQ4JS7AvsK5v")
	d.TLSConfig = &tls.Config{ServerName: "smtp.mail.ru"}

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
func getAllUserEmails() ([]string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT email FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userEmails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		userEmails = append(userEmails, email)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userEmails, nil
}
