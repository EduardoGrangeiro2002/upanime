package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"upanime/api/store"
)

const (
	mfaValidity    = 30 * 24 * time.Hour
	sessionTTL     = 30 * 24 * time.Hour
	minPasswordLen = 8
)

var (
	ErrInvalidCredentials = errors.New("email ou senha inválidos")
	ErrInvalidCode        = errors.New("código inválido ou expirado")
	ErrWeakPassword       = errors.New("a senha deve ter pelo menos 8 caracteres")
)

type Step string

const (
	StepChangePassword Step = "change_password"
	StepMFA            Step = "mfa"
	StepOK             Step = "ok"
)

type Result struct {
	Step  Step
	Token string
}

type Service struct {
	users  store.UserStore
	codes  *CodeStore
	mailer Mailer
	geo    GeoLookup
	signer *TokenSigner
	now    func() time.Time
}

func NewService(users store.UserStore, codes *CodeStore, mailer Mailer, geo GeoLookup, signer *TokenSigner, now func() time.Time) *Service {
	return &Service{users: users, codes: codes, mailer: mailer, geo: geo, signer: signer, now: now}
}

func (s *Service) Login(ctx context.Context, email, password, ip string) (Result, error) {
	user, err := s.verifyPassword(ctx, email, password)
	if err != nil {
		return Result{}, err
	}
	if user.MustChangePassword {
		return Result{Step: StepChangePassword}, nil
	}
	return s.gateMFA(ctx, email, ip, user.LastIP, user.LastLocation, user.LastMFAAt)
}

func (s *Service) ChangePassword(ctx context.Context, email, currentPassword, newPassword, ip string) (Result, error) {
	user, err := s.verifyPassword(ctx, email, currentPassword)
	if err != nil {
		return Result{}, err
	}
	if len(newPassword) < minPasswordLen {
		return Result{}, ErrWeakPassword
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return Result{}, err
	}
	if err := s.users.UpdatePassword(ctx, email, hash, false); err != nil {
		return Result{}, err
	}
	return s.gateMFA(ctx, email, ip, user.LastIP, user.LastLocation, user.LastMFAAt)
}

func (s *Service) VerifyMFA(ctx context.Context, email, code, ip string) (Result, error) {
	valid, err := s.codes.Verify(ctx, PurposeMFA, email, code)
	if err != nil {
		return Result{}, err
	}
	if !valid {
		return Result{}, ErrInvalidCode
	}

	if _, err := s.users.GetByEmail(ctx, email); err != nil {
		return Result{}, ErrInvalidCode
	}

	location := s.geo.Lookup(ctx, ip)
	if err := s.users.UpdateMFAContext(ctx, email, ip, location, s.now()); err != nil {
		return Result{}, err
	}
	return s.issueSession(email), nil
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	if _, err := s.users.GetByEmail(ctx, email); err != nil {
		return nil
	}

	code, err := s.codes.Issue(ctx, PurposeReset, email)
	if errors.Is(err, ErrCooldown) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.mailer.Send(
		email,
		"UpAnime — redefinição de senha",
		fmt.Sprintf("Seu código para redefinir a senha é: %s\n\nEle expira em 15 minutos. Se você não pediu isso, ignore este email.", code),
	)
}

func (s *Service) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	if len(newPassword) < minPasswordLen {
		return ErrWeakPassword
	}

	valid, err := s.codes.Verify(ctx, PurposeReset, email, code)
	if err != nil {
		return err
	}
	if !valid {
		return ErrInvalidCode
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.users.UpdatePassword(ctx, email, hash, false)
}

func (s *Service) VerifySession(token string) (string, bool) {
	return s.signer.Verify(token, s.now())
}

func (s *Service) verifyPassword(ctx context.Context, email, password string) (userRecord, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return userRecord{}, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return userRecord{}, ErrInvalidCredentials
	}
	return userRecord{
		MustChangePassword: user.MustChangePassword,
		LastIP:             user.LastIP,
		LastLocation:       user.LastLocation,
		LastMFAAt:          user.LastMFAAt,
	}, nil
}

type userRecord struct {
	MustChangePassword bool
	LastIP             string
	LastLocation       string
	LastMFAAt          time.Time
}

func (s *Service) gateMFA(ctx context.Context, email, ip, lastIP, lastLocation string, lastMFAAt time.Time) (Result, error) {
	location := s.geo.Lookup(ctx, ip)
	if !NeedsMFA(lastIP, ip, lastLocation, location, lastMFAAt, s.now()) {
		return s.issueSession(email), nil
	}

	code, err := s.codes.Issue(ctx, PurposeMFA, email)
	if errors.Is(err, ErrCooldown) {
		return Result{Step: StepMFA}, nil
	}
	if err != nil {
		return Result{}, err
	}
	if err := s.mailer.Send(
		email,
		"UpAnime — código de acesso",
		fmt.Sprintf("Seu código de acesso é: %s\n\nEle expira em 15 minutos.", code),
	); err != nil {
		return Result{}, err
	}
	return Result{Step: StepMFA}, nil
}

func (s *Service) issueSession(email string) Result {
	return Result{Step: StepOK, Token: s.signer.Sign(email, s.now().Add(sessionTTL))}
}

func NeedsMFA(lastIP, ip, lastLocation, location string, lastMFAAt, now time.Time) bool {
	if lastMFAAt.IsZero() {
		return true
	}
	if now.Sub(lastMFAAt) > mfaValidity {
		return true
	}
	if lastIP != ip {
		return true
	}
	if location != "" && lastLocation != location {
		return true
	}
	return false
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func GenerateTempPassword() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func GenerateSecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
