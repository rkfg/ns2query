package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.etcd.io/bbolt"
)

func bind(player string, user *discordgo.User) (id uint32, err error) {
	id, err = playerIDFromSteamID(player)
	if err != nil {
		return
	}
	err = putBind(id, user)
	return
}

func putBind(playerID uint32, discordUser *discordgo.User) (err error) {
	return bdb.Update(func(t *bbolt.Tx) (err error) {
		name := discordUser.String()
		err = newUsersBucket(t).put(name, playerID)
		if err != nil {
			return
		}
		return newLowercaseBucket(t).put(name)
	})
}

func getBind(username string) (playerID uint32, err error) {
	err = bdb.View(func(t *bbolt.Tx) (err error) {
		playerID, err = newUsersBucket(t).get(username)
		return
	})
	if err != nil {
		return 0, fmt.Errorf("player %s isn't in the database. Use `-bind <Steam ID>` to register", username)
	}
	return
}

func deleteBind(user *discordgo.User) (err error) {
	return bdb.Update(func(t *bbolt.Tx) (err error) {
		err = newUsersBucket(t).del(user.String())
		if err != nil {
			return
		}
		err = newLowercaseBucket(t).del(strings.ToLower(user.String()))
		return
	})
}
