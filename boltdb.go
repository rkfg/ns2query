package main

import (
	"bytes"
	"fmt"
	"strings"

	"go.etcd.io/bbolt"
)

var (
	bdb                 *bbolt.DB
	discordBucketName   = []byte("discord_to_steamid")
	lowercaseBucketName = []byte("lowercase_to_normalcase")
)

type usersBucket struct {
	*bbolt.Bucket
}

func newUsersBucket(tx *bbolt.Tx) usersBucket {
	return usersBucket{tx.Bucket(discordBucketName)}
}

func (b usersBucket) put(key string, value uint32) error {
	return b.Put([]byte(key), uint32ToBytes(value))
}

func (b usersBucket) get(key string) (uint32, error) {
	val := b.Get([]byte(key))
	if val == nil {
		return 0, fmt.Errorf("not found")
	}
	return uint32FromBytes(val), nil
}

func (b usersBucket) del(key string) error {
	return b.Delete([]byte(key))
}

type lowercaseBucket struct {
	*bbolt.Bucket
}

func newLowercaseBucket(tx *bbolt.Tx) lowercaseBucket {
	return lowercaseBucket{tx.Bucket(lowercaseBucketName)}
}

func (l lowercaseBucket) put(name string) error {
	return l.Put([]byte(strings.ToLower(name)), []byte(name))
}

func (l lowercaseBucket) get(key string) string {
	return string(l.Get([]byte(key)))
}

func (l lowercaseBucket) findFirstString(prefix string) (result string, err error) {
	c := l.Cursor()
	k, v := c.Seek([]byte(prefix))
	if k == nil || !bytes.HasPrefix(k, []byte(prefix)) {
		return "", fmt.Errorf("not found")
	}
	return string(v), nil
}

func (l lowercaseBucket) del(key string) error {
	return l.Delete([]byte(key))
}

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
