package main

import (
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
)

func updateDB() (err error) {
	var tx *leveldb.Transaction
	tx, err = db.OpenTransaction()
	if err != nil {
		return
	}
	defer commitOrDismiss(tx, &err)
	iter := tx.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		name := string(iter.Key())
		if !strings.HasPrefix(name, lowercasePrefix) {
			if err = putLowercaseIndex(tx, name); err != nil {
				return
			}
		}
	}
	return
}
