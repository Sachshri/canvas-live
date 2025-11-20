package handler

import (
	"auth-service/model"
	"auth-service/repository"
	"auth-service/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// User Registration
type AuthHandler struct {
	UserRepository *repository.UserRepository
}

// ================================================= New User Registration Handler ===========================================================================

func (h AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	requestMethod := r.Method

	fmt.Fprintf(w, "Request Method: %s\n", requestMethod)

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	var newUser model.User
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, "Invalid JSON data: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Set up context
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Create user in db
	createdUser, err := h.UserRepository.CreateUser(ctx, newUser)
	if err != nil {
		http.Error(w, "Error creating user "+err.Error(), http.StatusInternalServerError)
	}

	// Send success response
	// w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "User ID: %s", createdUser.ID)
}

// ================================================= Login Handler ===========================================================================

type LoginData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (h AuthHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	loginData := LoginData{}
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		http.Error(w, "Invalid json data format", http.StatusBadRequest)
		return
	}

	// 2. Set up context
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 3. Call the repository method
	user, err := h.UserRepository.FindUserByEmail(ctx, loginData.Email)
	if err != nil {
		// Handle the internal database error
		http.Error(w, "Internal server error during database lookup", http.StatusInternalServerError)
		return
	}

	// 4. Handle result
	if user == nil {
		http.NotFound(w, r)
		fmt.Fprintf(w, "User with email '%s' not found.", loginData.Email)
		return
	}

	if user.Password != loginData.Password {
		http.Error(w, "Incorrect credentials", http.StatusUnauthorized)
	}

	// 5. Generate JWT
	jwtToken, err := utils.CreateToken(user.ID.Hex(), loginData.Email, user.Username)
	if err != nil {
		http.Error(w, "Error signing you in - Try again.", http.StatusInternalServerError)
	}

	response := TokenResponse{AccessToken: jwtToken}
	json.NewEncoder(w).Encode(response)
}

// ================================================= Authenticate Request Handler ===========================================================================

func (h AuthHandler) AuthenticateRequest(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Invalid authorization format: expected 'Bearer <token>'", http.StatusBadRequest)
		return
	}

	token := authHeader[len("Bearer "):]

	// extract claims from token
	claims, err := utils.ParseToken(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// add UserID to request object
	// --- RESPONSE HEADER MODIFICATION (CRITICAL STEP) ---

	// 1. Set the custom headers on the ResponseWriter (w)
	// These are the headers Nginx's auth_request_set will read.
	w.Header().Set("X-User-ID", claims.UserID)
	w.Header().Set("X-Username", claims.Username)
	// w.Header().Set("X-User-Email", claims.UserEmail) // If you use the email header

	// 2. IMPORTANT: Send a 2xx Status Code (usually 200 OK)
	// Nginx requires a 2xx response from the auth_request to proceed with proxy_pass.
	// We send a minimal response and a 200 status code.
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Auth Success") // This response body is usually ignored by Nginx.
	// fmt.Fprintf(w, "Access granted: %s", claims)

}
