package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rumblefrog/go-a2s"
)

type queryError struct {
	description string
	err         error
}

func (e queryError) Error() string {
	return fmt.Sprintf(e.description, e.err)
}

func serverStatus(srv *ns2server) *discordgo.MessageSend {
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
	playersCount := len(srv.players)
	if playersCount < config.Seeding.AlmostFull {
		msg.Embed.Color = 0x009900
	} else if playersCount < srv.PlayerSlots {
		msg.Embed.Color = 0xcc9900
	} else if playersCount >= srv.PlayerSlots {
		msg.Embed.Color = 0xff3300
	}
	return &msg
}

func maybeNotify(srv *ns2server) {
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
			msg := serverStatus(srv)
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

func queryServer(client *a2s.Client, srv *ns2server) error {
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
	maybeNotify(srv)
	return nil
}

func query(srv *ns2server) {
	client, err := a2s.NewClient(srv.Address)
	if err != nil {
		log.Println("error creating client:", err)
		return
	}
	defer client.Close()
	log.Printf("Client created for %s [%s]", srv.Name, srv.Address)
	srv.currentMap = "<unknown>"
	srv.avgSkill = 0
	srv.maxStateToMessage = full
	srv.lastStateAnnounced = empty
	for {
		err = queryServer(client, srv)
		if err != nil {
			log.Printf("Error: %s", err)
		}
		if err, ok := err.(queryError); ok {
			if err, ok := err.err.(*net.OpError); ok && err.Op == "write" {
				log.Println("Error during sending data (our IP changed?), restarting myself")
				close(srv.restartChan)
				return
			}
		}
		select {
		case <-time.After(config.QueryInterval * time.Second):
		case <-srv.restartChan:
			log.Printf("Restart request received, stopping server polling: %s [%s]", srv.Name, srv.Address)
			return
		}
	}
}
