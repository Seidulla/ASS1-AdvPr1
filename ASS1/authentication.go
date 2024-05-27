package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type AdminPageData struct {
	Roles []Role
}

var jwtKey = []byte("my_secret_key")

type Claims struct {
	Username string `json:"username"`
	UserID   int    `json:"user_id"`
	jwt.RegisteredClaims
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "pages/login.html")
		return
	} else if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
		db, err := sql.Open(dbDriver, dsn)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		var user User
		err = db.QueryRow("SELECT id, username, email, password, token, confirmed FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Token, &user.Confirmed)
		if err != nil {
			log.Printf("Failed to retrieve user information: %v\n", err)
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		if !user.Confirmed {
			log.Println("User is not confirmed")
			http.Error(w, "Please confirm your email first", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			log.Println("Invalid password")
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		var role string
		log.Printf("Retrieving role for user ID: %d\n", user.ID)
		err = db.QueryRow("SELECT r.name FROM roles r JOIN user_roles ur ON r.id = ur.role_id WHERE ur.user_id = ?", user.ID).Scan(&role)
		if err != nil {
			log.Printf("Failed to retrieve user role: %v\n", err)
			http.Error(w, "Failed to retrieve user role", http.StatusInternalServerError)
			return
		}

		log.Printf("User role: %s\n", role)

		expirationTime := time.Now().Add(5 * time.Minute)
		claims := &Claims{
			Username: username,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
			},
			UserID: user.ID,
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(jwtKey)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:    "token",
			Value:   tokenString,
			Expires: expirationTime,
		})

		if role == "admin" {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/user", http.StatusSeeOther)
		}
	}
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			if err == http.ErrNoCookie {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		tokenStr := cookie.Value
		claims := &Claims{}

		tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !tkn.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func userProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "pages/profile.html")
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func adminProfileHandler(w http.ResponseWriter, r *http.Request) {
	dsn := fmt.Sprintf("%s:%s@tcp(sql12.freesqldatabase.com)/%s", dbUser, dbPass, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		log.Println("Failed to open database connection: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name FROM roles")
	if err != nil {
		log.Println("Failed to fetch roles: ", err)
		http.Error(w, "Failed to fetch roles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			log.Println("Failed to scan role: ", err)
			http.Error(w, "Failed to fetch roles", http.StatusInternalServerError)
			return
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		log.Println("Rows error: ", err)
		http.Error(w, "Failed to fetch roles", http.StatusInternalServerError)
		return
	}

	data := AdminPageData{Roles: roles}
	tmpl, err := template.ParseFiles("pages/admin.html")
	if err != nil {
		log.Println("Failed to parse template: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Println("Failed to execute template: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
