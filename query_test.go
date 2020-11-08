package main

import (
	"log"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestMain(t *testing.M) {
	if err := loadConfigFilename("config_test.json"); err != nil {
		log.Fatal(err)
	}
	os.Exit(t.Run())
}

func notif(t *testing.T, srv *ns2server, expected string) {
	sendChan := make(chan string)
	q := make(chan bool)
	go func() {
		maybeNotify(srv, sendChan)
		close(q)
	}()
	select {
	case m := <-sendChan:
		if expected == "" {
			t.Errorf("unexpected message reported: '%s'", m)
		} else if m != expected {
			t.Errorf("expected message '%s' got '%s'", expected, m)
		}
	case <-q:
		if expected != "" {
			t.Errorf("expected '%s' got nothing", expected)
		}
	}
}

func fillPlayers(num int) []string {
	players := []string{}
	for i := 0; i < num; i++ {
		players = append(players, strconv.FormatInt(int64(i+1), 10))
	}
	return players
}

func passTime(srv *ns2server) {
	srv.lastStatePromotion = time.Now().Add(-time.Minute) // a minute passed
}

func TestNotification(t *testing.T) {
	srv := &ns2server{
		Name:              "Test",
		currentMap:        "test",
		maxStateToMessage: full,
		PlayerSlots:       20,
		SpecSlots:         6,
	}
	notif(t, srv, "")
	srv.players = fillPlayers(4)
	notif(t, srv, "Test [test] started seeding! Skill: 0. There are 4 players there currently: 1, 2, 3, 4")
	// test demotion without cooldown
	srv.players = fillPlayers(3)
	notif(t, srv, "")
	srv.players = fillPlayers(5)
	notif(t, srv, "") // no messages after quick demotion
	passTime(srv)
	notif(t, srv, "")
	srv.players = fillPlayers(13)
	notif(t, srv, "Test [test] is almost full! Skill: 0. There are 13 players there currently")
	srv.players = fillPlayers(21)
	notif(t, srv, "Test [test] is full but you can still make it! Skill: 0. There are 5 spectator slots available currently")
	srv.players = fillPlayers(26)
	notif(t, srv, "") // no message when full
	srv.players = fillPlayers(19)
	notif(t, srv, "") // some fluctuations
	passTime(srv)
	srv.players = fillPlayers(21)
	notif(t, srv, "") // still no messages
	passTime(srv)
	srv.players = fillPlayers(13)
	notif(t, srv, "") // until it's empty
	passTime(srv)
	srv.players = fillPlayers(3)
	notif(t, srv, "") // server became empty, seeding messages enabled again
	passTime(srv)
	srv.players = fillPlayers(7)
	notif(t, srv, "Test [test] started seeding! Skill: 0. There are 7 players there currently: 1, 2, 3, 4, 5, 6, 7")
	srv.players = fillPlayers(12)
	notif(t, srv, "Test [test] is almost full! Skill: 0. There are 12 players there currently")
	srv.players = fillPlayers(6)
	passTime(srv)
	notif(t, srv, "") // some players left, seeding again but no message
	srv.players = fillPlayers(12)
	notif(t, srv, "") // no duplicate message even after cooldown
	srv.players = fillPlayers(3)
	passTime(srv)
	notif(t, srv, "") // server became empty, seeding messages enabled again
	srv.players = fillPlayers(4)
	notif(t, srv, "Test [test] started seeding! Skill: 0. There are 4 players there currently: 1, 2, 3, 4")
	srv.players = fillPlayers(3)
	passTime(srv)
	notif(t, srv, "") // server became empty
	srv.players = fillPlayers(4)
	notif(t, srv, "Test [test] started seeding! Skill: 0. There are 4 players there currently: 1, 2, 3, 4") // server is seeding again
}
