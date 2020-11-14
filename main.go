package main

import (
	"log"
)

func main() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}
	if err := openDB(config.DBPath); err != nil {
		log.Fatal(err)
	}
	defer closeDB()
	err := bot()
	if err != nil {
		log.Fatal(err)
	}
}
