package gobot

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"log"

	"github.com/kjx98/golib/to"
)

type Wecat struct {
	cfg         Config
	uuid        string
	baseURI     string
	redirectURI string
	loginRes    LoginResult
	deviceID    string
	syncKey     SyncKey
	user        User
	baseRequest map[string]interface{}
	syncHost    string
	client      *http.Client
	auto        bool
	showRebot   bool
	contacts    map[string]Contact
}

const (
	LoginBaseURL = "https://login.weixin.qq.com"
	WxReferer    = "https://wx.qq.com/"
	WxUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36"
)

var (
	errLoginFail    = errors.New("Login failed")
	errLoginTimeout = errors.New("Login time out")
	errGetCodeFail  = errors.New("get code fail")
	errInit         = errors.New("init fail ret <> 0")
	errNoGroup      = errors.New("没有找到群")
	errHandleExist  = errors.New("命令处理器已经存在")
)

type HandlerFunc func(args []string) string

var (
	Hosts = []string{
		"webpush.wx.qq.com",
		"webpush.weixin.qq.com",
		"webpush.wx2.qq.com",
		"webpush2.weixin.qq.com",
		//"webpush2.wx.qq.com",
		"webpush.wechat.com",
		"webpush1.wechat.com",
		"webpush2.wechat.com",
		//"webpush1.wechatapp.com",
	}
)

var handlers = map[string]HandlerFunc{}
var weGroups = map[string]string{}

func NewWecat(cfg Config) (*Wecat, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Print("get cookiejar fail", err)
		return nil, err
	}

	client := &http.Client{
		CheckRedirect: nil,
		Jar:           jar,
	}

	rand.Seed(time.Now().Unix())
	randID := strconv.Itoa(rand.Int())

	return &Wecat{
		cfg:         cfg,
		client:      client,
		deviceID:    "e" + randID[2:17],
		baseRequest: make(map[string]interface{}),
		contacts:    make(map[string]Contact),
		auto:        true,
	}, nil
}

func (w *Wecat) GetUUID() error {
	if w.uuid != "" {
		return nil
	}

	uri := LoginBaseURL + "/jslogin?appid=wx782c26e4c19acffb&fun=new&lang=zh_CN&_=" + w.timestamp()
	//result: window.QRLogin.code = 200; window.QRLogin.uuid = "xxx"; //wx782c26e4c19acffb  wxeb7ec651dd0aefa9
	data, err := w.get(uri)
	if err != nil {
		log.Print("get uuid fail", err)
		return err
	}

	res := make(map[string]string)
	datas := strings.Split(string(data), ";")
	for _, d := range datas {
		kvs := strings.Split(d, " = ")
		if len(kvs) == 2 {
			res[strings.Trim(kvs[0], " ")] = strings.Trim(strings.Trim(kvs[1], " "), "\"")
		}
	}
	if res["window.QRLogin.code"] == "200" {
		if uuid, ok := res["window.QRLogin.uuid"]; ok {
			w.uuid = uuid
			return nil
		}
	}

	return fmt.Errorf(string(data))
}

func (w *Wecat) GenQrcode() error {
	if w.uuid == "" {
		err := errors.New("haven't get uuid")
		log.Print("gen qrcode fail", err)
		return err
	}

	uri := LoginBaseURL + "/qrcode/" + w.uuid + "?t=webwx&_=" + w.timestamp()

	resp, err := w.get(uri)

	err = dispJPEG([]byte(resp))
	//img, err := jpeg.Decode(bytes.NewReader([]byte(resp)))

	if err != nil {
		fmt.Println("dispJPEG:", err)
		return err
	}

	return nil
}

func (w *Wecat) Login() error {
	defer shutJpegWin()
	return w.Relogin(false)
}

