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
slots for players and `spec_slots` is spectator slots. The bot uses those to post "last minute" notifications. `status_template` is an
optional parameter that defines the bot's status line. It's used to quickly see the server status without asking the bot directly.
The status is displayed on Discord as "Playing ...", you can specify the format in this parameter using Go's template syntax. See
`config_sample.json` for a full example with all available variables. tl;dr variables are used as `{{ .VarName }}`, all other characters
are printed as is. The variables are: `ServerName`, `Players`, `PlayerSlots`, `SpecSlots`, `FreeSlots`, `TotalSlots`, `Map`, `Skill`.
Hopefully, they're self-describing.

`id_url` is an optional per server parameter that lets you specify an URL that serves a JSON with player Steam IDs that are currently on
this server. You can user [this mod](https://steamcommunity.com/sharedfiles/filedetails/?id=2714142788) to grab them and then provide
web access to the file using any avaliable web server. The bot will announce connecting players that are in the database using their
Discord tags. The announce will be delayed by `announce_delay` seconds, if more known players join during that period they all will be
announced altogether. It's a simple rate limiter to prevent spam. `regular_timeout` is a period of time in seconds after which a known
player (aka regular) that left the server is forgotten by the bot and can be announced again. This is to prevent multiple announces in
case the player leaves and rejoins in a short time (because of a crash or otherwise). If you want these announcements to go to a different
channel, set `regular_channel_id`.

`down_notify_ids` and `up_notify_ids` may be optionally set to arrays of Discord IDs to notify (ping) if the server goes down and back online. It's NOT your Discord username but a long unique number ID that you can find by right-clicking a user and choosing "Copy User ID" in the dropdown menu. These parameters should ALWAYS be set as arrays even if you only want to ping one user.

If you already have a database that's been populated before these changes, run the bot with `--reindex` to fill the Steam ID => Discord
index. All new players registering themselves with `-bind` will be indexed automatically.

The `users` section lets you specify the Discord IDs that have special privileges. Currently it's only used for the `-bindu` command
that's not shown in the help message because it's special. To use it the user must be defined in the `users` section as
`"123123123": "admin"`, only `admin` role is defined by now and it only allows access to `-bindu`. This command allows to bind a steam ID
to any Discord user and is meant to be used by admins to populate the database. Call it as `-bindu DiscordName#3333 https://steamcommunity.com/id/steamprofilename`. To unbind any user call `-bindu DiscordName#3333`.

The `seeding` section defines the player number boundaries. Inside that section there are two most important parameters, `seeding` (the bot
will announce that the server is getting seeded when at least this many players have connected) and `almost_full` (it will say that the
server is getting filled but there are still slots if you want to play). The `cooldown` parameter is used when the number of players
fluctuates between two adjacent states. For example, if the `seeding` parameter is `4` and some players join and leave so the number of
players changes back and forth between 3 and 4, this cooldown parameter is used to temporarily mute the new messages about seeding. It's
the number of seconds after the last promotion (getting a higher status) during which demotions (lowering the status) are ignored. If the
server empties normally, then after this cooldown period the seeding announcements will be restored.