package db

import (
	"fmt"

	"go.etcd.io/bbolt"
)

var (
	discordBucketName   = []byte("discord_to_steamid")
	steamidBucketName   = []byte("steamid_to_discord")
	lowercaseBucketName = []byte("lowercase_to_normalcase")
	ErrNotFound         = fmt.Errorf("not found")
)

type UsersBucket struct {
	Bucket[string, uint32]
}

func NewUsersBucket(tx *bbolt.Tx) UsersBucket {
	return UsersBucket{
		Bucket[string, uint32]{
			tx.Bucket(discordBucketName),
			StringConverter{},
			U32Converter{},
		}}
}

type LowercaseBucket struct {
	Bucket[string, string]
}

func NewLowercaseBucket(tx *bbolt.Tx) LowercaseBucket {
	return LowercaseBucket{
		Bucket[string, string]{
			tx.Bucket(lowercaseBucketName),
			LowercaseStringConverter{},
			StringConverter{},
		}}
}

type SteamToDiscordBucket struct {
	Bucket[uint32, string]
}

func NewSteamToDiscordBucket(tx *bbolt.Tx) SteamToDiscordBucket {
	return SteamToDiscordBucket{
		Bucket[uint32, string]{
			tx.Bucket(steamidBucketName),
			U32Converter{},
			StringConverter{},
		}}
}
