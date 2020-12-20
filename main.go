package main

import (
	"log"

	"github.com/docopt/docopt-go"
)

func main() {
	usage := `Usage:
	ns2query [-c config]
	ns2query -u
	ns2query -h
	ns2query --convert

Options:
	-h --help    This help
	-c config    Use config file [default: config.json]
	-u           Update database
	--convert    Convert the database to BoltDB
`
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		log.Fatal("error parsing arguments:", err)
	}
	if err := loadConfig(opts["-c"].(string)); err != nil {
		log.Fatal("error loading config:", err)
	}
	if config.LevelDBPath != "" {
		if err := openDB(config.LevelDBPath); err != nil {
			log.Fatal("error opening LevelDB database:", err)
		}
		defer closeDB()
	}
	if config.BoltDBPath != "" {
		if err := openBoltDB(config.BoltDBPath); err != nil {
			log.Fatal("error opening BoltDB database:", err)
		}
		initBoltDB()
		defer closeBoltDB()
	}
	if update, err := opts.Bool("-u"); err == nil && update {
		if err := updateDB(); err != nil {
			log.Fatal("error updating db:", err)
		}
		log.Println("Database updated successfully.")
		return
	}
	if convert, _ := opts.Bool("--convert"); convert {
		if err := convertDB(); err != nil {
			log.Fatal("error converting db:", err)
		}
		log.Println("Database converted successfully.")
		return
	}
	err = bot()
	if err != nil {
		log.Fatal("error launching bot:", err)
	}
}
