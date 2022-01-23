package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rumblefrog/go-a2s"
	"go.etcd.io/bbolt"
)

const (
	timeFormat = "2 Jan 2006 15:04:05 -0700"
)

type queryError struct {
	description string
	err         error
}

func (e queryError) Error() string {
	return fmt.Sprintf(e.description, e.err)
}

func (srv *ns2server) serverStatus() *discordgo.MessageSend {
	specSlots := srv.SpecSlots
	playerSlots := srv.PlayerSlots - len(srv.players)
	freeSlots := playerSlots + srv.SpecSlots
	if freeSlots < specSlots {
		specSlots = freeSlots
	}
	if playerSlots < 0 {
		playerSlots = 0
	}
	msg := discordgo.MessageSend{Embed: &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s [%s]", srv.Name, srv.currentMap),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Players",
				Value:  fmt.Sprint(len(srv.players)),
				Inline: true,
			},
			{
				Name:   "Player slots",
				Value:  fmt.Sprint(playerSlots),
				Inline: true,
			},
			{
				Name:   "Spectator slots",
				Value:  fmt.Sprint(specSlots),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Skill: %d", srv.avgSkill)},
	},
	}
	if srv.failures > config.FailureLimit && srv.downSince != nil {
		msg.Embed.Title = fmt.Sprintf("%s [%s] currently DOWN since %s", srv.Name, srv.currentMap, srv.downSince.Format(timeFormat))
	}
	playersCount := len(srv.players)
	if playersCount < config.Seeding.AlmostFull {
		msg.Embed.Color = 0x009900
	} else if playersCount < srv.PlayerSlots {
		msg.Embed.Color = 0xcc9900
	} else if playersCount >= srv.PlayerSlots {
		msg.Embed.Color = 0xff3300
	}
	if len(srv.regularNames) > 0 {
		msg.Embed.Fields = append(msg.Embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Regulars",
			Value: strings.Join(srv.regularNames, ", "),
		})
	}
	return &msg
}

func (srv *ns2server) maybeNotify() {
	playersCount := len(srv.players)
	newState := empty
	if playersCount < config.Seeding.Seeding {
		newState = empty
	} else if playersCount < config.Seeding.AlmostFull {
		newState = seedingstarted
	} else if playersCount < srv.PlayerSlots {
		newState = almostfull
	} else if playersCount < srv.PlayerSlots+srv.SpecSlots {
		newState = specsonly
	} else {
		newState = full
	}
	if newState > srv.serverState && newState <= srv.maxStateToMessage {
		srv.lastStatePromotion = time.Now()
		srv.serverState = newState
		if srv.lastStateAnnounced != newState {
			srv.lastStateAnnounced = newState
			msg := srv.serverStatus()
			switch newState {
			case seedingstarted:
				msg.Embed.Description = "Seeding started! Players on the server: " + srv.playersString()
				srv.maxStateToMessage = specsonly
				sendChan <- message{MessageSend: msg}
			case almostfull:
				msg.Embed.Description = "Server is almost full!"
				sendChan <- message{MessageSend: msg}
			case specsonly:
				msg.Embed.Description = "Server is full but you can still make it!"
				srv.maxStateToMessage = seedingstarted
				sendChan <- message{MessageSend: msg}
			}
		}
	} else {
		if time.Since(srv.lastStatePromotion).Seconds() > float64(config.Seeding.Cooldown) {
			srv.serverState = newState
			if newState == empty {
				// if the server goes empty we should allow seeding messages again
				srv.lastStateAnnounced = empty
			}
		}
	}
}

func (srv *ns2server) queryServer(client *a2s.Client) error {
	info, err := client.QueryInfo()
	if err != nil {
		return queryError{"server info query: %s", err}
	}
	srv.currentMap = info.Map
	rules, err := client.QueryRules()
	if err != nil {
		return queryError{"rules query: %s", err}
	}
	srv.avgSkill = 0
	avgSkillStr := rules.Rules["AverageSkill"]
	if avgSkillStr != "nan" && avgSkillStr != "" {
		avgSkill, err := strconv.ParseFloat(avgSkillStr, 32)
		if err != nil {
			return queryError{"parsing avg skill: %s", err}
		}
		srv.avgSkill = int(avgSkill)
	}
	playersInfo, err := client.QueryPlayer()
	if err != nil {
		return queryError{"player query: %s", err}
	}
	srv.players = srv.players[:0]
	for _, p := range playersInfo.Players {
		srv.players = append(srv.players, p.Name)
	}
	srv.maybeNotify()
	return nil
}

