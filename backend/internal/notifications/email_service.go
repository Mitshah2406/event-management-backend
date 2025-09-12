package notifications

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// EmailService interface for sending emails
type EmailService interface {
	SendNotification(ctx context.Context, notification *UnifiedNotification) error
	SendHTML(ctx context.Context, to, subject, htmlBody, textBody string) error
	SendTemplate(ctx context.Context, to, subject, templateName string, data interface{}) error
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    bool
	Timeout   time.Duration
}

// NewSMTPConfigFromEnv creates SMTP config from environment variables
func NewSMTPConfigFromEnv() *SMTPConfig {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}

	timeout, _ := time.ParseDuration(os.Getenv("SMTP_TIMEOUT"))
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &SMTPConfig{
		Host:      os.Getenv("SMTP_HOST"),
		Port:      port,
		Username:  os.Getenv("SMTP_USERNAME"),
		Password:  os.Getenv("SMTP_PASSWORD"),
		FromEmail: os.Getenv("FROM_EMAIL"),
		FromName:  "Evently",
		UseTLS:    true,
		Timeout:   timeout,
	}
}

// SMTPEmailService is a real SMTP implementation of the EmailService interface
type SMTPEmailService struct {
	config    *SMTPConfig
	templates map[string]*template.Template
}

// NewSMTPEmailService creates a new SMTP email service
func NewSMTPEmailService(config *SMTPConfig) *SMTPEmailService {
	// Validate configuration
	if err := validateSMTPConfig(config); err != nil {
		log.Fatalf("Invalid SMTP configuration: %v", err)
	}

	service := &SMTPEmailService{
		config:    config,
		templates: make(map[string]*template.Template),
	}

	// Load default templates
	service.loadDefaultTemplates()

	return service
}

// validateSMTPConfig validates SMTP configuration
func validateSMTPConfig(config *SMTPConfig) error {
	if config == nil {
		return fmt.Errorf("SMTP config is nil")
	}

	if config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("SMTP port must be between 1 and 65535")
	}

	if config.Username == "" {
		return fmt.Errorf("SMTP username is required")
	}

	if config.Password == "" {
		return fmt.Errorf("SMTP password is required")
	}

	if config.FromEmail == "" {
		return fmt.Errorf("From email is required")
	}

	return nil
}

