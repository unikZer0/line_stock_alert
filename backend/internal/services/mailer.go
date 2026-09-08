package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type Mailer interface {
	SendVerificationOTP(to, otp string) error
}

type SMTPMailer struct {
	host, address, user, password, fromEmail, fromName string
	useTLS                                             bool
}

func NewSMTPMailer(host string, port int, user, password string, useTLS bool, fromEmail, fromName string) *SMTPMailer {
	return &SMTPMailer{
		host: host, address: fmt.Sprintf("%s:%d", host, port), user: user,
		password: password, useTLS: useTLS, fromEmail: fromEmail, fromName: fromName,
	}
}

func (m *SMTPMailer) SendVerificationOTP(to, otp string) error {
	subject := "Verify your Stock Alert account"
	body := fmt.Sprintf("Your verification code is %s. It expires soon. If you did not request this code, ignore this email.", otp)
	message := []byte(strings.Join([]string{
		"From: " + m.fromName + " <" + m.fromEmail + ">",
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n"))

	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.password, m.host)
	}
	if !m.useTLS {
		if err := smtp.SendMail(m.address, auth, m.fromEmail, []string{to}, message); err != nil {
			return fmt.Errorf("send verification email: %w", err)
		}
		return nil
	}

	conn, err := tls.Dial("tcp", m.address, &tls.Config{MinVersion: tls.VersionTLS12, ServerName: m.host})
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate SMTP client: %w", err)
		}
	}
	if err = client.Mail(m.fromEmail); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP body: %w", err)
	}
	if _, err = w.Write(message); err != nil {
		return fmt.Errorf("write SMTP body: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("close SMTP body: %w", err)
	}
	if err = client.Quit(); err != nil && !isConnectionClosed(err) {
		return fmt.Errorf("quit SMTP client: %w", err)
	}
	return nil
}

func isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok {
		return opErr.Err != nil
	}
	return false
}
