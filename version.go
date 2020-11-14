package main

import "fmt"

var (
	version = "unknown"
	date    = "unknown"
	source  = "https://github.com/rkfg/ns2query"
)

func versionString() string {
	return fmt.Sprintf("Version %s built on %s. Source: %s", version, date, source)
}
