package main

import (
	"encoding/binary"

	"go.etcd.io/bbolt"
)

var (
	bdb                 *bbolt.DB
)


func openBoltDB(dbPath string) (err error) {
	bdb, err = bbolt.Open(dbPath, 0600, nil)
	return
}

func initBoltDB() {
	err := bdb.Update(func(t *bbolt.Tx) error {
		_, err := t.CreateBucketIfNotExists(discordBucketName)
		if err != nil {
			return err
		}
		_, err = t.CreateBucketIfNotExists(steamidBucketName)
		if err != nil {
			return err
		}
		_, err = t.CreateBucketIfNotExists(lowercaseBucketName)
		return err
	})
	if err != nil {
		panic(err)
	}
}

func closeBoltDB() error {
	return bdb.Close()
}

func uint32FromBytes(val []byte) uint32 {
	return binary.LittleEndian.Uint32(val)
}

func uint32ToBytes(val uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	return buf[:]
}

func uint64ToBytes(val uint64) []byte {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], val)
	return buf[:]
}

func reindex() {
	bdb.Update(func(t *bbolt.Tx) error {
		discord := t.Bucket(discordBucketName)
		steam := t.Bucket(steamidBucketName)
		discord.ForEach(func(k, v []byte) error {
			steam.Put(v, k) // build reverse index
			return nil
		})
		return nil
	})
}
