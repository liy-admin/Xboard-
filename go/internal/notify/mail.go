package notify

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTP 对应后台的邮件设置（mail_host / mail_port / mail_encryption / mail_username / mail_password /
// mail_from_address / app_name）。Encryption 为 "ssl"（隐式 TLS）、"tls"（STARTTLS）或空。
type SMTP struct {
	Host       string
	Port       int
	Encryption string
	Username   string
	Password   string
	From       string
	FromName   string
	Timeout    time.Duration
}

// Send 发送一封 HTML 邮件。
func (s SMTP) Send(to, subject, htmlBody string) error {
	if s.Host == "" || s.From == "" {
		return fmt.Errorf("mail is not configured")
	}
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	addr := net.JoinHostPort(s.Host, fmt.Sprint(s.Port))
	var conn net.Conn
	var err error
	dialer := &net.Dialer{Timeout: timeout}
	if strings.EqualFold(s.Encryption, "ssl") {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.Host})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if strings.EqualFold(s.Encryption, "tls") {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: s.Host}); err != nil {
				return err
			}
		}
	}
	if s.Username != "" {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
				return err
			}
		}
	}
	if err := c.Mail(s.From); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(buildMessage(s.From, s.FromName, to, subject, htmlBody, time.Now())); err != nil {
		w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// buildMessage 生成带 UTF-8 主题与 base64 正文的 MIME 邮件。
func buildMessage(from, fromName, to, subject, htmlBody string, at time.Time) []byte {
	var b strings.Builder
	fromHeader := from
	if fromName != "" {
		fromHeader = mime.QEncoding.Encode("utf-8", fromName) + " <" + from + ">"
	}
	fmt.Fprintf(&b, "From: %s\r\n", fromHeader)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", at.Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(htmlBody))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	return []byte(b.String())
}
