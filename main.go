package main

import (
	"log"

	"github.com/docopt/docopt-go"
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
		if err := openBoltDB(config.BoltDBPath); err != nil {
			log.Fatal("error opening BoltDB database:", err)
		}
		initBoltDB()
		defer closeBoltDB()
	}
	if b, _ := opts.Bool("--reindex"); b {
		reindex()
	}
	err = bot()
	if err != nil {
		log.Fatal("error launching bot:", err)
	}
}
