package model

import "time"

type User struct {
	ID                 int64
	Email              string
	PasswordHash       string
	MustChangePassword bool
	IsAdmin            bool
	LastIP             string
	LastLocation       string
	LastMFAAt          time.Time
}
