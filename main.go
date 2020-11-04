package main

import (
	"log"
)

func main() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}
	err := bot()
	if err != nil {
		log.Fatal(err)
	}
}
