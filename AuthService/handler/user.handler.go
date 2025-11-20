package handler

import (
	"auth-service/repository"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type UserDto struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UserHandler struct {
	UserRepository *repository.UserRepository
}

func (h UserHandler) RetrieveSearchedUsers(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Only GET request allowed", http.StatusBadRequest)
		return
	}

	params := r.URL.Query()
	q := params.Get("q")
	if q == "" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		users, err := h.UserRepository.FindAll(ctx)
		if err != nil {
			http.Error(w, "Error retrieving users", http.StatusInternalServerError)
			return
		}

		userDtos := []UserDto{}
		for _, user := range users {
			userDto := UserDto{ID: user.ID.Hex(), Username: user.Username, Email: user.Email}
			userDtos = append(userDtos, userDto)
		}

		json.NewEncoder(w).Encode(userDtos)
	}
}
