package main

import (
	"encoding/binary"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/syndtr/goleveldb/leveldb"
)

var db *leveldb.DB

func openDB(dbPath string) (err error) {
	db, err = leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return
	}
	return nil
}

func closeDB() error {
	return db.Close()
}

func putBind(playerID uint32, discordUser *discordgo.User) error {
	var key [4]byte
	binary.LittleEndian.PutUint32(key[:], playerID)
	return db.Put([]byte(discordUser.String()), key[:], nil)
}

func getBind(user *discordgo.User) (playerID uint32, err error) {
	key := []byte(user.String())
	has, err := db.Has(key, nil)
	if !has {
		return 0, fmt.Errorf("player %s isn't in the database. Use `-bind <Steam ID>` to register", user.String())
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
