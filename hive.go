package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Philipp15b/go-steamapi"
	"github.com/bwmarrin/discordgo"
)

type hive struct {
	GameName    string
	Playerstats struct {
		Stats []struct {
			Name  string
			Value float64
		}
	}
}

func getSkill(playerID uint32) (*discordgo.MessageSend, error) {
	steamID := uint64(playerID) + 0x110000100000000
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetUserStatsForGame/v2/?key=%s&steamid=%d&appid=4920", config.SteamKey, steamID)
	hiveResp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	var steamData hive
	json.NewDecoder(hiveResp.Body).Decode(&steamData)
	if steamData.GameName == "" {
		summary, err := steamapi.GetPlayerSummaries([]uint64{steamID}, config.SteamKey)
		if err != nil || len(summary) == 0 {
			log.Printf("Error getting player info: %s", err)
			steamData.GameName = "<err>"
		} else {
			steamData.GameName = summary[0].PersonaName
		}
	}
	skill := [4]int{}
	skillOffset := [4]int{}
	signs := [4]bool{}
	for _, s := range steamData.Playerstats.Stats {
		switch s.Name {
		case "skill":
			skill[0] = int(s.Value)
		case "skill_offset":
			skillOffset[0] = int(s.Value)
		case "comm_skill":
			skill[1] = int(s.Value)
		case "comm_skill_offset":
			skillOffset[1] = int(s.Value)
		case "td_skill":
			skill[2] = int(s.Value)
		case "td_skill_offset":
			skillOffset[2] = int(s.Value)
		case "td_comm_skill":
			skill[3] = int(s.Value)
		case "td_comm_skill_offset":
			skillOffset[3] = int(s.Value)
		case "skill_offset_sign":
			if int(s.Value) == 1 {
				signs[0] = true
			}
		case "comm_skill_offset_sign":
			if int(s.Value) == 1 {
				signs[1] = true
			}
		case "td_skill_offset_sign":
			if int(s.Value) == 1 {
				signs[2] = true
			}
		case "td_comm_skill_offset_sign":
			if int(s.Value) == 1 {
				signs[3] = true
			}
		}
	}
	for i, s := range signs {
		if s {
			skillOffset[i] = -skillOffset[i]
		}
	}
	return &discordgo.MessageSend{Embed: &discordgo.MessageEmbed{
		Description: "Skill breakdown",
		Author:      &discordgo.MessageEmbedAuthor{Name: steamData.GameName, IconURL: getPlayerAvatar(playerID)},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Marine (field/comm)",
				Value:  fmt.Sprintf("%d/%d", skill[0]-skillOffset[0], skill[1]-skillOffset[1]),
				Inline: true,
			},
			{
				Name:   "Alien (field/comm)",
				Value:  fmt.Sprintf("%d/%d", skill[0]+skillOffset[0], skill[1]+skillOffset[1]),
				Inline: true,
			},
			{},
			{
				Name:   "TD Marine (field/comm)",
				Value:  fmt.Sprintf("%d/%d", skill[2]-skillOffset[2], skill[3]-skillOffset[3]),
				Inline: true,
			},
			{
				Name:   "TD Alien (field/comm)",
				Value:  fmt.Sprintf("%d/%d", skill[2]+skillOffset[2], skill[3]+skillOffset[3]),
				Inline: true,
			},
		},
	}}, nil
}
