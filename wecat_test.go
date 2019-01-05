package gobot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/op/go-logging"
)

var cfg = NewConfig("")

func TestWxStart(t *testing.T) {
	var wx *Wecat
	fmt.Println(cfg)
	wx.SetLogLevel(logging.WARNING)
	if w, err := NewWecat(cfg); err == nil {
		if err != nil {
			t.Error(err)
		}
		wx = w
	} else {
		t.Error("NewWecat", err)
		return
	}

	wx.RegisterTimeCmd()

	wx.Start()
	for gn, nn := range weGroups {
		t.Log("群", nn, "-->", gn)
	}
}

func TestWxConnect(t *testing.T) {
	wx, err := NewWecat(cfg)
	if err != nil {
		t.Error(err)
		return
	}

	// try load cookie
	wx.LoadCookie()
	wx.SetRobotName("JacK")
	wx.SetLogLevel(logging.INFO)
	if err := wx.Connect(); err != nil {
		t.Error(err)
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
	if wx.IsConnected() {
		startT := time.Now()
		wx.dailLoop(60)
		endt := time.Now()
		coT := endt.Sub(startT)
		t.Logf("dialLoop cost %.3f seconds", coT.Seconds())
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
