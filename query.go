package main

import (
	"fmt"
	"log"
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
		switch newState {
		case seedingstarted:
			sendChan <- fmt.Sprintf("%s started seeding! There are %d players there currently: %s",
				srv.Name, len(srv.players), srv.playersString())
			srv.maxStateToMessage = specsonly
		case almostfull:
			sendChan <- fmt.Sprintf("%s is almost full! There are %d players there currently",
				srv.Name, len(srv.players))
		case specsonly:
			srv.maxStateToMessage = seedingstarted
			sendChan <- fmt.Sprintf("%s is full but you can still make it! There are %d spectator slots available currently",
				srv.Name, srv.PlayerSlots+srv.SpecSlots-len(srv.players))
		}
	} else {
		if time.Since(srv.lastStatePromotion).Seconds() > float64(config.Seeding.Cooldown) {
			srv.serverState = newState
		}
	}
}

func query(srv *ns2server, sendChan chan string) error {
	client, err := a2s.NewClient(srv.Address)
	if err != nil {
		return fmt.Errorf("error creating client: %s", err)
	}
	defer client.Close()
	for {
		info, err := client.QueryPlayer()
		if err != nil {
			log.Printf("query error: %s", err)
		} else {
			srv.players = srv.players[:0]
			for _, p := range info.Players {
				srv.players = append(srv.players, p.Name)
			}
			maybeNotify(srv, sendChan)
		}
		time.Sleep(config.QueryInterval * time.Second)
	}
}
