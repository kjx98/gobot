package main

import (
	"time"
	"github.com/kjx98/gobot"
)

func timeFunc(args []string) string {
	return time.Now().Format("2006-01-02 15:03:04")
}

func main() {
	cfg := gobot.NewConfig("")
	rebot, err := gobot.NewWecat(cfg)
	if err != nil {
		panic(err)
	}
	rebot.RegisterHandle("time", timeFunc)
	rebot.RegisterHandle("时间", timeFunc)

	rebot.Start()
}
