package main

import (
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"go.etcd.io/bbolt"
)

func updateDB() (err error) {
	var tx *leveldb.Transaction
	tx, err = ldb.OpenTransaction()
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

func convertDB() (err error) {
	err = bdb.Update(func(t *bbolt.Tx) (err error) {
		users := newUsersBucket(t)
		lc := newLowercaseBucket(t)
		path := normalPath + "\x00"
		iter := ldb.NewIterator(util.BytesPrefix([]byte(path)), nil)
		defer iter.Release()
		for iter.Next() {
			name := strings.TrimPrefix(string(iter.Key()), path)
			users.put(name, uint32FromBytes(iter.Value()))
			lc.put(name)
		}
		return
	})
	return
}
