package handler

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"upanime/api/auth"
	"upanime/api/store"
)

const sessionCookieName = "upanime_session"

type AuthHandler struct {
	service      *auth.Service
	users        store.UserStore
	secureCookie bool
}

func NewAuthHandler(service *auth.Service, users store.UserStore, secureCookie bool) *AuthHandler {
	return &AuthHandler{service: service, users: users, secureCookie: secureCookie}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	Email           string `json:"email"`
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type mfaRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type forgotRequest struct {
	Email string `json:"email"`
}

type resetRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.service.Login(r.Context(), req.Email, req.Password, clientIP(r))
	h.respondAuthResult(w, result, err)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.service.ChangePassword(r.Context(), req.Email, req.CurrentPassword, req.NewPassword, clientIP(r))
	h.respondAuthResult(w, result, err)
}

func (h *AuthHandler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	var req mfaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.service.VerifyMFA(r.Context(), req.Email, req.Code, clientIP(r))
	h.respondAuthResult(w, result, err)
}

func (h *AuthHandler) Forgot(w http.ResponseWriter, r *http.Request) {
	var req forgotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.ForgotPassword(r.Context(), req.Email); err != nil {
		http.Error(w, `{"error":"falha ao enviar o código"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Reset(w http.ResponseWriter, r *http.Request) {
	var req resetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	err := h.service.ResetPassword(r.Context(), req.Email, req.Code, req.NewPassword)
	if err != nil {
		h.respondAuthError(w, err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.secureCookie,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	email := UserEmail(r.Context())
	isAdmin := false
	if user, err := h.users.GetByEmail(r.Context(), email); err == nil {
		isAdmin = user.IsAdmin
	}
	writeJSON(w, map[string]any{"email": email, "isAdmin": isAdmin})
}

func (h *AuthHandler) respondAuthResult(w http.ResponseWriter, result auth.Result, err error) {
	if err != nil {
		h.respondAuthError(w, err)
		return
	}

	if result.Step == auth.StepOK {
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    result.Token,
			Path:     "/",
			MaxAge:   30 * 24 * 60 * 60,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   h.secureCookie,
		})
	}
	writeJSON(w, map[string]string{"step": string(result.Step)})
}

func (h *AuthHandler) respondAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrInvalidCode) {
		writeJSONError(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if errors.Is(err, auth.ErrWeakPassword) {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSONError(w, "erro interno", http.StatusInternalServerError)
}

func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
