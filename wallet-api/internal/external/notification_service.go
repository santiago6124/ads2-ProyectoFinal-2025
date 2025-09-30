package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ExternalNotificationService interface {
	SendEmail(ctx context.Context, req *EmailRequest) error
	SendSMS(ctx context.Context, req *SMSRequest) error
	SendPushNotification(ctx context.Context, req *PushNotificationRequest) error
	SendWebhook(ctx context.Context, req *WebhookRequest) error
	SendSlackNotification(ctx context.Context, req *SlackNotificationRequest) error
}

type externalNotificationService struct {
	emailConfig *EmailConfig
	smsConfig   *SMSConfig
	pushConfig  *PushConfig
	webhookConfig *WebhookConfig
	slackConfig *SlackConfig
	httpClient  *http.Client
}

type EmailConfig struct {
	APIKey      string
	APISecret   string
	BaseURL     string
	FromEmail   string
	FromName    string
	Provider    string // "sendgrid", "mailgun", "ses"
}

type SMSConfig struct {
	APIKey    string
	APISecret string
	BaseURL   string
	Provider  string // "twilio", "nexmo", "aws"
	FromNumber string
}

type PushConfig struct {
	APIKey   string
	BaseURL  string
	Provider string // "firebase", "apns", "onesignal"
}

type WebhookConfig struct {
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

type SlackConfig struct {
	WebhookURL string
	Channel    string
	Username   string
}

func NewExternalNotificationService(
	emailConfig *EmailConfig,
	smsConfig *SMSConfig,
	pushConfig *PushConfig,
	webhookConfig *WebhookConfig,
	slackConfig *SlackConfig,
) ExternalNotificationService {
	if webhookConfig == nil {
		webhookConfig = &WebhookConfig{
			Timeout:       30 * time.Second,
			RetryAttempts: 3,
			RetryDelay:    5 * time.Second,
		}
	}

	return &externalNotificationService{
		emailConfig:   emailConfig,
		smsConfig:     smsConfig,
		pushConfig:    pushConfig,
		webhookConfig: webhookConfig,
		slackConfig:   slackConfig,
		httpClient: &http.Client{
			Timeout: webhookConfig.Timeout,
		},
	}
}

// Request types
type EmailRequest struct {
	To          []string               `json:"to"`
	CC          []string               `json:"cc,omitempty"`
	BCC         []string               `json:"bcc,omitempty"`
	Subject     string                 `json:"subject"`
	PlainText   string                 `json:"plain_text,omitempty"`
	HTML        string                 `json:"html,omitempty"`
	TemplateID  string                 `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`
	Attachments []EmailAttachment      `json:"attachments,omitempty"`
	Priority    string                 `json:"priority,omitempty"`
	ReplyTo     string                 `json:"reply_to,omitempty"`
}

type EmailAttachment struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"content"`
	ContentType string `json:"content_type"`
}

type SMSRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
	From    string `json:"from,omitempty"`
	Type    string `json:"type,omitempty"` // "transactional", "marketing"
}

type PushNotificationRequest struct {
	DeviceTokens []string               `json:"device_tokens"`
	Title        string                 `json:"title"`
	Body         string                 `json:"body"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Sound        string                 `json:"sound,omitempty"`
	Badge        int                    `json:"badge,omitempty"`
	Category     string                 `json:"category,omitempty"`
	ClickAction  string                 `json:"click_action,omitempty"`
}

type WebhookRequest struct {
	URL     string                 `json:"url"`
	Method  string                 `json:"method"`
	Headers map[string]string      `json:"headers,omitempty"`
	Payload map[string]interface{} `json:"payload"`
	Timeout time.Duration          `json:"timeout,omitempty"`
}

type SlackNotificationRequest struct {
	Channel     string                 `json:"channel,omitempty"`
	Username    string                 `json:"username,omitempty"`
	Text        string                 `json:"text"`
	Attachments []SlackAttachment      `json:"attachments,omitempty"`
	IconEmoji   string                 `json:"icon_emoji,omitempty"`
	IconURL     string                 `json:"icon_url,omitempty"`
}

