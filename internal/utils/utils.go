package utils

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(v); err != nil {
		return
	}
}

func WriteJSONError(w http.ResponseWriter, status int, code string, message string) {
	WriteJSON(w, status, map[string]string{
		"errorCode": code,
		"message":   message,
	})
}
