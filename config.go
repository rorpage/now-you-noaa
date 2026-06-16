package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	defaultConfigFile = "/etc/now-you-noaa/config.json"
	defaultStateFile  = "/var/lib/now-you-noaa/state.json"
	defaultPruneDays  = 7
	defaultUserAgent  = "now-you-noaa (https://github.com/rorpage/now-you-noaa)"
)

type appConfig struct {
	Areas                []string          `json:"areas"`
	Zones                []string          `json:"zones"`
	EventTypes           []string          `json:"eventTypes"`
	Severity             []string          `json:"severity"`
	UserAgent            string            `json:"userAgent"`
	NotificationURL      string            `json:"notificationUrl"`
	NotificationMethod   string            `json:"notificationMethod"`
	NotificationHeaders  map[string]string `json:"notificationHeaders"`
	NotificationType     string            `json:"notificationType"`
	NotificationTemplate string            `json:"notificationTemplate"`
	StateFilePath        string            `json:"stateFilePath"`
	PruneAfterDays       int               `json:"pruneAfterDays"`
}

func loadConfig() (*appConfig, error) {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = defaultConfigFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg appConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if url := os.Getenv("NOTIFICATION_URL"); url != "" {
		cfg.NotificationURL = url
	}
	if cfg.NotificationURL == "" {
		return nil, fmt.Errorf("notification URL required: set NOTIFICATION_URL env var or notificationUrl in config")
	}

	if sf := os.Getenv("STATE_FILE"); sf != "" {
		cfg.StateFilePath = sf
	}
	if cfg.StateFilePath == "" {
		cfg.StateFilePath = defaultStateFile
	}

	if cfg.PruneAfterDays <= 0 {
		cfg.PruneAfterDays = defaultPruneDays
	}
	if cfg.NotificationMethod == "" {
		cfg.NotificationMethod = "POST"
	}
	if cfg.NotificationType == "" {
		cfg.NotificationType = "webhook"
	}
	switch cfg.NotificationType {
	case "webhook", "slack", "discord", "template":
	default:
		return nil, fmt.Errorf("invalid notificationType %q: must be webhook, slack, discord, or template", cfg.NotificationType)
	}
	if cfg.NotificationType == "template" && cfg.NotificationTemplate == "" {
		return nil, fmt.Errorf("notificationTemplate is required when notificationType is \"template\"")
	}

	if len(cfg.Areas) == 0 && len(cfg.Zones) == 0 {
		return nil, fmt.Errorf("at least one area or zone required in %s", path)
	}

	for i := range cfg.Areas {
		cfg.Areas[i] = strings.ToUpper(cfg.Areas[i])
	}
	for i := range cfg.Zones {
		cfg.Zones[i] = strings.ToUpper(cfg.Zones[i])
	}

	if cfg.UserAgent == "" {
		cfg.UserAgent = defaultUserAgent
	}

	return &cfg, nil
}
