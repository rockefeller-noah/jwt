package demo

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/rockefeller-noah/jwt"
)

// Response is the JSON envelope shared by every example endpoint.
type Response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// NewHandler exposes a demonstration login and JWT-protected endpoints.
// The fixed demo credentials are not an application user database.
func NewHandler(auth *jwt.JWT, stateful bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, "ok", nil)
	})
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		var credentials struct {
			Account  string `json:"account"`
			Password string `json:"password"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&credentials); err != nil {
			writeJSON(w, http.StatusBadRequest, "invalid login JSON", nil)
			return
		}
		if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, "expected one JSON object", nil)
			return
		}
		if credentials.Account != "demo" || credentials.Password != "demo-password" {
			writeJSON(w, http.StatusUnauthorized, "invalid credentials", nil)
			return
		}
		token, err := auth.GenerateToken("1", jwt.Payload{"account": "demo"})
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, "token store unavailable", nil)
			return
		}
		writeJSON(w, http.StatusOK, "login success", token)
	})
	mux.HandleFunc("GET /profile", protect(auth, func(w http.ResponseWriter, r *http.Request) {
		payload, err := auth.Http().ParseToken(r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, "profile success", payload)
	}))
	mux.HandleFunc("PUT /refresh", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Http().RefreshToken(r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, "refresh success", token)
	})
	mux.HandleFunc("DELETE /logout", protect(auth, func(w http.ResponseWriter, r *http.Request) {
		if !stateful {
			writeJSON(w, http.StatusNotImplemented, "logout requires a token store", nil)
			return
		}
		if err := auth.Http().DestroyToken(r); err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, "logout success", nil)
	}))
	return mux
}

func protect(auth *jwt.JWT, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := auth.Http().Middleware(r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		next(w, request)
	}
}

func writeAuthError(w http.ResponseWriter, err error) {
	if jwt.IsMissingToken(err) || jwt.IsInvalidToken(err) || jwt.IsExpiredToken(err) ||
		jwt.IsAuthElsewhere(err) || jwt.IsIdentityMissing(err) || errors.Is(err, jwt.ErrSignAlgorithmNotMatch) {
		writeJSON(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, "token store unavailable", nil)
}

func writeJSON(w http.ResponseWriter, status int, message string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{Status: status, Message: message, Data: data})
}
