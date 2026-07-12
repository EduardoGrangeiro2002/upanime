package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"upanime/api/auth"
	"upanime/api/model"
	"upanime/api/store"
)

type InviteHandler struct {
	users  store.UserStore
	mailer auth.Mailer
}

func NewInviteHandler(users store.UserStore, mailer auth.Mailer) *InviteHandler {
	return &InviteHandler{users: users, mailer: mailer}
}

type inviteRequest struct {
	Email string `json:"email"`
}

type userSummary struct {
	Email   string `json:"email"`
	IsAdmin bool   `json:"isAdmin"`
	Pending bool   `json:"pending"`
}

func (h *InviteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !strings.Contains(email, "@") {
		writeJSONError(w, "email inválido", http.StatusBadRequest)
		return
	}

	if _, err := h.users.GetByEmail(r.Context(), email); err == nil {
		writeJSONError(w, "usuário já existe", http.StatusConflict)
		return
	}

	password, err := auth.GenerateTempPassword()
	if err != nil {
		writeJSONError(w, "erro interno", http.StatusInternalServerError)
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		writeJSONError(w, "erro interno", http.StatusInternalServerError)
		return
	}

	user := &model.User{Email: email, PasswordHash: hash, MustChangePassword: true, IsAdmin: false}
	if err := h.users.Create(r.Context(), user); err != nil {
		writeJSONError(w, "falha ao criar usuário", http.StatusInternalServerError)
		return
	}

	body := fmt.Sprintf(
		"Você foi convidado para o UpAnime por %s.\n\nSua senha temporária é: %s\n\nNo primeiro login será exigida a troca da senha.",
		UserEmail(r.Context()), password,
	)
	if err := h.mailer.Send(email, "UpAnime — você foi convidado", body); err != nil {
		writeJSONError(w, "usuário criado, mas o email de convite falhou — verifique o SMTP", http.StatusBadGateway)
		return
	}

	writeJSON(w, userSummary{Email: email, IsAdmin: false, Pending: true})
}

func (h *InviteHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		writeJSONError(w, "falha ao listar usuários", http.StatusInternalServerError)
		return
	}

	summaries := make([]userSummary, 0, len(users))
	for _, user := range users {
		summaries = append(summaries, userSummary{
			Email:   user.Email,
			IsAdmin: user.IsAdmin,
			Pending: user.MustChangePassword,
		})
	}
	writeJSON(w, summaries)
}
