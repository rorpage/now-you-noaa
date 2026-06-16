package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type stateEntry struct {
	AlertID    string `json:"alertId"`
	NotifiedAt string `json:"notifiedAt"`
}

type appState struct {
	NotifiedAlerts []stateEntry `json:"notifiedAlerts"`
}

func loadState(path string) appState {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[state] could not read %s, starting fresh: %v", path, err)
		}
		return appState{}
	}
	var s appState
	if err := json.Unmarshal(data, &s); err != nil {
		log.Printf("[state] could not parse %s, starting fresh: %v", path, err)
		return appState{}
	}
	return s
}

func saveState(path string, s appState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func pruneState(s appState, days int) appState {
	cutoff := time.Now().AddDate(0, 0, -days)
	before := len(s.NotifiedAlerts)

	kept := s.NotifiedAlerts[:0]
	for _, e := range s.NotifiedAlerts {
		t, err := time.Parse(time.RFC3339, e.NotifiedAt)
		if err != nil || t.After(cutoff) {
			kept = append(kept, e)
		}
	}

	if pruned := before - len(kept); pruned > 0 {
		log.Printf("[state] pruned %d old entr%s (older than %d days)", pruned, pluralSuffix(pruned, "y", "ies"), days)
	}
	return appState{NotifiedAlerts: kept}
}

func hasBeenNotified(s appState, alertID string) bool {
	for _, e := range s.NotifiedAlerts {
		if e.AlertID == alertID {
			return true
		}
	}
	return false
}

func markNotified(s *appState, alertID string) {
	s.NotifiedAlerts = append(s.NotifiedAlerts, stateEntry{
		AlertID:    alertID,
		NotifiedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func pluralSuffix(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
