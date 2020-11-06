package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

const cmdPrefix = "-"

func handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if s.State.User.ID == m.Author.ID {
		return
	}
	msg := m.Message.Content
	if !strings.HasPrefix(msg, cmdPrefix) {
		return
	}
	msg = strings.TrimPrefix(msg, cmdPrefix)
	response := ""
	switch msg {
	case "status":
		for i := range config.Servers {
			srv := &config.Servers[i]
			response += fmt.Sprintf("%s [%s], skill: %d, players: %d\n", srv.Name, srv.currentMap, srv.avgSkill, len(srv.players))
		}
	case "help":
		response = "Commands:\n\t-status\t\tshow server maps, skills and player count"
	}
	if response != "" {
		s.ChannelMessageSend(m.ChannelID, response)
	}
}

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
	err = dg.Open()
	if err != nil {
		return err
	}
	defer dg.Close()
	if tr, ok := http.DefaultTransport.(*http.Transport); ok {
		log.Println("Adjusting TLS session cache")
		tr.TLSClientConfig.ClientSessionCache = tls.NewLRUClientSessionCache(100)
	}
	dg.AddHandler(handleCommand)
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
