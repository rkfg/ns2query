package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rkfg/go-steamapi"
)

const cmdPrefix = "-"

type hive struct {
	Alias           string
	Skill           int
	SkillOffset     int `json:"skill_offset"`
	CommSkill       int
	CommSkillOffset int `json:"comm_skill_offset"`
}

var (
	vanityRegex  = regexp.MustCompile(`https://steamcommunity.com/id/([^/]*)/?`)
	profileRegex = regexp.MustCompile(`https://steamcommunity.com/profiles/(\d*)/?`)
)

func steamIDFromPlayer(player string) (uint32, error) {
	vanityName := vanityRegex.FindStringSubmatch(player)
	if vanityName != nil {
		player = vanityName[1]
	} else {
		profileID := profileRegex.FindStringSubmatch(player)
		if profileID != nil {
			player = profileID[1]
		}
	}
	steamid, err := steamapi.NewIdFromVanityUrl(player, config.SteamKey)
	if err != nil {
		id64, err := strconv.ParseUint(player, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("user/id %s not found", player)
		}
		steamid = steamapi.NewIdFrom64bit(id64)
	}
	id32 := steamid.As32Bit()
	return id32, nil
}

func getSkill(player string) (string, error) {
	id32, err := steamIDFromPlayer(player)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("http://hive2.ns2cdt.com/api/get/playerData/%d", id32)
	hiveResp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	var skill hive
	json.NewDecoder(hiveResp.Body).Decode(&skill)
	if skill.Alias == "" {
		return "", fmt.Errorf("user/id %s not found", player)
	}
	return fmt.Sprintf(
		`%s (ID: %d) skill breakdown:
Marine skill: %d (commander: %d)
Alien skill: %d (commander: %d)
`, skill.Alias, id32,
		skill.Skill+skill.SkillOffset, skill.CommSkill+skill.CommSkillOffset,
		skill.Skill-skill.SkillOffset, skill.CommSkill-skill.CommSkillOffset), nil
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
	response := ""
	if len(fields) > 0 {
		switch fields[0] {
		case "status":
			for i := range config.Servers {
				srv := &config.Servers[i]
				response += fmt.Sprintf("%s [%s], skill: %d, players: %d\n", srv.Name, srv.currentMap, srv.avgSkill, len(srv.players))
			}
		case "skill":
			if len(fields) < 2 {
				break
			}
			var err error
			if response, err = getSkill(fields[1]); err != nil {
				response = err.Error()
			}
		case "version":
			response = versionString()
		case "help":
			response =
				`Commands:
	-status				show server maps, skills and player count
	-skill player		show skill breakdown for player
	-version			show current bot version, build date and source code URL
	
If your Steam profile page URL looks like <https://steamcommunity.com/profiles/76561197960287930>, use 76561197960287930 as a -skill argument.
If it looks like <https://steamcommunity.com/id/gabelogannewell>, use gabelogannewell instead. Or just pass the entire URL, we don't judge!`
		}
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
