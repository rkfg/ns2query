package main

import (
	"encoding/gob"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/bwmarrin/discordgo"
	"github.com/rivo/duplo"
)

const (
	imagedbFilename = "imagedb.bin"
	urlsFilename    = "urls.bin"
	maxBodyLength   = 20 * 1024 * 1024
)

type msgUrls struct {
	Message *discordgo.Message
	Urls    []string
}

func formatMessageLink(m *discordgo.Message) string {
	return fmt.Sprintf("https://discord.com/channels/%s/%s/%s", m.GuildID, m.ChannelID, m.ID)
}

func startReposts(urlChan <-chan msgUrls, sendChan chan<- message) {
	knownUrls := map[string]string{}
	f, err := os.Open(urlsFilename)
	if err == nil {
		err = gob.NewDecoder(f).Decode(&knownUrls)
		if err != nil {
			log.Printf("Error loading url store: %s", err)
		}
		f.Close()
	} else {
		log.Printf("Error opening url store: %s", err)
	}
	d := duplo.New()
	idb, err := os.ReadFile(imagedbFilename)
	if err == nil {
		err = d.GobDecode(idb)
		if err == nil {
			log.Printf("Image db %s loaded", imagedbFilename)
		} else {
			log.Printf("Error loading %s contents: %s", imagedbFilename, err)
		}
	} else {
		log.Printf("Error opening %s, creating empty storage: %s", imagedbFilename, err)
	}
	c := http.Client{Timeout: 5 * time.Second}
	for mu := range urlChan {
		msg := ""
		imageAdded := false
		for _, u := range mu.Urls {
			if msgLink, ok := knownUrls[u]; ok {
				msg += fmt.Sprintf(" %s", msgLink)
				continue
			}
			knownUrls[u] = formatMessageLink(mu.Message)
			resp, err := c.Get(u)
			if err != nil {
				log.Printf("Error getting image %s: %s", u, err)
				continue
			}
			if resp.ContentLength > maxBodyLength {
				log.Printf("Response too big: %d bytes, skipping", resp.ContentLength)
			}
			img, _, err := image.Decode(io.LimitReader(resp.Body, maxBodyLength))
			if err != nil {
				log.Printf("Error reading image %s body: %s", u, err)
				continue
			}
			h, _ := duplo.CreateHash(img)
			matches := d.Query(h)
			sort.Sort(matches)
			matchFound := false
			for _, m := range matches {
				if m.Score < 0 {
					msg += fmt.Sprintf(" %s", m.ID)
					matchFound = true
				}
			}
			if !matchFound {
				d.Add(formatMessageLink(mu.Message), h)
				imageAdded = true
			}
		}
		if len(msg) > 0 {
			msg = "Potential repost of:" + msg
			sendChan <- message{
				channelID: mu.Message.ChannelID, MessageSend: &discordgo.MessageSend{
					Content: msg,
				}}
		}
		if imageAdded {
			db, err := d.GobEncode()
			if err == nil {
				err = os.WriteFile(imagedbFilename, db, 0644)
				if err != nil {
					log.Printf("Error writing image database file: %s", err)
				}
			} else {
				log.Printf("Error encoding image database: %s", err)
			}
		}
		if len(mu.Urls) > 0 {
			f, err := os.Create(urlsFilename)
			if err == nil {
				err = gob.NewEncoder(f).Encode(knownUrls)
				if err != nil {
					log.Printf("Error encoding url database: %s", err)
				}
			} else {
				log.Printf("Error creating url file: %s", err)
			}
			f.Close()
		}
	}
}
