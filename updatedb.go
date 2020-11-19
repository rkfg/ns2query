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
	defer commitOrDiscard(tx, &err)
	iter := tx.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		name := string(iter.Key())
		if strings.HasPrefix(name, lowercasePrefix) {
			deleteString(tx, "", name)
		} else {
			if err = putLowercaseIndex(tx, name); err != nil {
				return
			}
			if err = putUInt32(tx, normalPath, name, uint32FromBytes(iter.Value())); err != nil {
				return
			}
			deleteString(tx, "", name)
		}
	}
	return
}
