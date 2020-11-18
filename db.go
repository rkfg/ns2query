package main

import (
	"encoding/binary"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var db *leveldb.DB

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
	db, err = leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return
	}
	return nil
}

func closeDB() error {
	return db.Close()
}

func pathToString(path []string, key string) []byte {
	return []byte(strings.Join(append(path, key), "\x00"))
}

func findFirstString(path []string, prefix string) (result string, err error) {
	iter := db.NewIterator(util.BytesPrefix(pathToString(path, prefix)), nil)
	defer iter.Release()
	if ok := iter.Next(); ok {
		return string(iter.Value()), nil
	}
	return "", leveldb.ErrNotFound
}

func getUInt32(path []string, key string) (uint32, error) {
	val, err := db.Get(pathToString(path, key), nil)
	if err != nil {
		return 0, err
	}
	return uint32FromBytes(val), nil
}

func deleteString(tx *leveldb.Transaction, path []string, key string) error {
	return tx.Delete(pathToString(path, key), nil)
}

func putString(tx *leveldb.Transaction, path []string, key string, value string) error {
	return tx.Put(pathToString(path, key), []byte(value), nil)
}

func uint32FromBytes(val []byte) uint32 {
	return binary.LittleEndian.Uint32(val)
}

func uint32ToBytes(val uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	return buf[:]
}

func putUInt32(tx *leveldb.Transaction, path []string, key string, value uint32) error {
	return tx.Put(pathToString(path, key), uint32ToBytes(value), nil)
}
