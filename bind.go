package main

import (
	"encoding/binary"
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
	defer commitOrDismiss(tx, &err)
	var key [4]byte
	binary.LittleEndian.PutUint32(key[:], playerID)
	if err = tx.Put([]byte(discordUser.String()), key[:], nil); err != nil {
		return
	}
	if err = putLowercaseIndex(tx, discordUser.String()); err != nil {
		return
	}
	return
}

func putLowercaseIndex(tx *leveldb.Transaction, username string) error {
	return tx.Put([]byte(lowercasePrefix+strings.ToLower(username)), []byte(username), nil)
}

func getBind(username string) (playerID uint32, err error) {
	key := []byte(username)
	has, err := db.Has(key, nil)
	if !has {
		return 0, fmt.Errorf("player %s isn't in the database. Use `-bind <Steam ID>` to register", username)
	}
	if err != nil {
		return 0, err
	}
	val, err := db.Get(key, nil)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(val), nil
}

func deleteBind(user *discordgo.User) error {
	return db.Delete([]byte(user.String()), nil)
}
