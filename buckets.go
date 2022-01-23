package main

import (
	"bytes"
	"fmt"
	"strings"

	"go.etcd.io/bbolt"
)

var (
	discordBucketName   = []byte("discord_to_steamid")
	steamidBucketName   = []byte("steamid_to_discord")
	lowercaseBucketName = []byte("lowercase_to_normalcase")
	NotFoundError       = fmt.Errorf("not found")
)

type bucket struct {
	*bbolt.Bucket
}

func (b bucket) del(key string) error {
	return b.Delete([]byte(key))
}

type usersBucket struct {
	bucket
}

func newUsersBucket(tx *bbolt.Tx) usersBucket {
	return usersBucket{bucket{tx.Bucket(discordBucketName)}}
}

func (b usersBucket) put(key string, value uint32) error {
	return b.Put([]byte(key), uint32ToBytes(value))
}

func (b usersBucket) get(key string) (uint32, error) {
	val := b.Get([]byte(key))
	if val == nil {
		return 0, NotFoundError
	}
	return uint32FromBytes(val), nil
}

type lowercaseBucket struct {
	bucket
}

func newLowercaseBucket(tx *bbolt.Tx) lowercaseBucket {
	return lowercaseBucket{bucket{tx.Bucket(lowercaseBucketName)}}
}

func (l lowercaseBucket) put(name string) error {
	return l.Put([]byte(strings.ToLower(name)), []byte(name))
}

func (l lowercaseBucket) findFirstString(prefix string) (result string, err error) {
	c := l.Cursor()
	k, v := c.Seek([]byte(prefix))
	if k == nil || !bytes.HasPrefix(k, []byte(prefix)) {
		return "", NotFoundError
	}
	return string(v), nil
}

type steamToDiscordBucket struct {
	bucket
}

func newSteamToDiscordBucket(tx *bbolt.Tx) steamToDiscordBucket {
	return steamToDiscordBucket{bucket{tx.Bucket(steamidBucketName)}}
}

func (s steamToDiscordBucket) put(steamid uint32, name string) error {
	return s.Put(uint32ToBytes(steamid), []byte(name))
}

func (s steamToDiscordBucket) get(steamid uint32) (name string, err error) {
	val := s.Get(uint32ToBytes(steamid))
	if val == nil {
		return "", NotFoundError
	}
	return string(val), nil
}

func (s steamToDiscordBucket) del(steamid uint32) error {
	return s.Delete(uint32ToBytes(steamid))
}