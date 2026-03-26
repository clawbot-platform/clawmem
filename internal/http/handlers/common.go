package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	code := "bad_request"
	switch status {
	case http.StatusNotFound:
		code = "not_found"
	case http.StatusInternalServerError:
		code = "internal_error"
	case http.StatusServiceUnavailable:
		code = "not_ready"
	}
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"status":  status,
		},
	})
}

func decodeJSON(r *http.Request, dst any) error {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType == "" {
		return errors.New("Content-Type must be application/json")
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "application/json") {
		return fmt.Errorf("Content-Type must be application/json")
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func parsePagination(r *http.Request) (limit int, offset int, err error) {
	limit, err = parseOptionalInt(r.URL.Query().Get("limit"), 25)
	if err != nil {
		return 0, 0, errors.New("limit must be a valid integer")
	}
	offset, err = parseOptionalInt(r.URL.Query().Get("offset"), 0)
	if err != nil {
		return 0, 0, errors.New("offset must be a valid integer")
	}
	if limit <= 0 {
		return 0, 0, errors.New("limit must be greater than 0")
	}
	if offset < 0 {
		return 0, 0, errors.New("offset must be zero or greater")
	}
	return limit, offset, nil
}

func parseOptionalInt(raw string, fallback int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, err
	}
	return value, nil
}