func (w *Wecat) Relogin(reLogin bool) error {
	tip := 1
	if reLogin {
		tip = 0
	}
	for {
		if !reLogin {
			jpegLoop()
		}
		uri := fmt.Sprintf("%s/cgi-bin/mmwebwx-bin/login?tip=%d&uuid=%s&_=%s", LoginBaseURL, tip, w.uuid, w.timestamp())
		data, err := w.get(uri)
		if err != nil {
			return err
		}

		re := regexp.MustCompile(`window.code=(\d+);`)
		codes := re.FindStringSubmatch(string(data))
		if len(codes) > 1 {
			code := codes[1]
			switch code {
			case "201":
				log.Print("scan code success")
				tip = 0
			case "200":
				log.Print("login success, wait to redirect")
				re := regexp.MustCompile(`window.redirect_uri="(\S+?)";`)
				redirctURIs := re.FindStringSubmatch(string(data))

				if len(redirctURIs) > 1 {
					redirctURI := redirctURIs[1] + "&fun=new"
					w.redirectURI = redirctURI
					re = regexp.MustCompile(`/`)
					baseURIs := re.FindAllStringIndex(redirctURI, -1)
					w.baseURI = redirctURI[:baseURIs[len(baseURIs)-1][0]]
					if err := w.redirect(); err != nil {
						log.Print(err)
						return err
					}
					return nil
				}

				log.Print("get redirct URL fail")

			case "408":
				err := errLoginTimeout
				log.Print(err)
				return err
			default:
				err := errLoginFail
				log.Print(err)
				return err
			}
		} else {
			return errGetCodeFail
		}

		time.Sleep(time.Second * time.Duration(2))
	}
}

func (w *Wecat) redirect() error {
	data, err := w.get(w.redirectURI)
	if err != nil {
		log.Print("redirct fail", err)
		return err
	}

	var lr LoginResult
	if err = xml.Unmarshal(data, &lr); err != nil {
		log.Print("unmarshal fail", err)
		return err
	}

	w.loginRes = lr
	w.baseRequest["Uin"] = to.Int64(lr.Wxuin)
	w.baseRequest["Sid"] = lr.Wxsid
	w.baseRequest["Skey"] = lr.Skey
	w.baseRequest["DeviceID"] = w.deviceID
	return nil
}

func (w *Wecat) Init() error {
	uri := fmt.Sprintf("%s/webwxinit?pass_ticket=%s&skey=%s&r=%s", w.baseURI, w.loginRes.PassTicket, w.loginRes.Skey, w.timestamp())
	params := make(map[string]interface{})
	params["BaseRequest"] = w.baseRequest
	data, err := w.post(uri, params)
	if err != nil {
		log.Print("init post fail", err)
		return err
	}

	var res InitResult
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}

	w.user = res.User
	w.syncKey = res.SyncKey

	if res.BaseResponse.Ret != 0 {
		log.Print(errInit, "errMsg:", res.BaseResponse.ErrMsg)
	} else {
		// 订阅号
		/*
			for _, cc := range res.MPSubscribeMsgList {
				println(cc.UserName, " 订阅号:", cc.NickName)
			}
		*/
	}

	return nil
}

func (w *Wecat) strSyncKey() string {
	kvs := []string{}
	for _, list := range w.syncKey.List {
		kvs = append(kvs, to.String(list.Key)+"_"+to.String(list.Val))
	}

	return strings.Join(kvs, "|")
}

func (w *Wecat) SyncCheck() (retcode, selector int) {
	for _, host := range Hosts {
		uri := fmt.Sprintf("https://%s/cgi-bin/mmwebwx-bin/synccheck", host)
		v := url.Values{}
		v.Add("r", w.timestamp())
		v.Add("sid", w.loginRes.Wxsid)
		v.Add("uin", w.loginRes.Wxuin)
		v.Add("skey", w.loginRes.Skey)
		v.Add("deviceid", w.deviceID)
		v.Add("synckey", w.strSyncKey())
		v.Add("_", w.timestamp())
		uri = uri + "?" + v.Encode()

		data, err := w.get(uri)
		if err != nil {
			//log.Print("sync check fail", err)
			continue
		}

		re := regexp.MustCompile(`window.synccheck={retcode:"(\d+)",selector:"(\d+)"}`)
		codes := re.FindStringSubmatch(string(data))
		if len(codes) > 2 {
			return to.Int(codes[1]), to.Int(codes[2])
		}
	}

	return 9999, 0
}

func (w *Wecat) StatusNotify() error {
	uri := fmt.Sprintf("%s/webwxstatusnotify?lang=zh_CN&pass_ticket=%s", w.baseURI, w.loginRes.PassTicket)
	params := make(map[string]interface{})
	params["BaseRequest"] = w.baseRequest
	params["Code"] = 3
	params["FromUserName"] = w.user.UserName
	params["ToUserName"] = w.user.UserName
	params["ClientMsgId"] = int(time.Now().Unix())
	data, err := w.post(uri, params)
	if err != nil {
		return err
	}

	var res StatusNotifyResult

	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}

	if res.BaseResponse.Ret != 0 {
		return fmt.Errorf("%s", res.BaseResponse.ErrMsg)
	}
	return nil
}