type SlackAttachment struct {
	Color      string                 `json:"color,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Text       string                 `json:"text,omitempty"`
	Fields     []SlackField           `json:"fields,omitempty"`
	Footer     string                 `json:"footer,omitempty"`
	Timestamp  int64                  `json:"ts,omitempty"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Email implementation
func (s *externalNotificationService) SendEmail(ctx context.Context, req *EmailRequest) error {
	if s.emailConfig == nil {
		return fmt.Errorf("email configuration not provided")
	}

	switch s.emailConfig.Provider {
	case "sendgrid":
		return s.sendEmailViaSendGrid(ctx, req)
	case "mailgun":
		return s.sendEmailViaMailgun(ctx, req)
	case "ses":
		return s.sendEmailViaSES(ctx, req)
	default:
		return fmt.Errorf("unsupported email provider: %s", s.emailConfig.Provider)
	}
}

func (s *externalNotificationService) sendEmailViaSendGrid(ctx context.Context, req *EmailRequest) error {
	// Prepare SendGrid API request
	payload := map[string]interface{}{
		"from": map[string]string{
			"email": s.emailConfig.FromEmail,
			"name":  s.emailConfig.FromName,
		},
		"personalizations": []map[string]interface{}{
			{
				"to": s.formatEmailAddresses(req.To),
			},
		},
		"subject": req.Subject,
	}

	// Add CC and BCC if provided
	if len(req.CC) > 0 {
		payload["personalizations"].([]map[string]interface{})[0]["cc"] = s.formatEmailAddresses(req.CC)
	}
	if len(req.BCC) > 0 {
		payload["personalizations"].([]map[string]interface{})[0]["bcc"] = s.formatEmailAddresses(req.BCC)
	}

	// Add content
	var content []map[string]string
	if req.PlainText != "" {
		content = append(content, map[string]string{
			"type":  "text/plain",
			"value": req.PlainText,
		})
	}
	if req.HTML != "" {
		content = append(content, map[string]string{
			"type":  "text/html",
			"value": req.HTML,
		})
	}
	payload["content"] = content

	// Add template if provided
	if req.TemplateID != "" {
		payload["template_id"] = req.TemplateID
		if req.TemplateData != nil {
			payload["personalizations"].([]map[string]interface{})[0]["dynamic_template_data"] = req.TemplateData
		}
	}

	// Make API request
	return s.makeEmailAPIRequest(ctx, "https://api.sendgrid.com/v3/mail/send", "POST", payload, map[string]string{
		"Authorization": "Bearer " + s.emailConfig.APIKey,
		"Content-Type":  "application/json",
	})
}

func (s *externalNotificationService) sendEmailViaMailgun(ctx context.Context, req *EmailRequest) error {
	// Prepare form data for Mailgun
	data := url.Values{}
	data.Set("from", fmt.Sprintf("%s <%s>", s.emailConfig.FromName, s.emailConfig.FromEmail))
	data.Set("to", strings.Join(req.To, ","))
	data.Set("subject", req.Subject)

	if req.PlainText != "" {
		data.Set("text", req.PlainText)
	}
	if req.HTML != "" {
		data.Set("html", req.HTML)
	}

	// Add template variables if provided
	if req.TemplateData != nil {
		for key, value := range req.TemplateData {
			data.Set(fmt.Sprintf("v:%s", key), fmt.Sprintf("%v", value))
		}
	}

	// Make API request
	apiURL := fmt.Sprintf("%s/messages", s.emailConfig.BaseURL)

	req_http, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req_http.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req_http.SetBasicAuth("api", s.emailConfig.APIKey)

	resp, err := s.httpClient.Do(req_http)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *externalNotificationService) sendEmailViaSES(ctx context.Context, req *EmailRequest) error {
	// AWS SES implementation would go here
	// For now, return not implemented
	return fmt.Errorf("AWS SES integration not implemented")
}

// SMS implementation
func (s *externalNotificationService) SendSMS(ctx context.Context, req *SMSRequest) error {
	if s.smsConfig == nil {
		return fmt.Errorf("SMS configuration not provided")
	}

	switch s.smsConfig.Provider {
	case "twilio":
		return s.sendSMSViaTwilio(ctx, req)
	case "nexmo":
		return s.sendSMSViaNexmo(ctx, req)
	default:
		return fmt.Errorf("unsupported SMS provider: %s", s.smsConfig.Provider)
	}
}

func (s *externalNotificationService) sendSMSViaTwilio(ctx context.Context, req *SMSRequest) error {
	fromNumber := req.From
	if fromNumber == "" {
		fromNumber = s.smsConfig.FromNumber
	}

	// Prepare Twilio API request
	data := url.Values{}
	data.Set("From", fromNumber)
	data.Set("To", req.To)
	data.Set("Body", req.Message)

	apiURL := fmt.Sprintf("%s/Messages.json", s.smsConfig.BaseURL)

	req_http, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create SMS request: %w", err)
	}

	req_http.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req_http.SetBasicAuth(s.smsConfig.APIKey, s.smsConfig.APISecret)

	resp, err := s.httpClient.Do(req_http)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SMS API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *externalNotificationService) sendSMSViaNexmo(ctx context.Context, req *SMSRequest) error {
	// Vonage (Nexmo) implementation
	payload := map[string]interface{}{
		"from": s.smsConfig.FromNumber,
		"to":   req.To,
		"text": req.Message,
		"type": "text",
	}

	return s.makeSMSAPIRequest(ctx, s.smsConfig.BaseURL+"/sms/json", "POST", payload, map[string]string{
		"Content-Type": "application/json",
	})
}

