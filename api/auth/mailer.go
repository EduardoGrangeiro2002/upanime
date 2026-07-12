package auth

import (
	"fmt"
	"log"
	"net/smtp"
)

type Mailer interface {
	Send(to, subject, body string) error
}

type SMTPMailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPMailer(host, port, username, password, from string) *SMTPMailer {
	return &SMTPMailer{host: host, port: port, username: username, password: password, from: from}
}

func (m *SMTPMailer) Send(to, subject, body string) error {
	message := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		m.from, to, subject, body,
	)

	var authentication smtp.Auth
	if m.username != "" {
		authentication = smtp.PlainAuth("", m.username, m.password, m.host)
	}
	return smtp.SendMail(m.host+":"+m.port, authentication, m.from, []string{to}, []byte(message))
}

type LogMailer struct{}

func (l *LogMailer) Send(to, subject, body string) error {
	log.Printf("SMTP não configurado — email para %s NÃO enviado. Assunto: %q. Corpo:\n%s", to, subject, body)
	return nil
}
