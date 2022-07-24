package main

import (
	"log"

	"github.com/docopt/docopt-go"
	"go.etcd.io/bbolt"
	"rkfg.me/ns2query/db"
)

var (
	bdb *bbolt.DB
)

func main() {
	usage := `Usage:
	ns2query [-c config]
	ns2query --reindex
	ns2query -h

Options:
	-h --help    This help
	-c config    Use config file [default: config.json]
`
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		log.Fatal("error parsing arguments:", err)
	}
	if err := loadConfig(opts["-c"].(string)); err != nil {
		log.Fatal("error loading config:", err)
	}
	if config.BoltDBPath != "" {
		if bdb, err = db.OpenBoltDB(config.BoltDBPath); err != nil {
			log.Fatal("error opening BoltDB database:", err)
		}
		db.InitBoltDB(bdb)
		defer db.CloseBoltDB(bdb)
	}
	if b, _ := opts.Bool("--reindex"); b {
		db.Reindex(bdb)
	}
	err = bot()
	if err != nil {
		log.Fatal("error launching bot:", err)
	}
}
