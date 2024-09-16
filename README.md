# NS2 Query Bot

This is a simple bot that queries the Natural Selection 2 servers and sends messages to the specified Discord channel
if there are enough players to start the round.

# Compiling

Grab a [Golang](https://golang.org/dl/) installer or install it from your repository, then run `go get -u github.com/rkfg/ns2query`. You should find the binary in your `$GOPATH/bin` directory. Alternatively, clone the repository and run `go build` inside. Another way is to run `go run cmd/build.go` to make a release for all supported platforms.

# Configuration

Copy the provided `config_sample.json` file to `config.json` and change it to your needs. You should put your Discord bot token to the `token` parameter, put the channel ID to the `channel_id` parameter (it's the last long number in the Discord URL: `https://discord.com/channels/AAAAAAAAAAAAAAA/BBBBBBBBBBBBBB`, you need to copy the `BBBBBBBBBBBBBB` part). `query_interval` specifies the interval (in seconds) between querying the same server. `query_timeout` sets the server query timeout and defaults to 3 seconds. All servers are queried in parallel.

Then setup the servers you want to watch. `name` can be anything, the bot will use it for announcing, address should be in the `ip:port` form (where port is `the game port + 1`, i.e. if you see 27015 in the Steam server browser use 27016 here). `player_slots` is the number of slots for players and `spec_slots` is spectator slots. The bot uses those to post "last minute" notifications. `status_template` is an optional parameter that defines the bot's status line. It's used to quickly see the server status without asking the bot directly. The status is displayed on Discord as "Playing ...", you can specify the format in this parameter using Go's template syntax. See `config_sample.json` for a full example with all available variables. tl;dr variables are used as `{{ .VarName }}`, all other characters are printed as is. The variables are: `ServerName`, `Players`, `PlayerSlots`, `SpecSlots`, `FreeSlots`, `TotalSlots`, `Map`, `Skill`. Hopefully, they're self-describing.

`id_url` is an optional per server parameter that lets you specify an URL that serves a JSON with player Steam IDs that are currently on this server. You can use [this mod](https://steamcommunity.com/sharedfiles/filedetails/?id=2714142788) to grab them and then provide web access to the file using any avaliable web server. The bot will announce connecting players that are in the database using their Discord tags. The announce will be delayed by `announce_delay` seconds, if more known players join during that period they all will be announced altogether. It's a simple rate limiter to prevent spam. `regular_timeout` is a period of time in seconds after which a known player (aka regular) that left the server is forgotten by the bot and can be announced again. This is to prevent multiple announces in case the player leaves and rejoins in a short time (because of a crash or otherwise). If you want these announcements to go to a different channel, set `regular_channel_id`.

`down_notify_ids` and `up_notify_ids` may be optionally set to arrays of Discord IDs to notify (ping) if the server goes down and back online. It's NOT your Discord username but a long unique number ID that you can find by right-clicking a user and choosing "Copy User ID" in the dropdown menu. These parameters should ALWAYS be set as arrays even if you only want to ping one user.

If you already have a database that's been populated before these changes, run the bot with `--reindex` to fill the Steam ID => Discord index. All new players registering themselves with `-bind` will be indexed automatically.

The `users` section lets you specify the Discord IDs that have special privileges. Currently it's only used for the `-bindu` command that's not shown in the help message because it's special. To use it the user must be defined in the `users` section as `"123123123": "admin"`, only `admin` role is defined by now and it only allows access to `-bindu`. This command allows to bind a steam ID to any Discord user and is meant to be used by admins to populate the database. Call it as `-bindu DiscordName#3333 https://steamcommunity.com/id/steamprofilename`. To unbind any user call `-bindu DiscordName#3333`.

The `seeding` section defines the player number boundaries. Inside that section there are two most important parameters, `seeding` (the bot will announce that the server is getting seeded when at least this many players have connected) and `almost_full` (it will say that the server is getting filled but there are still slots if you want to play). The `cooldown` parameter is used when the number of players fluctuates between two adjacent states. For example, if the `seeding` parameter is `4` and some players join and leave so the number of players changes back and forth between 3 and 4, this cooldown parameter is used to temporarily mute the new messages about seeding. It's the number of seconds after the last promotion (getting a higher status) during which demotions (lowering the status) are ignored. If the server empties normally, then after this cooldown period the seeding announcements will be restored.

`threads` lets you list the channel threads the bot should participate in, the `join` parameter specifies whether the bot should enter the thread automatically (or you can invite it manually by mentioning). Threads and channels are mostly the same internally, just a number from the channel URL (or click "Copy Channel ID"/"Copy Thread ID" in the context menu). The `meme` parameter makes the bot upvote every image/video/URL posted in that channel/thread, to make it easier for everyone to upvote by just clicking the existing reaction. `competition` (which would not work without `meme`) will count the upvotes every day and post the most upvoted meme in the channel which ID is specified by `announce_winner_to`.

`no_self_upvote` would prevent the poster to upvote themselves for free, the bot would then cancel the autoupvote and post a clown reaction instead. If the poster removes their vote, the bot would additionally post a wink reaction. It would also only post a wink reaction if the poster manages to upvote before the bot's autoupvote.

The competition parameters define the announcement time (`competition_announcement`, hours in UTC+0), the backward deadline for the meme (`competition_deadline`, hours in UTC+0) and the day length (`competition_length` in hours). The backward deadline means the hour of the previous day that's still considered "today". This is done to allow the late memes to be able to win, for example, a meme posted at 23:00 would only have 1 hour to get the most votes while a meme posted at 10:00 would have 14 hours. This isn't fair and as such it's possible to include the late memes from 2 days ago into the competition. Here's an example (all times are in UTC+0):
```json
{
    "competition_deadline": 17,
    "competition_announcement": 5,
    "competition_length": 30,
}
```
Assuming today's Wednesday, this config tells the following:
- the memes posted on Monday (!) since 17:00 participate
- 30 hours since Monday 17:00 are considered, that is until Tuesday 23:00
- the winner is announced today at 05:00

While it could be confusing, it's pretty flexible to set whatever overlapping periods you might want. If you only want to consider the previous day (from midnight to midnight) set both `competition_deadline` and `competition_length` to 24. If you want to pick the memes from two last days, set `competition_deadline` to 0 and `competition_length` to 48. If you prefer nicely aged memes from 2 days ago but only for that day, set `competition_deadline` to 0 and `competition_length` to 24. Also, `competition_deadline` can be negative so you could push it even further back. Currently the winner is announced daily and it can't be changed (except for the announcement hour).

`track_reposts` should help with detecting meme reposts (only works in `meme` channels/threads). The bot keeps perceptual image hashes, video thumbnails, and raw URLs in the file databases, `imagedb.bin` and `urls.bin`. If a similar image/video/same URL is posted again the bot reports the duplicate with the relevant links. Of course, false positives and negatives are possible but it's good enough to detect compressed/resized images. Not very good with cropping. This code uses the awesome [duplo](https://github.com/rivo/duplo) library.