package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const noaaBase = "https://api.weather.gov/alerts/active"

type alertResult struct {
	ID          string `json:"id"`
	Event       string `json:"event"`
	Headline    string `json:"headline"`
	Severity    string `json:"severity"`
	Urgency     string `json:"urgency"`
	Certainty   string `json:"certainty"`
	AreaDesc    string `json:"areaDesc"`
	SenderName  string `json:"senderName"`
	Sent        string `json:"sent"`
	Effective   string `json:"effective"`
	Onset       string `json:"onset"`
	Expires     string `json:"expires"`
	Ends        string `json:"ends"`
	Description string `json:"description"`
	Instruction string `json:"instruction"`
}

// NOAA GeoJSON API response shapes

type noaaCollection struct {
	Features []noaaFeature `json:"features"`
}

type noaaFeature struct {
	ID         string         `json:"id"`
	Properties noaaProperties `json:"properties"`
}

type noaaProperties struct {
	Event       string  `json:"event"`
	Headline    *string `json:"headline"`
	Severity    string  `json:"severity"`
	Urgency     string  `json:"urgency"`
	Certainty   string  `json:"certainty"`
	AreaDesc    string  `json:"areaDesc"`
	SenderName  string  `json:"senderName"`
	Sent        string  `json:"sent"`
	Effective   string  `json:"effective"`
	Onset       *string `json:"onset"`
	Expires     string  `json:"expires"`
	Ends        *string `json:"ends"`
	Description *string `json:"description"`
	Instruction *string `json:"instruction"`
}

func fetchAlerts(cfg *appConfig) ([]alertResult, error) {
	seen := make(map[string]bool)
	var results []alertResult

	if len(cfg.Areas) > 0 {
		endpoint := noaaBase + "?area=" + strings.Join(cfg.Areas, ",")
		alerts, err := fetchFromEndpoint(endpoint, cfg.UserAgent)
		if err != nil {
			return nil, err
		}
		for _, a := range alerts {
			if !seen[a.ID] {
				seen[a.ID] = true
				results = append(results, a)
			}
		}
	}

	if len(cfg.Zones) > 0 {
		endpoint := noaaBase + "?zone=" + strings.Join(cfg.Zones, ",")
		alerts, err := fetchFromEndpoint(endpoint, cfg.UserAgent)
		if err != nil {
			return nil, err
		}
		for _, a := range alerts {
			if !seen[a.ID] {
				seen[a.ID] = true
				results = append(results, a)
			}
		}
	}

	return filterAlerts(results, cfg), nil
}

func fetchFromEndpoint(endpoint, userAgent string) ([]alertResult, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", endpoint, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", endpoint, resp.StatusCode)
	}

	var collection noaaCollection
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", endpoint, err)
	}

	var alerts []alertResult
	for _, f := range collection.Features {
		if a, ok := parseFeature(f); ok {
			alerts = append(alerts, a)
		}
	}
	return alerts, nil
}

func parseFeature(f noaaFeature) (alertResult, bool) {
	if f.ID == "" {
		return alertResult{}, false
	}
	p := f.Properties
	a := alertResult{
		ID:         f.ID,
		Event:      p.Event,
		Severity:   p.Severity,
		Urgency:    p.Urgency,
		Certainty:  p.Certainty,
		AreaDesc:   p.AreaDesc,
		SenderName: p.SenderName,
		Sent:       p.Sent,
		Effective:  p.Effective,
		Expires:    p.Expires,
	}
	if p.Headline != nil {
		a.Headline = *p.Headline
	}
	if p.Onset != nil {
		a.Onset = *p.Onset
	}
	if p.Ends != nil {
		a.Ends = *p.Ends
	}
	if p.Description != nil {
		a.Description = *p.Description
	}
	if p.Instruction != nil {
		a.Instruction = *p.Instruction
	}
	return a, true
}

func filterAlerts(alerts []alertResult, cfg *appConfig) []alertResult {
	if len(cfg.EventTypes) == 0 && len(cfg.Severity) == 0 {
		return alerts
	}

	eventSet := make(map[string]bool, len(cfg.EventTypes))
	for _, e := range cfg.EventTypes {
		eventSet[strings.ToLower(e)] = true
	}
	severitySet := make(map[string]bool, len(cfg.Severity))
	for _, s := range cfg.Severity {
		severitySet[strings.ToLower(s)] = true
	}

	var filtered []alertResult
	for _, a := range alerts {
		if len(eventSet) > 0 && !eventSet[strings.ToLower(a.Event)] {
			continue
		}
		if len(severitySet) > 0 && !severitySet[strings.ToLower(a.Severity)] {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered
}
