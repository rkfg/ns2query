package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/Philipp15b/go-steamapi"
	"go.etcd.io/bbolt"
	"rkfg.me/ns2query/db"
)

var (
	vanityRegex  = regexp.MustCompile(`https://steamcommunity.com/id/([^/]*)/?`)
	profileRegex = regexp.MustCompile(`https://steamcommunity.com/profiles/(\d*)/?`)
)

func playerIDFromDiscordName(username string) (uint32, error) {
	var discordName string
	err := bdb.View(func(t *bbolt.Tx) (err error) {
		discordName, err = db.NewLowercaseBucket(t).FindFirstValue(username)
		return
	})
	if err != nil {
		return 0, fmt.Errorf("discord user name starting with '%s' was not found", username)
	}
	return getBind(discordName)
}

func playerIDFromSteamID(player string) (uint32, error) {
	vanityName := vanityRegex.FindStringSubmatch(player)
	if vanityName != nil {
		player = vanityName[1]
	} else {
		profileID := profileRegex.FindStringSubmatch(player)
		if profileID != nil {
			player = profileID[1]
		}
	}
	steamid, err := steamapi.NewIdFromString(player)
	if err != nil {
		steamid, err = steamapi.NewIdFromVanityUrl(player, config.SteamKey)
		if err != nil {
			id64, err := strconv.ParseUint(player, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("steam ID %s not found", player)
			}
			steamid = steamapi.NewIdFrom64bit(id64)
		}
	}
	return steamid.As32Bit(), nil
}

func getPlayerAvatar(playerID uint32) string {
	sum, err := steamapi.GetPlayerSummaries([]uint64{steamapi.NewIdFrom32bit(playerID).As64Bit()}, config.SteamKey)
	if err != nil {
		log.Printf("Error getting avatar for player %d: %s", playerID, err)
		return ""
	}
	if len(sum) > 0 {
		return sum[0].SmallAvatarURL
	}
	log.Printf("No data found for player %d", playerID)
	return ""
}