// Push notification implementation
func (s *externalNotificationService) SendPushNotification(ctx context.Context, req *PushNotificationRequest) error {
	if s.pushConfig == nil {
		return fmt.Errorf("push notification configuration not provided")
	}

	switch s.pushConfig.Provider {
	case "firebase":
		return s.sendPushViaFirebase(ctx, req)
	case "onesignal":
		return s.sendPushViaOneSignal(ctx, req)
	default:
		return fmt.Errorf("unsupported push notification provider: %s", s.pushConfig.Provider)
	}
}

func (s *externalNotificationService) sendPushViaFirebase(ctx context.Context, req *PushNotificationRequest) error {
	// Firebase Cloud Messaging implementation
	payload := map[string]interface{}{
		"registration_ids": req.DeviceTokens,
		"notification": map[string]interface{}{
			"title": req.Title,
			"body":  req.Body,
		},
	}

	if req.Data != nil {
		payload["data"] = req.Data
	}

	if req.Sound != "" {
		payload["notification"].(map[string]interface{})["sound"] = req.Sound
	}

	return s.makePushAPIRequest(ctx, "https://fcm.googleapis.com/fcm/send", "POST", payload, map[string]string{
		"Authorization": "key=" + s.pushConfig.APIKey,
		"Content-Type":  "application/json",
	})
}

func (s *externalNotificationService) sendPushViaOneSignal(ctx context.Context, req *PushNotificationRequest) error {
	// OneSignal implementation
	payload := map[string]interface{}{
		"app_id":               s.pushConfig.APIKey,
		"include_player_ids":   req.DeviceTokens,
		"headings":            map[string]string{"en": req.Title},
		"contents":            map[string]string{"en": req.Body},
	}

	if req.Data != nil {
		payload["data"] = req.Data
	}

	return s.makePushAPIRequest(ctx, "https://onesignal.com/api/v1/notifications", "POST", payload, map[string]string{
		"Authorization": "Basic " + s.pushConfig.APIKey,
		"Content-Type":  "application/json",
	})
}

// Webhook implementation
func (s *externalNotificationService) SendWebhook(ctx context.Context, req *WebhookRequest) error {
	timeout := req.Timeout
	if timeout == 0 {
		timeout = s.webhookConfig.Timeout
	}

	// Create HTTP client with specific timeout
	client := &http.Client{Timeout: timeout}

	var body io.Reader
	if req.Payload != nil {
		jsonData, err := json.Marshal(req.Payload)
		if err != nil {
			return fmt.Errorf("failed to marshal webhook payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, body)
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	if req.Payload != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Retry logic
	var lastErr error
	for attempt := 0; attempt < s.webhookConfig.RetryAttempts; attempt++ {
		resp, err := client.Do(httpReq)
		if err != nil {
			lastErr = err
			if attempt < s.webhookConfig.RetryAttempts-1 {
				time.Sleep(s.webhookConfig.RetryDelay * time.Duration(attempt+1))
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		// Read error response
		errorBody, _ := io.ReadAll(resp.Body)
		lastErr = fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(errorBody))

		// Only retry on server errors
		if resp.StatusCode < 500 {
			break
		}

		if attempt < s.webhookConfig.RetryAttempts-1 {
			time.Sleep(s.webhookConfig.RetryDelay * time.Duration(attempt+1))
		}
	}

	return fmt.Errorf("webhook failed after %d attempts: %w", s.webhookConfig.RetryAttempts, lastErr)
}

// Slack notification implementation
func (s *externalNotificationService) SendSlackNotification(ctx context.Context, req *SlackNotificationRequest) error {
	if s.slackConfig == nil {
		return fmt.Errorf("Slack configuration not provided")
	}

	// Prepare Slack webhook payload
	payload := map[string]interface{}{
		"text": req.Text,
	}

	// Use config defaults if not specified
	channel := req.Channel
	if channel == "" {
		channel = s.slackConfig.Channel
	}
	if channel != "" {
		payload["channel"] = channel
	}

	username := req.Username
	if username == "" {
		username = s.slackConfig.Username
	}
	if username != "" {
		payload["username"] = username
	}

	if req.Attachments != nil {
		payload["attachments"] = req.Attachments
	}

	if req.IconEmoji != "" {
		payload["icon_emoji"] = req.IconEmoji
	}

	if req.IconURL != "" {
		payload["icon_url"] = req.IconURL
	}

	// Send to Slack
	return s.makeWebhookRequest(ctx, s.slackConfig.WebhookURL, payload)
}

// Helper methods
func (s *externalNotificationService) formatEmailAddresses(addresses []string) []map[string]string {
	var formatted []map[string]string
	for _, addr := range addresses {
		formatted = append(formatted, map[string]string{"email": addr})
	}
	return formatted
}

func (s *externalNotificationService) makeEmailAPIRequest(ctx context.Context, url, method string, payload interface{}, headers map[string]string) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *externalNotificationService) makeSMSAPIRequest(ctx context.Context, url, method string, payload interface{}, headers map[string]string) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SMS API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *externalNotificationService) makePushAPIRequest(ctx context.Context, url, method string, payload interface{}, headers map[string]string) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push notification API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *externalNotificationService) makeWebhookRequest(ctx context.Context, url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}