// SendNotification sends a unified notification via email
func (s *SMTPEmailService) SendNotification(ctx context.Context, notification *UnifiedNotification) error {
	log.Printf("📧 [SMTP] Sending %s notification to %s (%s)",
		notification.Type,
		notification.RecipientEmail,
		notification.RecipientName,
	)

	// Generate email content from template if available
	htmlBody, textBody, err := s.generateContent(notification)
	if err != nil {
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	return s.SendHTML(ctx, notification.RecipientEmail, notification.Subject, htmlBody, textBody)
}

// SendHTML sends an HTML email
func (s *SMTPEmailService) SendHTML(ctx context.Context, to, subject, htmlBody, textBody string) error {
	// Create message
	message := s.buildMessage(to, subject, htmlBody, textBody)

	// Connect to SMTP server
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	var err error
	if s.config.UseTLS {
		err = s.sendWithSTARTTLS(addr, auth, to, message)
	} else {
		err = smtp.SendMail(addr, auth, s.config.FromEmail, []string{to}, message)
	}

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("📧 [SMTP] Email sent successfully to %s", to)
	return nil
}

// SendTemplate sends an email using a template
func (s *SMTPEmailService) SendTemplate(ctx context.Context, to, subject, templateName string, data interface{}) error {
	tmpl, exists := s.templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	var htmlBuf, textBuf bytes.Buffer

	// Execute HTML template
	if err := tmpl.ExecuteTemplate(&htmlBuf, "html", data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	// Execute text template (fallback to HTML if text template doesn't exist)
	if err := tmpl.ExecuteTemplate(&textBuf, "text", data); err != nil {
		// If text template doesn't exist, create a simple text version
		textBuf.WriteString("Please view this email in HTML format.")
	}

	return s.SendHTML(ctx, to, subject, htmlBuf.String(), textBuf.String())
}

// sendWithSTARTTLS sends email with STARTTLS encryption (recommended for Gmail)
func (s *SMTPEmailService) sendWithSTARTTLS(addr string, auth smtp.Auth, to string, message []byte) error {
	// Connect to SMTP server without TLS first
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Quit()

	// Start TLS
	tlsconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.config.Host,
	}

	if err = client.StartTLS(tlsconfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	// Set sender
	if err = client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return w.Close()
}

// sendWithTLS sends email with direct TLS encryption (legacy method)
func (s *SMTPEmailService) sendWithTLS(addr string, auth smtp.Auth, to string, message []byte) error {
	// Create TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.config.Host,
	}

	// Connect and send
	conn, err := tls.Dial("tcp", addr, tlsconfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return err
	}
	defer client.Quit()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	if err = client.Mail(s.config.FromEmail); err != nil {
		return err
	}

	if err = client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(message)
	if err != nil {
		return err
	}

	return w.Close()
}

// buildMessage creates the email message with proper headers
func (s *SMTPEmailService) buildMessage(to, subject, htmlBody, textBody string) []byte {
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	// Create multipart message
	boundary := "boundary_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	headers["Content-Type"] = fmt.Sprintf("multipart/alternative; boundary=%s", boundary)

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n"

	// Text part
	if textBody != "" {
		message += fmt.Sprintf("--%s\r\n", boundary)
		message += "Content-Type: text/plain; charset=UTF-8\r\n"
		message += "\r\n"
		message += textBody + "\r\n"
	}

	// HTML part
	if htmlBody != "" {
		message += fmt.Sprintf("--%s\r\n", boundary)
		message += "Content-Type: text/html; charset=UTF-8\r\n"
		message += "\r\n"
		message += htmlBody + "\r\n"
	}

	message += fmt.Sprintf("--%s--\r\n", boundary)

	return []byte(message)
}

// generateContent generates email content from notification data
func (s *SMTPEmailService) generateContent(notification *UnifiedNotification) (string, string, error) {
	// Use template if available
	if notification.TemplateID != "" {
		if tmpl, exists := s.templates[notification.TemplateID]; exists {
			var htmlBuf, textBuf bytes.Buffer

			if err := tmpl.ExecuteTemplate(&htmlBuf, "html", notification.TemplateData); err != nil {
				return "", "", err
			}

			tmpl.ExecuteTemplate(&textBuf, "text", notification.TemplateData)

			return htmlBuf.String(), textBuf.String(), nil
		}
	}

	// Generate default content based on notification type
	return s.generateDefaultContent(notification)
}

// generateDefaultContent creates default email content for notification types
func (s *SMTPEmailService) generateDefaultContent(notification *UnifiedNotification) (string, string, error) {
	data := notification.TemplateData

	switch notification.Type {
	case NotificationTypeWaitlistSpotAvailable:
		htmlBody := fmt.Sprintf(`
			<h2>🎉 Great News! A spot is available</h2>
			<p>Hi %s,</p>
			<p>A spot has become available for <strong>%s</strong>.</p>
			<p>You have until <strong>%v</strong> to secure your booking.</p>
			<p>Your position was #%v.</p>
			<p>Best regards,<br>Evently Team</p>
		`,
			notification.RecipientName,
			data["event_title"],
			data["expires_at"],
			data["position"],
		)

		textBody := fmt.Sprintf(
			"Hi %s,\n\nA spot has become available for %s.\nYou have until %v to secure your booking.\nYour position was #%v.\n\nBest regards,\nEvently Team",
			notification.RecipientName,
			data["event_title"],
			data["expires_at"],
			data["position"],
		)

		return htmlBody, textBody, nil

	case NotificationTypeBookingConfirmed:
		htmlBody := fmt.Sprintf(`
			<h2>✅ Booking Confirmed</h2>
			<p>Hi %s,</p>
			<p>Your booking for <strong>%s</strong> has been confirmed!</p>
			<p>Booking Number: <strong>%s</strong></p>
			<p>Quantity: %v tickets</p>
			<p>Total Amount: $%.2f</p>
			<p>Best regards,<br>Evently Team</p>
		`,
			notification.RecipientName,
			data["event_title"],
			data["booking_number"],
			data["quantity"],
			data["total_amount"],
		)

		textBody := fmt.Sprintf(
			"Hi %s,\n\nYour booking for %s has been confirmed!\nBooking Number: %s\nQuantity: %v tickets\nTotal Amount: $%.2f\n\nBest regards,\nEvently Team",
			notification.RecipientName,
			data["event_title"],
			data["booking_number"],
			data["quantity"],
			data["total_amount"],
		)

		return htmlBody, textBody, nil

	default:
		// Generic template
		htmlBody := fmt.Sprintf(`
			<h2>%s</h2>
			<p>Hi %s,</p>
			<p>This is a notification from Evently.</p>
			<p>Best regards,<br>Evently Team</p>
		`,
			notification.Subject,
			notification.RecipientName,
		)

		textBody := fmt.Sprintf(
			"Hi %s,\n\nThis is a notification from Evently.\n\nBest regards,\nEvently Team",
			notification.RecipientName,
		)

		return htmlBody, textBody, nil
	}
}

// loadDefaultTemplates loads default email templates
func (s *SMTPEmailService) loadDefaultTemplates() {
	// This would typically load from files or database
	// For now, we'll use the generateDefaultContent method
	log.Println("📧 Default email templates loaded")
}

type MockEmailService struct{}

// NewMockEmailService creates a new mock email service
func NewMockEmailService() *MockEmailService {
	return &MockEmailService{}
}

// SendNotification sends a mock notification
func (s *MockEmailService) SendNotification(ctx context.Context, notification *UnifiedNotification) error {
	log.Printf("📧 [MOCK] Sending %s notification to %s (%s)",
		notification.Type,
		notification.RecipientEmail,
		notification.RecipientName,
	)
	return nil
}

// SendHTML sends a mock HTML email
func (s *MockEmailService) SendHTML(ctx context.Context, to, subject, htmlBody, textBody string) error {
	log.Printf("� [MOCK] To: %s, Subject: %s", to, subject)
	log.Printf("� [MOCK] HTML Body: %s", strings.TrimSpace(htmlBody))
	return nil
}

// SendTemplate sends a mock template email
func (s *MockEmailService) SendTemplate(ctx context.Context, to, subject, templateName string, data interface{}) error {
	log.Printf("� [MOCK] Template: %s, To: %s, Subject: %s", templateName, to, subject)
	return nil
}
