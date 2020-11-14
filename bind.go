package main

import (
	"github.com/bwmarrin/discordgo"
)

func bind(player string, user *discordgo.User) (id uint32, err error) {
	id, err = playerIDFromSteamID(player)
	if err != nil {
		return
	}
	return id, putBind(id, user)
}
