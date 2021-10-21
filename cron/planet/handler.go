package handler

import (
	"encoding/json"
	"net/http"
)

type Ok struct {
	Message string `json:"message" bson:"message"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	response := Ok{Message: "Ok"}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
