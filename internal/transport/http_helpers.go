package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func ReadJSON(r *http.Request, v any) error {
	if r.ContentLength > 2<<20 {
		return errors.New("body too large")
	}
	d := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 2<<20))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func PathID(path, prefix string) (string, error) {
	id := strings.TrimPrefix(path, prefix)
	if id == "" || strings.Contains(id, "/") {
		return "", errors.New("invalid id")
	}
	return id, nil
}
func HeightParam(path, prefix string) (uint64, error) {
	id, e := PathID(path, prefix)
	if e != nil {
		return 0, e
	}
	return strconv.ParseUint(id, 10, 64)
}
func Timeout(next http.Handler, d time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		_ = ctx
		next.ServeHTTP(w, r.WithContext(context.Background()))
	})
}
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				WriteJSON(w, 500, ErrorBody{Code: "internal", Message: "internal error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
