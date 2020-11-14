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

func playerIDFromPlayer(player string) (uint32, error) {
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

func getSkill(playerID uint32) (string, error) {
	url := fmt.Sprintf("http://hive2.ns2cdt.com/api/get/playerData/%d", playerID)
	hiveResp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	var skill hive
	json.NewDecoder(hiveResp.Body).Decode(&skill)
	if skill.Alias == "" {
		return "", fmt.Errorf("player id %d isn't present on Hive", playerID)
	}
	return fmt.Sprintf(
		`%s (ID: %d) skill breakdown:
Marine skill: %d (commander: %d)
Alien skill: %d (commander: %d)
`, skill.Alias, playerID,
		skill.Skill+skill.SkillOffset, skill.CommSkill+skill.CommSkillOffset,
		skill.Skill-skill.SkillOffset, skill.CommSkill-skill.CommSkillOffset), nil
}

func parseFields(fields []string, author *discordgo.User) (response string, err error) {
	var playerID uint32
	switch fields[0] {
	case "status":
		for i := range config.Servers {
			srv := &config.Servers[i]
			response += fmt.Sprintf("%s [%s], skill: %d, players: %d\n", srv.Name, srv.currentMap, srv.avgSkill, len(srv.players))
		}
	case "skill":
		switch len(fields) {
		case 1:
			playerID, err = getBind(author)
			if err != nil {
				return
			}
		case 2:
			playerID, err = playerIDFromPlayer(fields[1])
			if err != nil {
				return
			}
		default:
			return "", fmt.Errorf("invalid argument for `-skill`")
		}
		if response, err = getSkill(playerID); err != nil {
			return
		}
	case "bind":
		if len(fields) > 2 {
			return "", fmt.Errorf("invalid argument for `-bind`")
		}
		if len(fields) == 2 {
			playerID, err = bind(fields[1], author)
			if err != nil {
				return
			}
			return fmt.Sprintf("User %s has been bound to player ID %d. You can use `-skill` without arguments now.",
				author.String(), playerID), nil
		}
		err = deleteBind(author)
		if err != nil {
			return
		}
		return fmt.Sprintf("User %s has been unbound.", author.String()), nil
	case "version":
		return versionString(), nil
	case "help":
		return `Commands:
	-status				show server maps, skills and player count
	-skill [player]		show skill breakdown for player, the argument can be omitted if the player is bound
	-bind [player]		bind your Discord accound to the specified player so you can use ` + "`-skill`" + ` without argument. Use ` + "`-bind`" + ` without argument to unbind yourself.
	-version			show current bot version, build date and source code URL
	
If your Steam profile page URL looks like <https://steamcommunity.com/profiles/76561197960287930>, use 76561197960287930 as a -skill argument.
If it looks like <https://steamcommunity.com/id/gabelogannewell>, use gabelogannewell instead. Or just pass the entire URL, we don't judge!`,
			nil
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
		response, err := parseFields(fields, m.Author)
		if err != nil {
			response = "Error: " + err.Error()
		}
		if response != "" {
			s.ChannelMessageSend(m.ChannelID, response)
		}
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