func (w *Wecat) GetContact() error {
	uri := fmt.Sprintf("%s/webwxgetcontact?sid=%s&skey=%s&pass_ticket=%s", w.baseURI, w.loginRes.Wxsid, w.loginRes.Skey, w.loginRes.PassTicket)
	params := make(map[string]interface{})
	params["BaseRequest"] = w.baseRequest

	data, err := w.post(uri, params)
	if err != nil {
		return err
	}

	var contacts Contacts
	if err := json.Unmarshal(data, &contacts); err != nil {
		return err
	}

	for _, contact := range contacts.MemberList {
		if contact.NickName == "" {
			contact.NickName = contact.UserName
		}
		w.contacts[contact.UserName] = contact
		if contact.UserName[:2] == "@@" {
			if _, ok := weGroups[contact.NickName]; !ok {
				weGroups[contact.NickName] = contact.UserName
			}
		}
	}

	return nil
}

func (w *Wecat) WxSync() (*Message, error) {
	uri := fmt.Sprintf("%s/webwxsync?sid=%s&skey=%s&pass_ticket=%s", w.baseURI, w.loginRes.Wxsid, w.loginRes.Skey, w.loginRes.PassTicket)
	params := make(map[string]interface{})
	params["BaseRequest"] = w.baseRequest
	params["SyncKey"] = w.syncKey
	params["rr"] = ^int(time.Now().Unix())

	data, err := w.post(uri, params)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	if msg.BaseResponse.Ret == 0 {
		w.syncKey = msg.SyncKey
	}
	//TODO
	return &msg, nil
}

func (w *Wecat) run(desc string, f func() error) {
	start := time.Now()
	log.Println(desc)
	if err := f(); err != nil {
		log.Print("FAIL, exit now", err)
		os.Exit(1)
	}

	log.Print("SUCCESS, use time", time.Now().Sub(start).Nanoseconds())
}

func (w *Wecat) SendGroupMessage(message string, to string) error {
	if toGrp, ok := weGroups[to]; ok {
		return w.SendMessage(message, toGrp)
	}
	return errNoGroup
}

func (w *Wecat) SendMessage(message string, to string) error {
	uri := fmt.Sprintf("%s/webwxsendmsg?pass_ticket=%s", w.baseURI, w.loginRes.PassTicket)
	clientMsgID := w.timestamp() + "0" + strconv.Itoa(rand.Int())[3:6]
	params := make(map[string]interface{})
	params["BaseRequest"] = w.baseRequest
	msg := make(map[string]interface{})
	msg["Type"] = 1
	msg["Content"] = message
	msg["FromUserName"] = w.user.UserName
	msg["ToUserName"] = to
	msg["LocalID"] = clientMsgID
	msg["ClientMsgId"] = clientMsgID
	params["Msg"] = msg
	_, err := w.post(uri, params)
	if err != nil {
		return err
	}

	return nil
}

func (w *Wecat) RegisterHandle(cmd string, cmdFunc HandlerFunc) error {
	if _, ok := handlers[cmd]; ok {
		return errHandleExist
	}
	handlers[cmd] = cmdFunc
	return nil
}

func (w *Wecat) getNickName(userName string) string {
	if v, ok := w.contacts[userName]; ok {
		return v.NickName
	}

	return userName
}

