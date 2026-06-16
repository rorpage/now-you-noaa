package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
)

type notificationPayload struct {
	Alert   alertResult `json:"alert"`
	Summary string      `json:"summary"`
}

func buildPayload(alert alertResult) notificationPayload {
	summary := alert.Event
	if alert.Headline != "" {
		summary = alert.Headline
	} else if alert.AreaDesc != "" {
		summary = fmt.Sprintf("%s: %s", alert.Event, alert.AreaDesc)
	}
	return notificationPayload{Alert: alert, Summary: summary}
}

func buildBody(cfg *appConfig, payload notificationPayload) ([]byte, error) {
	switch cfg.NotificationType {
	case "slack":
		return json.Marshal(map[string]string{"text": payload.Summary})
	case "discord":
		return json.Marshal(map[string]string{"content": payload.Summary})
	case "template":
		tmpl, err := template.New("notification").Parse(cfg.NotificationTemplate)
		if err != nil {
			return nil, fmt.Errorf("parsing notification template: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, payload); err != nil {
			return nil, fmt.Errorf("executing notification template: %w", err)
		}
		return buf.Bytes(), nil
	default: // "webhook"
		return json.Marshal(payload)
	}
}

func sendNotification(cfg *appConfig, alert alertResult) error {
	payload := buildPayload(alert)

	body, err := buildBody(cfg, payload)
	if err != nil {
		return fmt.Errorf("building notification body: %w", err)
	}

	req, err := http.NewRequest(cfg.NotificationMethod, cfg.NotificationURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.NotificationHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", cfg.NotificationMethod, cfg.NotificationURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: HTTP %d", cfg.NotificationMethod, cfg.NotificationURL, resp.StatusCode)
	}

	fmt.Printf("[notify] %s\n", payload.Summary)
	return nil
}
