package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"
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

type regular struct {
	id   uint32
	name string
}

type ns2server struct {
	Name                 string        `json:"name"`
	Address              string        `json:"address"`
	SpecSlots            int           `json:"spec_slots"`
	PlayerSlots          int           `json:"player_slots"`
	StatusTemplate       string        `json:"status_template"`
	IDURL                string        `json:"id_url"`
	QueryIDInterval      time.Duration `json:"query_id_interval"`
	AnnounceDelay        time.Duration `json:"announce_delay"`
	RegularTimeout       time.Duration `json:"regular_timeout"`
	RegularChannelID     string        `json:"regular_channel_id"`
	DownNotifyDiscordIDs []string      `json:"down_notify_ids"`
	UpNotifyDiscordIDs   []string      `json:"up_notify_ids"`
	regularTimeouts      map[uint32]*time.Time
	regularNames         []string
	newRegulars          map[uint32]regular
	announceScheduled    bool
	statusTemplate       *template.Template
	players              []string
	serverState          state
	maxStateToMessage    state
	lastStateAnnounced   state
	lastStatePromotion   time.Time
	currentMap           string
	avgSkill             int
	restartChan          chan struct{}
	failures             int
	downSince            *time.Time
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

func idsToPing(ids []string) (result string) {
	for _, id := range ids {
		result += fmt.Sprintf("<@%s>", id)
	}
	return
}

func (s *ns2server) formatDowntimeMsg(down bool) string {
	if down {
		return fmt.Sprintf("Server %s is down! %s", s.Name, idsToPing(s.DownNotifyDiscordIDs))
	} else {
		return fmt.Sprintf("Server %s is back up! Was down since: %s %s",
			s.Name, s.downSince.Format(timeFormat), idsToPing(s.UpNotifyDiscordIDs))
	}
}

type seeding struct {
	Seeding    int               `json:"seeding"`
	AlmostFull int               `json:"almost_full"`
	Cooldown   int               `json:"cooldown"`
	PingRoles  map[string]string `json:"ping_roles"`
}

type thread struct {
	Join                    bool   `json:"join"`
	Meme                    bool   `json:"meme"`
	NoSelfUpvote            bool   `json:"no_self_upvote"`
	Competition             bool   `json:"competition"`
	CompetitionDeadline     int    `json:"competition_deadline"`
	CompetitionLength       int    `json:"competition_length"`
	CompetitionAnnouncement int    `json:"competition_announcement"`
	AnnounceWinnerTo        string `json:"announce_winner_to"`
}

type users map[string]string

var config struct {
	Token         string            `json:"token"`
	SteamKey      string            `json:"steam_key"`
	ChannelID     string            `json:"channel_id"`
	Threads       map[string]thread `json:"threads"`
	BoltDBPath    string            `json:"bdb_database_path"`
	QueryInterval time.Duration     `json:"query_interval"`
	FailureLimit  int               `json:"failure_limit"`
	QueryTimeout  time.Duration     `json:"query_timeout"`
	Servers       []*ns2server      `json:"servers"`
	Seeding       seeding           `json:"seeding"`
	Users         users             `json:"users"`
}

func loadConfig(path string) error {
	return loadConfigFilename(path)
}

func loadConfigFilename(filename string) error {
	if file, err := os.Open(filename); err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(&config)
	} else {
		return err
	}
	if config.QueryInterval < 1 {
		return fmt.Errorf("invalid query interval in config.json: %d", config.QueryInterval)
	} else {
		config.QueryInterval *= time.Second
	}
	if config.ChannelID == "" {
		return fmt.Errorf("specify channel_id in config.json")
	}
	if config.Token == "" {
		return fmt.Errorf("specify token in config.json")
	}
	if config.BoltDBPath == "" {
		return fmt.Errorf("specify bdb_database_path in config.json")
	}
	return nil
}
