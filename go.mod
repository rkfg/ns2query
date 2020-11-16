module rkfg.me/ns2query

go 1.15

require (
	github.com/Philipp15b/go-steamapi v0.0.0-20200122161829-728086d96bab
	github.com/bwmarrin/discordgo v0.22.0
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/rumblefrog/go-a2s v1.0.0
	github.com/syndtr/goleveldb v1.0.0
)

// temporary override until pull requests #11 and #12 are merged
replace github.com/Philipp15b/go-steamapi => github.com/rkfg/go-steamapi v0.0.0-20201114201332-f991dbb01c37
