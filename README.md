# NS2 Query Bot

This is a simple bot that queries the Natural Selection 2 servers and sends messages to the specified Discord channel
if there are enough players to start the round.

# Compiling

Grab a [Golang](https://golang.org/dl/) installer or install it from your repository, then run `go get -u github.com/rkfg/ns2query`.
You should find the binary in your `$GOPATH/bin` directory. Alternatively, clone the repository and run `go build` inside.

# Configuration

Copy the provided `config_sample.json` file to `config.json` and change it to your needs. You should put your Discord bot token to the
`token` parameter, put the channel ID to the `channel_id` parameter (it's the last long number in the Discord URL: 
`https://discord.com/channels/AAAAAAAAAAAAAAA/BBBBBBBBBBBBBB`, you need to copy the `BBBBBBBBBBBBBB` part). `query_interval` specifies
the interval (in seconds) between querying the same server. All servers are queried in parallel.

Then setup the servers you want to watch. `name` can be anything, the bot will use it for announcing, address should be in the `ip:port`
form (where port is `the game port + 1`, i.e. if you see 27015 in the Steam server browser use 27016 here). `player_slots` is the number of
slots for players and `spec_slots` is spectator slots. The bot uses those to post "last minute" notifications.

The `seeding` section defines the player number boundaries. Inside that section there are two most important parameters, `seeding` (the bot
will announce that the server is getting seeded when at least this many players have connected) and `almost_full` (it will say that the
server is getting filled but there are still slots if you want to play). The `cooldown` parameter is used when the number of players
fluctuates between two adjacent states. For example, if the `seeding` parameter is `4` and some players join and leave so the number of
players changes back and forth between 3 and 4, this cooldown parameter is used to temporarily mute the new messages about seeding. It's
the number of seconds after the last promotion (getting a higher status) during which demotions (lowering the status) are ignored. If the
server empties normally, then after this cooldown period the seeding announcements will be restored.