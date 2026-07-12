package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestAdminInvitesUserWhoCanCompleteFirstLogin(t *testing.T) {
	env := setupAuthTest(t)
	env.createUserWithRole(t, "admin@upanime.dev", "senha-do-admin", false, true)
	cookie := env.sessionFor(t, "admin@upanime.dev", "senha-do-admin")

	invite := env.post(t, "/api/invites", map[string]string{"email": "convidado@upanime.dev"}, cookie)
	if invite.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", invite.Code, invite.Body.String())
	}

	var summary map[string]any
	json.NewDecoder(invite.Body).Decode(&summary)
	if summary["email"] != "convidado@upanime.dev" || summary["isAdmin"] != false || summary["pending"] != true {
		t.Fatalf("unexpected invite response: %v", summary)
	}

	if len(env.mailer.emails) == 0 || env.mailer.emails[len(env.mailer.emails)-1] != "convidado@upanime.dev" {
		t.Fatalf("expected invite email to convidado, got %v", env.mailer.emails)
	}

	invited, err := env.users.GetByEmail(t.Context(), "convidado@upanime.dev")
	if err != nil {
		t.Fatal(err)
	}
	if invited.IsAdmin {
		t.Fatal("invited user must NOT be admin")
	}
	if !invited.MustChangePassword {
		t.Fatal("invited user must be forced to change password")
	}

	body := env.mailer.bodies[len(env.mailer.bodies)-1]
	match := regexp.MustCompile(`senha temporária é: (\S+)`).FindStringSubmatch(body)
	if match == nil {
		t.Fatalf("temp password not found in invite email body: %q", body)
	}

	login := env.post(t, "/api/auth/login", map[string]string{
		"email": "convidado@upanime.dev", "password": match[1],
	})
	if login.Code != http.StatusOK || decodeStep(t, login) != "change_password" {
		t.Fatalf("expected invited user to reach change_password with emailed temp password, got %d %s", login.Code, login.Body.String())
	}
}

func TestInviteRequiresAdmin(t *testing.T) {
	env := setupAuthTest(t)
	env.createUserWithRole(t, "comum@upanime.dev", "senha-do-comum", false, false)
	cookie := env.sessionFor(t, "comum@upanime.dev", "senha-do-comum")

	invite := env.post(t, "/api/invites", map[string]string{"email": "novo@upanime.dev"}, cookie)
	if invite.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d", invite.Code)
	}

	users := httptest.NewRequest("GET", "/api/users", nil)
	users.AddCookie(cookie)
	response := httptest.NewRecorder()
	env.router.ServeHTTP(response, users)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 listing users as non-admin, got %d", response.Code)
	}
}

func TestInviteRequiresSession(t *testing.T) {
	env := setupAuthTest(t)

	invite := env.post(t, "/api/invites", map[string]string{"email": "novo@upanime.dev"})
	if invite.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", invite.Code)
	}
}

func TestInviteDuplicateEmailConflicts(t *testing.T) {
	env := setupAuthTest(t)
	env.createUserWithRole(t, "admin@upanime.dev", "senha-do-admin", false, true)
	env.createUser(t, "existente@upanime.dev", "qualquer-senha", false)
	cookie := env.sessionFor(t, "admin@upanime.dev", "senha-do-admin")

	invite := env.post(t, "/api/invites", map[string]string{"email": "existente@upanime.dev"}, cookie)
	if invite.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate, got %d", invite.Code)
	}
}

func TestListUsersShowsRolesAndPendingState(t *testing.T) {
	env := setupAuthTest(t)
	env.createUserWithRole(t, "admin@upanime.dev", "senha-do-admin", false, true)
	env.createUser(t, "pendente@upanime.dev", "senha-temporaria", true)
	cookie := env.sessionFor(t, "admin@upanime.dev", "senha-do-admin")

	request := httptest.NewRequest("GET", "/api/users", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	env.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var users []map[string]any
	json.NewDecoder(response.Body).Decode(&users)
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0]["email"] != "admin@upanime.dev" || users[0]["isAdmin"] != true || users[0]["pending"] != false {
		t.Fatalf("unexpected admin summary: %v", users[0])
	}
	if users[1]["email"] != "pendente@upanime.dev" || users[1]["isAdmin"] != false || users[1]["pending"] != true {
		t.Fatalf("unexpected invited summary: %v", users[1])
	}
}

func TestMeExposesAdminFlag(t *testing.T) {
	env := setupAuthTest(t)
	env.createUserWithRole(t, "admin@upanime.dev", "senha-do-admin", false, true)
	cookie := env.sessionFor(t, "admin@upanime.dev", "senha-do-admin")

	request := httptest.NewRequest("GET", "/api/auth/me", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	env.router.ServeHTTP(response, request)

	var me map[string]any
	json.NewDecoder(response.Body).Decode(&me)
	if me["email"] != "admin@upanime.dev" || me["isAdmin"] != true {
		t.Fatalf("unexpected me payload: %v", me)
	}
}
