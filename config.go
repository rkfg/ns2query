package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type state int

const (
	empty state = iota
	seedingstarted
	almostfull
	specsonly
	full
)

type ns2server struct {
	Name               string
	Address            string
	SpecSlots          int `json:"spec_slots"`
	PlayerSlots        int `json:"player_slots"`
	players            []string
	serverState        state
	maxStateToMessage  state
	lastStatePromotion time.Time
}

func (s *ns2server) playersString() string {
	unknowns := 0
	result := ""
	for _, p := range s.players {
		if p == "Unknown" {
			unknowns++
		} else {
			if result == "" {
				result = p
			} else {
				result += ", " + p
			}
		}
	}
	if unknowns > 0 {
		suffix := ""
		if unknowns > 1 {
			suffix = "s"
		}
		if result == "" {
			return fmt.Sprintf("%d connecting player%s", unknowns, suffix)
		}
		return fmt.Sprintf("%s and %d connecting player%s", result, unknowns, suffix)
	}
	return result
}

type seeding struct {
	Seeding    int
	AlmostFull int `json:"almost_full"`
	Cooldown   int
}

var config struct {
	Token         string
	ChannelID     string        `json:"channel_id"`
	QueryInterval time.Duration `json:"query_interval"`
	Servers       []ns2server
	Seeding       seeding
}

func loadConfig() error {
	return loadConfigFilename("config.json")
}

func loadConfigFilename(filename string) error {
	if file, err := os.Open(filename); err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(&config)
	} else {
		return err
	}
	for i := range config.Servers {
		config.Servers[i].maxStateToMessage = full
	}
	if config.QueryInterval < 1 {
		return fmt.Errorf("invalid query interval in config.json: %d", config.QueryInterval)
	}
	if config.ChannelID == "" {
		return fmt.Errorf("specify channel_id in config.json")
	}
	if config.Token == "" {
		return fmt.Errorf("specify token in config.json")
	}
	return nil
}
