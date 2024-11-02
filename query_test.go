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
	wait := time.Second
	if expected == "" {
		wait = time.Millisecond * 100
	}
	q := time.After(wait)
	go func() {
		srv.maybeNotify()
	}()
	select {
	case m := <-sendChan:
		msg := m.Embed.Description
		if expected == "" {
			t.Errorf("unexpected message reported: '%s'", msg)
		} else if msg != expected {
			t.Errorf("expected message '%s' got '%s'", expected, msg)
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
	notif(t, srv, "Seeding started! Players on the server: 1, 2, 3, 4")
	// test demotion without cooldown
	srv.players = fillPlayers(3)
	notif(t, srv, "")
	srv.players = fillPlayers(5)
	notif(t, srv, "") // no messages after quick demotion
	passTime(srv)
	notif(t, srv, "")
	srv.players = fillPlayers(13)
	notif(t, srv, "Server is almost full!")
	srv.players = fillPlayers(21)
	notif(t, srv, "Server is full but you can still make it!")
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
	notif(t, srv, "Seeding started! Players on the server: 1, 2, 3, 4, 5, 6, 7")
	srv.players = fillPlayers(12)
	notif(t, srv, "Server is almost full!")
	srv.players = fillPlayers(6)
	passTime(srv)
	notif(t, srv, "") // some players left, seeding again but no message
	srv.players = fillPlayers(12)
	notif(t, srv, "") // no duplicate message even after cooldown
	srv.players = fillPlayers(3)
	passTime(srv)
	notif(t, srv, "") // server became empty, seeding messages enabled again
	srv.players = fillPlayers(4)
	notif(t, srv, "Seeding started! Players on the server: 1, 2, 3, 4")
	srv.players = fillPlayers(3)
	passTime(srv)
	notif(t, srv, "") // server became empty
	srv.players = fillPlayers(4)
	notif(t, srv, "Seeding started! Players on the server: 1, 2, 3, 4") // server is seeding again
	config.Seeding.NotifyEmpty = true
	srv.players = fillPlayers(12)
	notif(t, srv, "Server is almost full!")
	srv.players = fillPlayers(21)
	notif(t, srv, "Server is full but you can still make it!")
	s := time.Now().Add(-time.Minute)
	srv.sessionStart = &s
	passTime(srv)
	srv.players = fillPlayers(3)
	notif(t, srv, "Server is now empty. Session time: 1m0s")
}
