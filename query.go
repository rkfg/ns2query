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
	"rkfg.me/ns2query/db"
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
	for k, v := range srv.regularTimeouts {
		if v != nil && time.Now().After(*v) {
			delete(srv.regularTimeouts, k)
		}
	}
	bdb.View(func(t *bbolt.Tx) error {
		steamBucket := db.NewSteamToDiscordBucket(t)
		srv.regularNames = srv.regularNames[:0]
		for _, id := range ids {
			name, err := steamBucket.GetValue(id)
			if err == nil {
				srv.regularNames = append(srv.regularNames, name)
				if _, exists := srv.newRegulars[id]; !exists {
					if srv.regularTimeouts[id] == nil {
						log.Printf("Adding regular to announce %s", name)
						srv.newRegulars[id] = regular{id: id, name: name}
					} else {
						newTimeout := time.Now().Add(srv.RegularTimeout)
						srv.regularTimeouts[id] = &newTimeout
					}
				}
			}
		}
		return nil
	})
}

func (srv *ns2server) getPlayerIDs() (result []uint32, err error) {
	httpClient := http.Client{Timeout: time.Second * 3}
	resp, err := httpClient.Get(srv.IDURL)
	if err != nil {
		return nil, fmt.Errorf("Error querying %s: %w", srv.IDURL, err)
	} else {
		err = json.NewDecoder(resp.Body).Decode(&result)
	}
	return
}

func (srv *ns2server) announceRegulars() {
	defer func() {
		srv.newRegulars = map[uint32]regular{}
		srv.announceScheduled = false
	}()
	ids, err := srv.getPlayerIDs()
	if err != nil {
		log.Printf("Error getting IDs: %s", err)
		return
	}
	msg := ""
	idmap := map[uint32]struct{}{} // players currently on the server
	for _, id := range ids {
		idmap[id] = struct{}{}
	}
	for id, r := range srv.newRegulars {
		if _, ok := idmap[id]; ok { // only announce those who are playing
			if msg != "" {
				msg += ", "
			}
			msg += r.name
			newTimeout := time.Now().Add(srv.RegularTimeout)
			srv.regularTimeouts[id] = &newTimeout
		}
	}
	if msg != "" {
		channelID := srv.RegularChannelID
		if channelID == "" {
			channelID = config.ChannelID
		}
		sendChan <- message{MessageSend: &discordgo.MessageSend{
			Embed: &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("%s [%s]", srv.Name, srv.currentMap),
				Footer:      &discordgo.MessageEmbedFooter{Text: "Recently joined"},
				Description: msg,
				Color:       0x00aaff,
			},
		}, channelID: channelID}
	}
}

func (srv *ns2server) idsLoop() {
	var announceChan <-chan time.Time
	srv.newRegulars = map[uint32]regular{}
	for {
		ids, err := srv.getPlayerIDs()
		if err != nil {
			log.Printf("Error decoding ids: %s", err)
			continue
		}
		srv.checkRegulars(ids)
		if len(srv.newRegulars) == 0 {
			// make sure this never fires too early if there are no queued regulars
			announceChan = time.After(srv.QueryIDInterval * 5)
		} else {
			if !srv.announceScheduled {
				announceChan = time.After(srv.AnnounceDelay)
				srv.announceScheduled = true
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
