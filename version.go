package main

import (
	"github.com/bwmarrin/discordgo"
)

var (
	version = "unknown"
	date    = "unknown"
	source  = "https://github.com/rkfg/ns2query"
)

func versionEmbed() *discordgo.MessageSend {
	return &discordgo.MessageSend{Embed: &discordgo.MessageEmbed{
		Title:       "Version " + version,
		Description: "Built on " + date,
		URL:         source,
	}}
}
