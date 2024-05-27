package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dgrijalva/jwt-go"
)

type Role struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func methodOverrideMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if override := r.FormValue("_method"); override != "" {
				r.Method = override
			}
		}
		next.ServeHTTP(w, r)
	})
}

func createRoleHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")

	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		log.Println("Failed to open database connection: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	query := "INSERT INTO roles (name) VALUES (?)"
	_, err = db.Exec(query, name)
	if err != nil {
		log.Println("Failed to create role: ", err)
		http.Error(w, "Failed to create role", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func updateRoleHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		log.Println("Invalid role ID: ", err)
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")

	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		log.Println("Failed to open database connection: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	query := "UPDATE roles SET name = ? WHERE id = ?"
	_, err = db.Exec(query, name, id)
	if err != nil {
		log.Println("Failed to update role: ", err)
		http.Error(w, "Failed to update role", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func deleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		log.Println("Invalid role ID: ", err)
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		log.Println("Failed to open database connection: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	query := "DELETE FROM roles WHERE id = ?"
	_, err = db.Exec(query, id)
	if err != nil {
		log.Println("Failed to delete role: ", err)
		http.Error(w, "Failed to delete role", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserIDFromRequest(r)
		if userID == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if !isAdmin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

const AdminRoleID = 1

func isAdmin(r *http.Request) bool {
	userID := getUserIDFromRequest(r)
	if userID == "" {
		log.Error("User ID not found in request")
		return false
	}

	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		log.Error("Failed to open database connection: ", err)
		return false
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role_id = ?", userID, AdminRoleID).Scan(&count)
	if err != nil {
		log.Error("Failed to check user roles: ", err)
		return false
	}

	return count > 0
}

func getUserIDFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("token")
	if err != nil {
		log.Error("Token not found in cookie: ", err)
		return ""
	}
	tokenString := cookie.Value

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		log.Error("Invalid token: ", err)
		return ""
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Error("Failed to extract claims")
		return ""
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		log.Error("User ID not found in claims")
		return ""
	}

	userID := strconv.Itoa(int(userIDFloat))
	return userID
}
