package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	cmdPrefix       = "-"
	lowercasePrefix = "/lc/"
	discordPrefix   = "!"
)

type reaction struct {
	messageID string
	emojiID   string
}

type message struct {
	*discordgo.MessageSend
	*reaction
	channelID string
}

type currentServerStatus struct {
	ServerName  string
	Players     int
	TotalSlots  int
	PlayerSlots int
	SpecSlots   int
	FreeSlots   int
	Map         string
	Skill       int
}

var (
	sendChan         = make(chan message, 10)
	discordNameRegex = regexp.MustCompile(`^.*#\d{4}$`)
)

func parseFields(fields []string, author *discordgo.User, channelID string) (response *discordgo.MessageSend, err error) {
	var playerID uint32
	switch strings.ToLower(fields[0]) {
	case "status":
		for i := range config.Servers {
			msg := config.Servers[i].serverStatus()
			sendChan <- message{MessageSend: msg, channelID: channelID}
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
			playerID, err = bind(fields[1], author.String())
			if err != nil {
				return
			}
			return &discordgo.MessageSend{Content: fmt.Sprintf("User %s has been bound to player ID %d. You can use `-skill` without arguments now.",
				author.String(), playerID)}, nil
		}
		err = deleteBind(author.String())
		if err != nil {
			return
		}
		return &discordgo.MessageSend{Content: fmt.Sprintf("User %s has been unbound.", author.String())}, nil
	case "bindu":
		if role, ok := config.Users[author.ID]; !ok || role != "admin" {
			return nil, fmt.Errorf("insufficient privilege")
		}
		if len(fields) > 3 {
			return nil, fmt.Errorf("invalid arguments for `-bindu`")
		}
		if len(fields) < 2 {
			return nil, fmt.Errorf("not enough arguments for `-bindu`")
		}
		username := fields[1]
		if !discordNameRegex.MatchString(username) {
			return nil, fmt.Errorf("invalid Discord name, must be in the form of `name#3333`")
		}
		if len(fields) == 3 {
			playerID, err = bind(fields[2], username)
			if err != nil {
				return
			}
			return &discordgo.MessageSend{Content: fmt.Sprintf("User %s has been bound to player ID %d.",
				username, playerID)}, nil
		}
		err = deleteBind(username)
		if err != nil {
			return
		}
		return &discordgo.MessageSend{Content: fmt.Sprintf("User %s has been unbound.", username)}, nil
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
					Value: "bind your Discord accound to the specified player so you can use `-skill` " +
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

func processThreadMessage(s *discordgo.Session, m *discordgo.MessageCreate, t thread) {
	if !t.Meme {
		return
	}
	if !hasMeme(m.Message) {
		return
	}
	sendChan <- message{channelID: m.ChannelID, reaction: &reaction{messageID: m.ID, emojiID: "\U0001F44D"}}
}

func handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if s.State.User.ID == m.Author.ID {
		return
	}
	msg := m.Message.Content
	if t, ok := config.Threads[m.ChannelID]; ok {
		processThreadMessage(s, m, t)
	}
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
			sendChan <- message{MessageSend: response, channelID: m.ChannelID}
		}
	}
}

func statusUpdate(restartChan chan struct{}, s *discordgo.Session) {
	for {
		status := &bytes.Buffer{}
		for _, s := range config.Servers {
			if s.statusTemplate != nil {
				if status.Len() > 0 {
					status.WriteString(" | ")
				}
				cs := currentServerStatus{
					ServerName:  s.Name,
					Players:     len(s.players),
					PlayerSlots: s.PlayerSlots,
					SpecSlots:   s.SpecSlots,
					FreeSlots:   s.SpecSlots + s.PlayerSlots - len(s.players),
					TotalSlots:  s.SpecSlots + s.PlayerSlots,
					Map:         s.currentMap,
					Skill:       s.avgSkill,
				}
				if cs.Players > 0 {
					if err := s.statusTemplate.Execute(status, cs); err != nil {
						log.Printf("Error executing template for server %s: %s", s.Name, err)
					}
				}
			}
		}
		statusStr := "Natural Selection 2"
		if status.Len() > 0 {
			statusStr = status.String()
		}
		s.UpdateStatusComplex(discordgo.UpdateStatusData{
			Status: "online",
			Activities: []*discordgo.Activity{{
				Type: discordgo.ActivityTypeGame,
				Name: statusStr,
			}},
		})
		select {
		case <-time.After(config.QueryInterval):
		case <-restartChan:
			log.Print("Restart request received, stopping status updater")
			return
		}
	}
}

func sendMsg(c chan message, s *discordgo.Session) {
	for msg := range c {
		channelID := msg.channelID
		if channelID == "" {
			channelID = config.ChannelID
		}
		if msg.MessageSend != nil {
			s.ChannelMessageSendComplex(channelID, msg.MessageSend)
		}
		if msg.reaction != nil {
			s.MessageReactionAdd(channelID, msg.messageID, msg.emojiID)
		}
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
	restartChan := make(chan struct{})
	for i := range config.Servers {
		config.Servers[i].restartChan = restartChan
		if config.Servers[i].StatusTemplate != "" {
			t, err := template.New(config.Servers[i].Address + "/template").Parse(config.Servers[i].StatusTemplate)
			if err != nil {
				log.Printf("Error in status template '%s': %s", config.Servers[i].StatusTemplate, err)
			} else {
				config.Servers[i].statusTemplate = t
			}
		}
		if config.Servers[i].QueryIDInterval < 1 {
			config.Servers[i].QueryIDInterval = config.QueryInterval
		} else {
			config.Servers[i].QueryIDInterval *= time.Second
		}
		if config.Servers[i].AnnounceDelay < 1 {
			config.Servers[i].AnnounceDelay = time.Minute * 5
		} else {
			config.Servers[i].AnnounceDelay *= time.Second
		}
		if config.Servers[i].RegularTimeout < 1 {
			config.Servers[i].RegularTimeout = time.Hour
		} else {
			config.Servers[i].RegularTimeout *= time.Second
		}
		config.Servers[i].regularTimeouts = make(map[uint32]*time.Time)
		config.Servers[i].query()
	}
	for tid := range config.Threads {
		if config.Threads[tid].Join {
			if err := dg.ThreadJoin(tid); err != nil {
				log.Printf("Error joining thread %s: %s", tid, err)
			}
		}
	}
	go statusUpdate(restartChan, dg)
	startCompetitions(dg)
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	select {
	case <-sc:
	case <-restartChan:
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
