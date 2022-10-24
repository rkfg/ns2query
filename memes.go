package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.etcd.io/bbolt"
	"rkfg.me/ns2query/db"
)

var (
	ErrAlreadyAnnounced = fmt.Errorf("meme already announced")
)

func hasMeme(m *discordgo.Message) bool {
	return len(m.Attachments) > 0 || strings.Contains(m.Content, "https://") || strings.Contains(m.Content, "http://")
}

func chooseMemeOfTheDay(s *discordgo.Session, memeChannelID string, deadlineHour int, dayLen int) ([]*discordgo.Message, int, error) {
	notBefore := time.Now().UTC().Truncate(time.Hour*24).AddDate(0, 0, -2).Add(time.Hour * time.Duration(deadlineHour))
	notAfter := notBefore.Add(time.Hour * time.Duration(dayLen))
	log.Printf("OWL time from %s to %s", notBefore.Format(time.Stamp), notAfter.Format(time.Stamp))
	allMessages := []*discordgo.Message{}
	beforeID := ""
	for {
		messages, err := s.ChannelMessages(memeChannelID, 100, beforeID, "", "")
		if err != nil {
			return nil, 0, err
		}
		allMessages = append(allMessages, messages...)
		if len(allMessages) == 0 {
			return nil, 0, nil
		}
		if len(messages) == 0 || messages[len(messages)-1].Timestamp.Before(notBefore) {
			break
		}
		beforeID = messages[len(messages)-1].ID
	}
	maxUpvotes := 0
	winners := []*discordgo.Message{}
	lastWinnerID := ""
	err := bdb.View(func(tx *bbolt.Tx) error {
		memesBucket := db.NewMemesBucket(tx)
		status, err := memesBucket.GetValue(memeChannelID)
		if err != nil {
			return err
		}
		lastWinnerID = status.LastWinnerID
		return nil
	})
	if err != nil {
		log.Printf("Error looking for last winner ID: %s", err)
	} else {
		log.Printf("Last winner message ID: %s", lastWinnerID)
	}
	for _, m := range allMessages {
		if !hasMeme(m) || m.Timestamp.Before(notBefore) || m.Timestamp.After(notAfter) || m.ID == lastWinnerID {
			continue
		}
		url := m.Content
		if len(m.Attachments) > 0 {
			url = m.Attachments[0].URL
		}
		log.Printf("Considering message %s", url)
		for _, r := range m.Reactions {
			if r.Emoji.Name == "\U0001F44D" {
				if r.Count > maxUpvotes {
					maxUpvotes = r.Count
					winners = []*discordgo.Message{}
				}
				if r.Count >= maxUpvotes {
					winners = append(winners, m)
				}
			}
		}
	}
	return winners, maxUpvotes, nil
}

func announceMOTD(s *discordgo.Session, channelID string, deadlineHour int, dayLen int) error {
	if !config.Threads[channelID].Meme {
		return nil
	}
	winners, upvotes, err := chooseMemeOfTheDay(s, channelID, deadlineHour, dayLen)
	if err != nil {
		return err
	}
	if len(winners) == 0 {
		return nil
	}
	winner := winners[0]
	winnerURL := winner.Content
	if len(winner.Attachments) > 0 {
		winnerURL = winner.Attachments[0].URL
	}
	if len(winners) > 1 {
		winnerURL = fmt.Sprintf("%s (tied between %d best memes)", winnerURL, len(winners))
	}
	response := fmt.Sprintf("Meme of the day from <#%s> (%d upvotes): %s", channelID, upvotes, winnerURL)
	targetChannelID := config.Threads[channelID].AnnounceWinnerTo
	if targetChannelID == "" {
		s.ChannelMessageSend(channelID, response)
	} else {
		s.ChannelMessageSend(targetChannelID, response)
	}
	bdb.Update(func(tx *bbolt.Tx) error {
		memesBucket := db.NewMemesBucket(tx)
		return memesBucket.PutValue(channelID,
			db.MemeStatus{
				LastAnnouncementDay: time.Now().Day(),
				LastWinnerID:        winner.ID,
			})
	})
	return nil
}

func competition(s *discordgo.Session, channelID string, t thread) {
	if t.CompetitionLength == 0 {
		t.CompetitionLength = 24
	}
	log.Printf("Running meme competition in channel %s, announcing at %d:00 (UTC+0) to channel %s, "+
		"considering all memes since %d:00 (UTC+0) of the prior day for the next %d hours", channelID, t.CompetitionAnnouncement,
		t.AnnounceWinnerTo, t.CompetitionDeadline, t.CompetitionLength)
	for {
		now := time.Now().UTC()
		nextAnnouncement := now.Truncate(time.Hour * 24).Add(time.Hour * time.Duration(t.CompetitionAnnouncement))
		if now.Hour() >= t.CompetitionAnnouncement { // next announcement is tomorrow
			nextAnnouncement = nextAnnouncement.Add(time.Hour * 24)
		}
		if now.Hour() == t.CompetitionAnnouncement {
			err := bdb.View(func(tx *bbolt.Tx) error {
				memesBucket := db.NewMemesBucket(tx)
				status, err := memesBucket.GetValue(channelID)
				if err != nil {
					return err
				}
				if status.LastAnnouncementDay == now.Day() {
					return ErrAlreadyAnnounced
				}
				return nil
			})
			if err == ErrAlreadyAnnounced {
				log.Printf("Meme from channel %s has already been announced to %s",
					channelID, t.AnnounceWinnerTo)
			} else {
				if err != nil && err != db.ErrNotFound {
					log.Printf("Error querying meme announcement status: %s", err)
				} else {
					announceMOTD(s, channelID, t.CompetitionDeadline, t.CompetitionLength)
				}
			}
		}
		log.Printf("Next announcement time: %s, sleeping until then", nextAnnouncement.Format("02-01-2006 15:04:05"))
		time.Sleep(time.Until(nextAnnouncement))
	}
}

func startCompetitions(s *discordgo.Session) {
	for id, t := range config.Threads {
		if t.Meme && t.Competition {
			go competition(s, id, t)
		}
	}
}
