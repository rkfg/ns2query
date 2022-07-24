package main

import (
	"fmt"
	"strings"

	"go.etcd.io/bbolt"
	"rkfg.me/ns2query/db"
)

func bind(player string, name string) (id uint32, err error) {
	id, err = playerIDFromSteamID(player)
	if err != nil {
		return
	}
	err = putBind(id, name)
	return
}

func putBind(playerID uint32, name string) (err error) {
	return bdb.Update(func(t *bbolt.Tx) (err error) {
		err = db.NewUsersBucket(t).PutValue(name, playerID)
		if err != nil {
			return
		}
		steamBucket := db.NewSteamToDiscordBucket(t)
		err = steamBucket.PutValue(playerID, name)
		if err != nil {
			return
		}
		return db.NewLowercaseBucket(t).PutValue(name, name)
	})
}

func getBind(username string) (playerID uint32, err error) {
	err = bdb.View(func(t *bbolt.Tx) (err error) {
		playerID, err = db.NewUsersBucket(t).GetValue(username)
		return
	})
	if err != nil {
		return 0, fmt.Errorf("player %s isn't in the database. Use `-bind <Steam ID>` to register", username)
	}
	return
}

func deleteBind(name string) (err error) {
	return bdb.Update(func(t *bbolt.Tx) (err error) {
		userBucket := db.NewUsersBucket(t)
		steamBucket := db.NewSteamToDiscordBucket(t)
		steamId, _ := userBucket.GetValue(name)
		err = userBucket.DeleteValue(name)
		if err != nil {
			return
		}
		if steamId > 0 {
			err = steamBucket.DeleteValue(steamId)
			if err != nil {
				return
			}
		}
		err = db.NewLowercaseBucket(t).DeleteValue(strings.ToLower(name))
		return
	})
}
