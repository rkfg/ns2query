package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

func sendMsg(c chan string, s *discordgo.Session) {
	for {
		s.ChannelMessageSend(config.ChannelID, <-c)
		time.Sleep(time.Second)
	}
}

func bot() error {
	dg, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		return err
	}
	defer dg.Close()
	err = dg.Open()
	if err != nil {
		return err
	}
	sendChan := make(chan string, 10)
	go sendMsg(sendChan, dg)
	for i := range config.Servers {
		go query(&config.Servers[i], sendChan)
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	return nil
}
