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

Options:
	-h --help    This help
	-c config    Use config file [default: config.json]
	-u           Update database
`
	opts, err := docopt.ParseDoc(usage)
	if err != nil {
		log.Fatal(err)
	}
	if err := loadConfig(opts["-c"].(string)); err != nil {
		log.Fatal(err)
	}
	if err := openDB(config.DBPath); err != nil {
		log.Fatal(err)
	}
	defer closeDB()
	if update, err := opts.Bool("-u"); err == nil && update {
		if err := updateDB(); err != nil {
			log.Fatal(err)
		}
		return
	}
	err = bot()
	if err != nil {
		log.Fatal(err)
	}
}
