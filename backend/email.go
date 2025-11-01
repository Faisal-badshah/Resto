package main

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
)

func sendEmail(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	
	if host == "" || user == "" {
		fmt.Println("SMTP not configured, skipping email to:", to)
		return nil
	}
	
	if from == "" {
		from = user
	}
	
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body)
	
	auth := smtp.PlainAuth("", user, pass, host)
	
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}
	
	addr := host + ":" + port
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	
	if err = client.Auth(auth); err != nil {
		return err
	}
	
	if err = client.Mail(from); err != nil {
		return err
	}
	
	if err = client.Rcpt(to); err != nil {
		return err
	}
	
	w, err := client.Data()
	if err != nil {
		return err
	}
	
	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}
	
	err = w.Close()
	if err != nil {
		return err
	}
	
	return client.Quit()
}
