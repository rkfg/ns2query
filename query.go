package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/rumblefrog/go-a2s"
)

func maybeNotify(srv *ns2server, sendChan chan string) {
	playersCount := len(srv.players)
	newState := empty
	if playersCount < config.Seeding.Seeding {
		newState = empty
	} else if playersCount < config.Seeding.AlmostFull {
		newState = seedingstarted
	} else if playersCount < srv.PlayerSlots {
		newState = almostfull
	} else if playersCount < srv.PlayerSlots+srv.SpecSlots {
		newState = specsonly
	} else {
		newState = full
	}
	if newState > srv.serverState && newState <= srv.maxStateToMessage {
		srv.lastStatePromotion = time.Now()
		srv.serverState = newState
		if srv.lastStateAnnounced != newState {
			srv.lastStateAnnounced = newState
			switch newState {
			case seedingstarted:
				sendChan <- fmt.Sprintf("%s [%s] started seeding! Skill: %d. There are %d players there currently: %s",
					srv.Name, srv.currentMap, srv.avgSkill, len(srv.players), srv.playersString())
				srv.maxStateToMessage = specsonly
			case almostfull:
				sendChan <- fmt.Sprintf("%s [%s] is almost full! Skill: %d. There are %d players there currently",
					srv.Name, srv.currentMap, srv.avgSkill, len(srv.players))
			case specsonly:
				srv.maxStateToMessage = seedingstarted
				sendChan <- fmt.Sprintf("%s [%s] is full but you can still make it! Skill: %d. There are %d spectator slots available currently",
					srv.Name, srv.currentMap, srv.avgSkill, srv.PlayerSlots+srv.SpecSlots-len(srv.players))
			}
		}
	} else {
		if time.Since(srv.lastStatePromotion).Seconds() > float64(config.Seeding.Cooldown) {
			srv.serverState = newState
			if newState == empty {
				// if the server goes empty we should allow seeding messages again
				srv.lastStateAnnounced = empty
			}
		}
	}
}

func query(srv *ns2server, sendChan chan string) error {
	client, err := a2s.NewClient(srv.Address)
	if err != nil {
		return fmt.Errorf("error creating client: %s", err)
	}
	defer client.Close()
	srv.currentMap = "<unknown>"
	srv.avgSkill = 0
	srv.maxStateToMessage = full
	srv.lastStateAnnounced = empty
	for {
		info, err := client.QueryInfo()
		if err != nil {
			log.Printf("server info query error: %s", err)
		} else {
			srv.currentMap = info.Map
		}
		rules, err := client.QueryRules()
		if err != nil {
			log.Printf("rules query error: %s", err)
		} else {
			srv.avgSkill = 0
			avgSkillStr := rules.Rules["AverageSkill"]
			if avgSkillStr != "nan" && avgSkillStr != "" {
				avgSkill, err := strconv.ParseFloat(avgSkillStr, 32)
				if err != nil {
					log.Printf("error parsing avg skill: %s", err)
				} else {
					srv.avgSkill = int(avgSkill)
				}
			}
		}
		playersInfo, err := client.QueryPlayer()
		if err != nil {
			log.Printf("player query error: %s", err)
		} else {
			srv.players = srv.players[:0]
			for _, p := range playersInfo.Players {
				srv.players = append(srv.players, p.Name)
			}
			maybeNotify(srv, sendChan)
		}
		time.Sleep(config.QueryInterval * time.Second)
	}
}
