package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/syndtr/goleveldb/leveldb"
)

func bind(player string, user *discordgo.User) (id uint32, err error) {
	id, err = playerIDFromSteamID(player)
	if err != nil {
		return
	}
	return id, putBind(id, user)
}

func putBind(playerID uint32, discordUser *discordgo.User) (err error) {
	var tx *leveldb.Transaction
	tx, err = db.OpenTransaction()
	if err != nil {
		return
	}
	defer commitOrDiscard(tx, &err)
	if err = putUInt32(tx, normalPath, discordUser.String(), playerID); err != nil {
		return
	}
	if err = putLowercaseIndex(tx, discordUser.String()); err != nil {
		return
	}
	return
}

func putLowercaseIndex(tx *leveldb.Transaction, username string) error {
	return putString(tx, lowercasePath, strings.ToLower(username), username)
}

func getBind(username string) (playerID uint32, err error) {
	val, err := getUInt32(normalPath, username)
	if err == leveldb.ErrNotFound {
		return 0, fmt.Errorf("player %s isn't in the database. Use `-bind <Steam ID>` to register", username)
	}
	if err != nil {
		return 0, err
	}
	return val, nil
}

func deleteBind(user *discordgo.User) (err error) {
	var tx *leveldb.Transaction
	tx, err = db.OpenTransaction()
	if err != nil {
		return err
	}
	defer commitOrDiscard(tx, &err)
	err = deleteString(tx, normalPath, user.String())
	return
}
