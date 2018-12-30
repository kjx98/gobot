package gobot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/op/go-logging"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func timeFunc(args []string) string {
	return time.Now().Format("2006-01-02 15:03:04")
}

var cfg = NewConfig("")
var wx *Wecat

func TestWxStart(t *testing.T) {
	fmt.Println(cfg)
	wx.SetLogLevel(logging.WARNING)
	if wx == nil {
		w, err := NewWecat(cfg)
		if err != nil {
			t.Error(err)
		}
		wx = w
	}

	wx.RegisterHandle("time", timeFunc)
	wx.RegisterHandle("时间", timeFunc)

	wx.Start()
	for gn, nn := range weGroups {
		t.Log("群", nn, "-->", gn)
	}
}

func TestWxConnect(t *testing.T) {
	if wx == nil {
		w, err := NewWecat(cfg)
		if err != nil {
			t.Error(err)
		}
		wx = w
	}

	wx.SetRobotName("JacK")
	wx.SetLogLevel(logging.INFO)
	if err := wx.Connect(); err != nil {
		t.Error(err)
	}
}

func TestSendGroupMessage(t *testing.T) {
	if wx == nil {
		w, err := NewWecat(cfg)
		if err != nil {
			t.Error(err)
		}
		wx = w
	}
	if wx.IsConnected() {
		if err := wx.SendGroupMessage("test only!!!\n测试换行\n", "test群"); err != nil {
			t.Error(err)
		}
		//wx.Dail()
	} else {
		t.Log("Not login, no test SendGroupMessage")
	}
	if err := wx.Logout(); err != nil {
		t.Error(err)
	}
}

func TestTuling(t *testing.T) {
	params := make(map[string]interface{})
	params["userid"] = "123123123"
	params["key"] = "808811ad0fd34abaa6fe800b44a9556a"
	params["info"] = "你好"

	data, err := json.Marshal(params)
	if err != nil {
		fmt.Println(err)
		return
	}

	body := bytes.NewBuffer(data)
	req, err := http.NewRequest("POST", "http://www.tuling123.com/openapi/api", body)
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Referer", wxReferer)
	req.Header.Add("User-agent", wxUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	data, _ = ioutil.ReadAll(resp.Body)

	fmt.Println(string(data))
}