func (w *Wecat) handle(msg *Message) error {
	for _, contact := range msg.ModContactList {
		if _, ok := w.contacts[contact.UserName]; !ok {
			if contact.NickName == "" {
				contact.NickName = contact.UserName
			}
			w.contacts[contact.UserName] = contact
			if contact.UserName[:2] == "@@" {
				if _, ok := weGroups[contact.NickName]; !ok {
					weGroups[contact.NickName] = contact.UserName
				}
			}
			println("Mod contact", contact.UserName, contact.NickName)
		}
	}

	for _, m := range msg.AddMsgList {
		m.Content = strings.Replace(m.Content, "&lt;", "<", -1)
		m.Content = strings.Replace(m.Content, "&gt;", ">", -1)
		switch m.MsgType {
		case 1:
			if m.FromUserName[:2] == "@@" { //群消息
				content := strings.Split(m.Content, ":<br/>")[1]
				if (w.user.NickName != "" && strings.Contains(content, "@"+w.user.NickName)) ||
					(w.user.RemarkName != "" && strings.Contains(content, "@"+w.user.RemarkName)) {
					content = strings.Replace(content, "@"+w.user.NickName, "", -1)
					content = strings.Replace(content, "@"+w.user.RemarkName, "", -1)
					//println("From group: ", w.getNickName(m.FromUserName))
					fmt.Println("[*] ", w.getNickName(m.FromUserName), ": ", content)
					if w.auto {
						reply, err := w.getReply(m.Content, m.FromUserName)
						if err != nil {
							return err
						}

						if w.showRebot {
							reply = w.cfg.Tuling.Keys[w.user.NickName].Name + ": " + reply
						}
						if err := w.SendMessage(reply, m.FromUserName); err != nil {
							return err
						}
						fmt.Println("[#] ", w.user.NickName, ": ", reply)
					}
				} else {
					log.Print("From group: ", w.getNickName(m.FromUserName))
					contents := strings.Split(m.Content, ":<br/>")
					fmt.Println("[*] ", w.getNickName(contents[0]), ": ", contents[1])
				}
			} else {
				if m.FromUserName != w.user.UserName {
					fmt.Println("[*] ", w.getNickName(m.FromUserName), ": ", m.Content)
					cmds := strings.Split(m.Content, " ,")
					if len(cmds) == 0 {
						return nil
					}
					if cmdFunc, ok := handlers[cmds[0]]; ok {
						reply := cmdFunc(cmds[1:])
						if err := w.SendMessage(reply, m.FromUserName); err != nil {
							return err
						}
						fmt.Println("[#] ", w.user.NickName, ": ", reply)
					} else {
						if w.auto {
							reply, err := w.getReply(m.Content, m.FromUserName)
							if err != nil {
								return err
							}

							if w.showRebot {
								reply = w.cfg.Tuling.Keys[w.user.NickName].Name + ": " + reply
							}
							if err := w.SendMessage(reply, m.FromUserName); err != nil {
								return err
							}
							fmt.Println("[#] ", w.user.NickName, ": ", reply)
						}
					}

				} else {
					switch m.Content {
					case "退下":
						w.auto = false
					case "来人":
						w.auto = true
					case "显示":
						w.showRebot = true
					case "隐身":
						w.showRebot = false
					default:
						fmt.Println("[#] ", w.user.NickName, ": ", m.Content)
					}
				}
			}
		case 51:
			log.Print("sync ok")
		}
	}

	return nil
}

func (w *Wecat) Dail() error {
	for {
		retcode, selector := w.SyncCheck()
		switch retcode {
		case 1100:
			log.Print("logout with phone, bye")
			return nil
		case 1101:
			log.Print("login web wecat at other palce, bye")
			return nil
		case 1102:
			// web wecat wanna login
			log.Print("web wecat try to login")
		case 0:
			switch selector {
			case 2:
				msg, err := w.WxSync()
				if err != nil {
					log.Print(err)
				}

				if err := w.handle(msg); err != nil {
					log.Print(err)
				}
			case 0:
				time.Sleep(time.Second)
			case 6, 4:
				w.WxSync()
				time.Sleep(time.Second)
			}
		default:
			log.Print("unknow code", retcode)
		}
	}
}

func (w *Wecat) Start() {
	w.run("[*] get uuid ...", w.GetUUID)
	w.run("[*] generate qrcode ...", w.GenQrcode)
	w.run("[*] login ...", w.Login)
	w.run("[*] init wecat ...", w.Init)
	w.run("[*] open status notify ...", w.StatusNotify)
	w.run("[*] get contact ...", w.GetContact)
	/*
		for _, cc := range w.contacts {
			if cc.MemberCount == 0 {
				fmt.Printf("%s,(%s),(%s)\n", cc.UserName, cc.NickName, cc.RemarkName)
			} else {
				fmt.Printf("%s,@(%s),(%s)\n", cc.UserName, cc.NickName, cc.RemarkName)
			}
		}
	*/
	w.run("[*] dail sync message ...", w.Dail)
}

func (w *Wecat) timestamp() string {
	return to.String(time.Now().Unix())
}

func (w *Wecat) get(uri string) ([]byte, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Referer", WxReferer)
	req.Header.Add("User-agent", WxUserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (w *Wecat) post(uri string, params map[string]interface{}) ([]byte, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	body := bytes.NewBuffer(data)
	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Referer", WxReferer)
	req.Header.Add("User-agent", WxUserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
