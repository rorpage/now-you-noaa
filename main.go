package main

import (
	"fmt"
	"log"
	"os"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	log.SetFlags(0)

	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}

	fmt.Printf("=== Now You NOAA %s ===\n", version)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("[config] %v", err)
	}
	log.Printf("[config] monitoring %d area(s), %d zone(s)", len(cfg.Areas), len(cfg.Zones))

	state := loadState(cfg.StateFilePath)
	state = pruneState(state, cfg.PruneAfterDays)

	alerts, err := fetchAlerts(cfg)
	if err != nil {
		log.Fatalf("[noaa] %v", err)
	}
	log.Printf("[noaa] %d active alert(s) found", len(alerts))

	notified := 0
	for _, alert := range alerts {
		if hasBeenNotified(state, alert.ID) {
			log.Printf("[state] already notified for alert %s, skipping", alert.ID)
			continue
		}
		if err := sendNotification(cfg, alert); err != nil {
			log.Printf("[notify] %v", err)
			continue
		}
		markNotified(&state, alert.ID)
		notified++
	}

	if err := saveState(cfg.StateFilePath, state); err != nil {
		log.Printf("[state] failed to save: %v", err)
		os.Exit(1)
	}

	fmt.Printf("=== Done. %d new notification(s) sent. ===\n", notified)
}
