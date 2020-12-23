package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var (
	version = "unknown"
	date    = "unknown"
	source  = "https://github.com/rkfg/ns2query"
)

func versionString() string {
	return fmt.Sprintf("Version %s built on %s. Source: %s", version, date, source)
}

func versionEmbed() *discordgo.MessageSend {
	return &discordgo.MessageSend{Embed: &discordgo.MessageEmbed{
		Title:       "Version " + version,
		Description: "Built on " + date,
		URL:         source,
	}}
}
