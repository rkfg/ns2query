package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	cmdPrefix       = "-"
	lowercasePrefix = "/lc/"
	discordPrefix   = "!"
)

type message struct {
	*discordgo.MessageSend
	channelID string
}

var (
	sendChan = make(chan message, 10)
)

func parseFields(fields []string, author *discordgo.User, channelID string) (response *discordgo.MessageSend, err error) {
	var playerID uint32
	switch fields[0] {
	case "status":
		for i := range config.Servers {
			msg := serverStatus(config.Servers[i])
			sendChan <- message{msg, channelID}
		}
	case "skill":
		if len(fields) == 1 {
			playerID, err = getBind(author.String())
		} else {
			if strings.HasPrefix(fields[1], discordPrefix) {
				playerID, err = playerIDFromDiscordName(
					strings.TrimPrefix(strings.ToLower(strings.Join(fields[1:], " ")), discordPrefix))
			} else {
				playerID, err = playerIDFromSteamID(fields[1])
			}
		}
		if err != nil {
			return
		}
		if response, err = getSkill(playerID); err != nil {
			return
		}
	case "bind":
		if len(fields) > 2 {
			return nil, fmt.Errorf("invalid argument for `-bind`")
		}
		if len(fields) == 2 {
			playerID, err = bind(fields[1], author)
			if err != nil {
				return
			}
			return &discordgo.MessageSend{Content: fmt.Sprintf("User %s has been bound to player ID %d. You can use `-skill` without arguments now.",
				author.String(), playerID)}, nil
		}
		err = deleteBind(author)
		if err != nil {
			return
		}
		return &discordgo.MessageSend{Content: fmt.Sprintf("User %s has been unbound.", author.String())}, nil
	case "version":
		return versionEmbed(), nil
	case "help":
		return &discordgo.MessageSend{Embed: &discordgo.MessageEmbed{Title: "Commands",
			Description: "Use your Steam profile page URL or its last part as a [Steam ID] argument.",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "-status",
					Value: "show server maps, skills and player count.",
				},
				{
					Name: "-skill [Steam ID]",
					Value: "show skill breakdown for player, the argument can be omitted if the player is bound. Use `!discordname` " +
						"argument to query other registered players; no need to type the whole name, several characters should be enough.",
				},
				{
					Name: "-bind [Steam ID]",
					Value: "bind your Discord accound to the specified player so you can use `-skill`" +
						"without argument. Use `-bind` without argument to unbind yourself.",
				},
				{
					Name:  "-version",
					Value: "show current bot version, build date and source code URL.",
				},
			},
		}}, nil
	}
	return
}

func handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if s.State.User.ID == m.Author.ID {
		return
	}
	msg := m.Message.Content
	if !strings.HasPrefix(msg, cmdPrefix) {
		return
	}
	msg = strings.TrimPrefix(msg, cmdPrefix)
	fields := strings.Fields(msg)
	if len(fields) > 0 {
		response, err := parseFields(fields, m.Author, m.ChannelID)
		if err != nil {
			response = &discordgo.MessageSend{Content: "Error: " + err.Error()}
		}
		if response != nil {
			sendChan <- message{response, m.ChannelID}
		}
	}
}

func sendMsg(c chan message, s *discordgo.Session) {
	for {
		msg := <-c
		channelID := msg.channelID
		if channelID == "" {
			channelID = config.ChannelID
		}
		s.ChannelMessageSendComplex(channelID, msg.MessageSend)
		time.Sleep(time.Second)
	}
}

func tryConnect(dg *discordgo.Session) (err error) {
	for {
		err = dg.Open()
		if err == nil {
			return
		}
		log.Printf("Error connecting: %s, retrying...", err)
		time.Sleep(5 * time.Second)
	}
}

func bot() (err error) {
	if tr, ok := http.DefaultTransport.(*http.Transport); ok {
		log.Println("Adjusting TLS session cache and handshake timeout")
		tr.TLSClientConfig = &tls.Config{ClientSessionCache: tls.NewLRUClientSessionCache(100)}
		tr.TLSHandshakeTimeout = 2 * time.Minute
	}
	var dg *discordgo.Session
	dg, err = discordgo.New("Bot " + config.Token)
	if err != nil {
		return
	}
	if err = tryConnect(dg); err != nil {
		return
	}
	defer dg.Close()
	dg.AddHandler(handleCommand)
	go sendMsg(sendChan, dg)
	restartChan := make(chan bool)
	for i := range config.Servers {
		config.Servers[i].restartChan = restartChan
		go query(config.Servers[i])
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	select {
	case <-sc:
	case <-restartChan:
		ldb.Close()
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		log.Println("Restarting myself...")
		err := cmd.Start()
		if err != nil {
			log.Println("Error restarting myself:", err)
		}
		log.Printf("Restart issued, pid: %d", cmd.Process.Pid)
	}
	return nil
}