func (srv *ns2server) serverLoop() {
	client, err := a2s.NewClient(srv.Address)
	if err != nil {
		log.Println("error creating client:", err)
		return
	}
	defer client.Close()
	log.Printf("Client created for %s [%s]", srv.Name, srv.Address)
	for {
		err := srv.queryServer(client)
		if err != nil {
			log.Printf("Error: %s", err)
			srv.failures++
			if srv.failures > config.FailureLimit && srv.downSince == nil {
				now := time.Now().In(time.UTC)
				srv.downSince = &now
				sendChan <- message{MessageSend: &discordgo.MessageSend{Content: fmt.Sprintf("Server %s is down!", srv.Name)}}
			}
		} else {
			if srv.failures > config.FailureLimit && srv.downSince != nil {
				sendChan <- message{MessageSend: &discordgo.MessageSend{Content: fmt.Sprintf("Server %s is back up! Was down since: %s",
					srv.Name, srv.downSince.Format(timeFormat))}}
				srv.downSince = nil
			}
			srv.failures = 0
		}
		if err, ok := err.(queryError); ok {
			if err, ok := err.err.(*net.OpError); ok && err.Op == "write" {
				log.Println("Error during sending data (our IP changed?), restarting myself")
				close(srv.restartChan)
				return
			}
		}
		select {
		case <-time.After(config.QueryInterval):
		case <-srv.restartChan:
			log.Printf("Restart request received, stopping server polling: %s [%s]", srv.Name, srv.Address)
			return
		}
	}
}

func (srv *ns2server) checkRegulars(ids []uint32) {
	bdb.View(func(t *bbolt.Tx) error {
		steamBucket := newSteamToDiscordBucket(t)
		srv.regularNames = srv.regularNames[:0]
		for _, id := range ids {
			name, err := steamBucket.get(id)
			if err == nil {
				srv.regularNames = append(srv.regularNames, name)
				timeout := srv.regularTimeouts[id]
				if timeout == nil {
					log.Printf("Adding regular %s", name)
					srv.newRegularNames = append(srv.newRegularNames, name)
				}
				newTimeout := time.Now().Add(srv.RegularTimeout)
				srv.regularTimeouts[id] = &newTimeout
			}
		}
		for k, v := range srv.regularTimeouts {
			if v != nil && time.Now().After(*v) {
				delete(srv.regularTimeouts, k)
			}
		}
		return nil
	})
}

func (srv *ns2server) announceRegulars() {
	sendChan <- message{MessageSend: &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s [%s]", srv.Name, srv.currentMap),
			Footer:      &discordgo.MessageEmbedFooter{Text: "Recently joined"},
			Description: strings.Join(srv.newRegularNames, ", "),
			Color:       0x00aaff,
		},
	}}
	srv.newRegularNames = srv.newRegularNames[:0]
	srv.newRegulars = false
}

func (srv *ns2server) idsLoop() {
	httpClient := http.Client{Timeout: time.Second * 3}
	var announceChan <-chan time.Time
	for {
		resp, err := httpClient.Get(srv.IDURL)
		ids := []uint32{}
		if err != nil {
			log.Printf("Error querying %s: %s", srv.IDURL, err)
		} else {
			err = json.NewDecoder(resp.Body).Decode(&ids)
			if err != nil {
				log.Printf("Error decoding ids: %s", err)
			} else {
				srv.checkRegulars(ids)
			}
		}
		if len(srv.newRegularNames) == 0 {
			// make sure this never fires too early if there are no queued regulars
			announceChan = time.After(srv.QueryIDInterval * 60)
		} else {
			if !srv.newRegulars {
				announceChan = time.After(srv.AnnounceDelay)
				srv.newRegulars = true
			}
		}
		select {
		case <-time.After(srv.QueryIDInterval):
		case <-announceChan:
			srv.announceRegulars()
		case <-srv.restartChan:
			log.Printf("Restart request received, stopping steam ids polling: %s [%s]", srv.Name, srv.Address)
			return
		}
	}
}

func (srv *ns2server) query() {
	srv.currentMap = "<unknown>"
	srv.avgSkill = 0
	srv.maxStateToMessage = full
	srv.lastStateAnnounced = empty
	go srv.serverLoop()
	if srv.IDURL != "" {
		go srv.idsLoop()
	}
}
