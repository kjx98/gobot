package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/kjx98/gobot"
	"github.com/kjx98/jabot"
	"github.com/op/go-logging"
	"time"

	"os"
	"strings"
)

var log = logging.MustGetLogger("wxJabot")
var username = flag.String("username", "mon@quant.zqhy8.com", "username")
var password = flag.String("password", "testme", "password")
var wx *gobot.Wecat
var wkPB = "wkpb@quant.zqhy8.com"
var pingInterval int64 = 300 // 5 minutes

func checkWxLive() {
	if err := wx.Connect(); err != nil {
		log.Info("无法登录微信", err)
	}
	if wx.IsConnected() {
		go func() {
			wx.Dail()
			if wx.IsConnected() {
				log.Error("Dail exit with wxConnected!!!")
			}
		}()
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: wxjabot [options]\n")
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()
	cfg := jabot.NewConfig("")
	cfg.Jid = *username
	cfg.Passwd = *password
	rebot, err := jabot.NewJabot(&cfg)
	if err != nil {
		panic(err)
	}
	rebot.RegisterTimeCmd()

	wxcfg := gobot.NewConfig("b689c0a4af2f424f8ab3ad6bc323d36e")
	if w, err := gobot.NewWecat(wxcfg); err != nil {
		panic("无法初始化网页微信")
	} else {
		wx = w
	}
	wxHook := func(args string) {
		if rebot.IsConnected() {
			log.Info("send wkpb:", args)
			rebot.SendMessage(args, wkPB)
		}
	}
	rebotHook := func(args string) {
		if wx.IsConnected() {
			log.Info("send defGroup:", args)
			wx.SendGroupMessage(args, "")
		}
	}
	//wx.RegisterTimeCmd()
	wx.RegisterHook("JacK", wxHook)

	rebot.RegisterHook("wkpb", rebotHook)
	//wx.SetRobotName("JacK")
	//wx.SetLogLevel(logging.INFO)
	wx.LoadCookie()
	checkWxLive()

	if err := rebot.Connect(); err != nil {
		fmt.Println("Connect", err)
		return
	}
	//go rebot.Dail()
	// start jabot daemon
	go func() {
		retry := 0
		// try Ping every 5 minutes
		nextPing := time.Now().Unix() + pingInterval
		ticker := time.NewTicker(time.Second * 10)
		// go routine for sendPing
		go func() {
			var curT int64
			for wx.IsConnected() {
				select {
				case <-ticker.C:
					curT = time.Now().Unix()
				}
				if rebot.IsConnected() && curT >= nextPing {
					rebot.Ping()
					nextPing = curT + pingInterval
				}
			}
		}()

		for wx.IsConnected() {
			if rebot.IsConnected() {
				rebot.Dail()
			}
			retry++
			if retry > 5 {
				time.Sleep(time.Second * 60)
			} else {
				time.Sleep(time.Second * 5)
			}
			if err := rebot.Connect(); err == nil {
				retry = 0
				nextPing = time.Now().Unix() + pingInterval
				rebot.AddChat(wkPB)
				log.Warning("Jabot reconnected ok")
			} else {
				log.Error("jabot connect", err)
			}
		}
		ticker.Stop()
		rebot.Close()
		log.Warning("wxJabot exit!!!")
	}()

	//wx.SetLogLevel(logging.WARNING)
	for wx.IsConnected() {
		in := bufio.NewReader(os.Stdin)
		line, err := in.ReadString('\n')
		if err != nil {
			continue
		}
		if len(line) >= 4 && strings.ToLower(line[:4]) == "quit" {
			break
		}
		line = strings.TrimRight(line, "\n")

		tokens := strings.SplitN(line, " ", 2)
		if len(tokens) == 2 {
			rebot.SendMessage(tokens[1], tokens[0])
		}
	}
}

//  `%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
func init() {
	var format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05}  ▶ %{level:.4s} %{color:reset} %{message}`,
	)

	logback := logging.NewLogBackend(os.Stderr, "", 0)
	logfmt := logging.NewBackendFormatter(logback, format)
	logging.SetBackend(logfmt)
}
