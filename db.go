package main

import (
	"github.com/syndtr/goleveldb/leveldb"
)

var db *leveldb.DB

// commitOrDismiss either commits or discard the transaction depending on the error presence.
// This function is supposed to be called in defer.
// Arguments of deferred functions are evaluated at the declaration line
// so we need to pass a pointer to error to get the real value later. It's a rare use case of pointer to interface.
func commitOrDismiss(tx *leveldb.Transaction, err *error) {
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
