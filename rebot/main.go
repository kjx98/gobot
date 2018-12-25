package main

import (
	"github.com/kjx98/gobot"
)

func main() {
	cfg := gobot.NewConfig("")
	rebot, err := gobot.NewWecat(cfg)
	if err != nil {
		panic(err)
	}

	rebot.Start()
}
