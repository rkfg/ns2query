package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

type hive struct {
	Alias           string
	Skill           int
	SkillOffset     int `json:"skill_offset"`
	CommSkill       int `json:"comm_skill"`
	CommSkillOffset int `json:"comm_skill_offset"`
}

func getSkill(playerID uint32) (*discordgo.MessageSend, error) {
	url := fmt.Sprintf("http://hive2.ns2cdt.com/api/get/playerData/%d", playerID)
	hiveResp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	var skill hive
	json.NewDecoder(hiveResp.Body).Decode(&skill)
	if skill.Alias == "" {
		return nil, fmt.Errorf("player id %d isn't present on Hive", playerID)
	}
	return &discordgo.MessageSend{Embed: &discordgo.MessageEmbed{
		Description: "Skill breakdown",
		Author:      &discordgo.MessageEmbedAuthor{Name: skill.Alias, IconURL: getPlayerAvatar(playerID)},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Marine (field/comm)",
				Value:  fmt.Sprintf("%d/%d", skill.Skill+skill.SkillOffset, skill.CommSkill+skill.CommSkillOffset),
				Inline: true,
			},
			{
				Name:   "Alien (field/comm)",
				Value:  fmt.Sprintf("%d/%d", skill.Skill-skill.SkillOffset, skill.CommSkill-skill.CommSkillOffset),
				Inline: true,
			},
		},
	}}, nil
}
