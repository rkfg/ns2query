package main

import (
	"encoding/binary"
	"log"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

const pathSeparator = "\x00"

var ldb *leveldb.DB

// commitOrDiscard either commits or discard the transaction depending on the error presence.
// This function is supposed to be called in defer.
// Arguments of deferred functions are evaluated at the declaration line
// so we need to pass a pointer to error to get the real value later. It's a rare use case of pointer to interface.
func commitOrDiscard(tx *leveldb.Transaction, err *error) {
	if *err != nil {
		tx.Discard()
	} else {
		tx.Commit()
	}
}

func openDB(dbPath string) (err error) {
	ldb, err = leveldb.OpenFile(dbPath, nil)
	if e, corrupted := err.(*errors.ErrCorrupted); corrupted {
		log.Printf("WARNING: database corruption: %s. Attempting to recover...", e)
		ldb, err = leveldb.RecoverFile(dbPath, nil)
	}
	return
}

func closeDB() error {
	return ldb.Close()
}

func makePath(path ...string) string {
	return strings.Join(path, pathSeparator)
}

func pathKey(path string, key string) []byte {
	if path == "" {
		return []byte(key)
	}
	return []byte(path + pathSeparator + key)
}

func deleteString(tx *leveldb.Transaction, path string, key string) error {
	return tx.Delete(pathKey(path, key), nil)
}

func putString(tx *leveldb.Transaction, path string, key string, value string) error {
	return tx.Put(pathKey(path, key), []byte(value), nil)
}

func uint32FromBytes(val []byte) uint32 {
	return binary.LittleEndian.Uint32(val)
}

func uint32ToBytes(val uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	return buf[:]
}

func putUInt32(tx *leveldb.Transaction, path string, key string, value uint32) error {
	return tx.Put(pathKey(path, key), uint32ToBytes(value), nil)
}

func putLowercaseIndex(tx *leveldb.Transaction, username string) error {
	return putString(tx, lowercasePath, strings.ToLower(username), username)
}
