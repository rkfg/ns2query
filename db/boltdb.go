package db

import (
	"go.etcd.io/bbolt"
)

func OpenBoltDB(dbPath string) (bdb *bbolt.DB, err error) {
	bdb, err = bbolt.Open(dbPath, 0600, nil)
	return
}

func InitBoltDB(bdb *bbolt.DB) {
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

func CloseBoltDB(bdb *bbolt.DB) error {
	return bdb.Close()
}

func Reindex(bdb *bbolt.DB) {
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
