package gobot

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"image/jpeg"
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
	"unicode"

	"github.com/op/go-logging"

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
	bConnected  bool
	robotName   string
	contacts    map[string]Contact
}

const (
	wxLoginBaseURL = "https://login.weixin.qq.com"
	wxReferer      = "https://wx.qq.com/"
	wxUserAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.2840.71 Safari/537.36"
)

var (
	errLoginFail    = errors.New("Login failed")
	errLoginTimeout = errors.New("Login time out")
	errGetCodeFail  = errors.New("get code fail")
	errInit         = errors.New("init fail ret <> 0")
	errNoGroup      = errors.New("没有找到群")
	errHandleExist  = errors.New("命令处理器已经存在")
	errUUID         = errors.New("haven't get uuid")
	errUIN          = errors.New("haven't get uin")
	errPushLogin    = errors.New("PushLogin error")
)
var log = logging.MustGetLogger("wecat")

// HandleFunc type
//	used for RegisterHandle
type HandlerFunc func(args []string) string

var (
	wxHosts = []string{
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
		log.Error("get cookiejar fail", err)
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
		auto:        false,
	}, nil
}

// SetLogLevel
//	logging.Level   from github.com/op/go-logging
func (w *Wecat) SetLogLevel(l logging.Level) {
	logging.SetLevel(l, "wecat")
}

func (w *Wecat) SetRobotName(name string) {
	w.robotName = name
}

func (w *Wecat) GetUUID() error {
	if w.uuid != "" {
		return nil
	}

	// new AppIP useless
	/* 网页版微信有两个AppID，早期的是wx782c26e4c19acffb，在微信客户端上显示为应用名称为Web微信；现在用的是wxeb7ec651dd0aefa9，显示名称为微信网页版
	 */
	uri := wxLoginBaseURL + "/jslogin?appid=wx782c26e4c19acffb&fun=new&lang=zh_CN&_=" + w.timestamp()
	//uri := wxLoginBaseURL + "/jslogin?appid=wxeb7ec651dd0aefa9&fun=new&lang=zh_CN&_=" + w.timestamp()
	//result: window.QRLogin.code = 200; window.QRLogin.uuid = "xxx"; //wx782c26e4c19acffb  wxeb7ec651dd0aefa9
	data, err := w.get(uri)
	if err != nil {
		log.Error("get uuid fail", err)
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

func (w *Wecat) PushLogin() error {
	if w.loginRes.Wxuin == "" {
		if w.cfg.Uin == "" {
			log.Error("Never logined", errUIN)
			return errUIN
		}
		w.loginRes.Wxuin = w.cfg.Uin
	}

	uri := "https://wx.qq.com/cgi-bin/mmwebwx-bin/webwxpushloginurl?uin=" +
		w.loginRes.Wxuin

	if resp, err := w.get(uri); err != nil {
		return err
	} else {
		var res PushLoginResult
		if err := json.Unmarshal(resp, &res); err != nil {
			return err
		}
		if to.Int(res.RetCode) != 0 {
			log.Error("PushLogin", res.Msg)
			return errPushLogin
		}
		w.uuid = res.Uuid
		log.Info("PushLogin ok, uuid:", res.Uuid, ",uin:", w.loginRes.Wxuin)
	}

	// resp:   { 'msg':'all ok', 'uuid':'xxx', 'ret':'0' }
	return w.checkLogin(true)
}

func (w *Wecat) GenQrcode() error {
	if w.uuid == "" {
		err := errors.New("haven't get uuid")
		log.Error("gen qrcode fail", err)
		return err
	}

	uri := wxLoginBaseURL + "/qrcode/" + w.uuid + "?t=webwx&_=" + w.timestamp()
	//uri := wxLoginBaseURL + "/l/" + w.uuid

	resp, err := w.get(uri)

	//err = dispJPEG([]byte(resp))
	img, err := jpeg.Decode(bytes.NewReader([]byte(resp)))
	if err != nil {
		log.Error("Decode Qrcode", err)
		return err
	}

	if err := dispImage(img); err != nil {
		log.Error("dispImage:", err)
		return err
	}

	return nil
}

func (w *Wecat) Login() error {
	return w.checkLogin(false)
}

func (w *Wecat) checkLogin(scanned bool) error {
	if !scanned {
		defer shutJpegWin()
	}
	tip := 1
	if scanned {
		tip = 0
	}
	for {
		if !scanned {
			jpegLoop()
		}
		uri := fmt.Sprintf("%s/cgi-bin/mmwebwx-bin/login?tip=%d&uuid=%s&_=%s",
			wxLoginBaseURL, tip, w.uuid, w.timestamp())
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
				log.Info("scan code success")
				shutJpegWin()
				scanned = true
				tip = 0
			case "200":
				log.Info("login success, wait to redirect")
				re := regexp.MustCompile(`window.redirect_uri="(\S+?)";`)
				redirctURIs := re.FindStringSubmatch(string(data))

				if len(redirctURIs) > 1 {
					redirctURI := redirctURIs[1] + "&fun=new"
					w.redirectURI = redirctURI
					re = regexp.MustCompile(`/`)
					baseURIs := re.FindAllStringIndex(redirctURI, -1)
					w.baseURI = redirctURI[:baseURIs[len(baseURIs)-1][0]]
					if err := w.redirect(); err != nil {
						log.Error(err)
						return err
					}
					w.bConnected = true
					return nil
				}

				log.Notice("get redirct URL fail")

			case "408":
				err := errLoginTimeout
				log.Error(err)
				return err
			default:
				err := errLoginFail
				log.Error(err)
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
		log.Error("redirct fail", err)
		return err
	}

	var lr LoginResult
	if err = xml.Unmarshal(data, &lr); err != nil {
		log.Error("unmarshal fail", err)
		return err
	}

	w.loginRes = lr
	w.baseRequest["Uin"] = to.Int64(lr.Wxuin)
	w.baseRequest["Sid"] = lr.Wxsid
	w.baseRequest["Skey"] = lr.Skey
	w.baseRequest["DeviceID"] = w.deviceID
	return nil
}

// webwxlogout?redirect=1&type=0&skey=...
// application/x-www-form-urlenconded
//    sid=
//    uin=
func (w *Wecat) Logout() error {
	uri := w.baseURI + "/webwxlogout?redirect=1&type=1&skey=" + w.loginRes.Skey
	var urlValues = url.Values{}
	urlValues.Set("sid", to.String(w.baseRequest["Sid"]))
	urlValues.Set("uin", to.String(w.baseRequest["Uin"]))
	resp, err := http.PostForm(uri, urlValues)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	/*
		res, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil
		}
		log.Info(string(res))
	*/
	return nil
}

// Init after Login/PushLogin
func (w *Wecat) Init() error {
	uri := fmt.Sprintf("%s/webwxinit?pass_ticket=%s&skey=%s&r=%s", w.baseURI,
		w.loginRes.PassTicket, w.loginRes.Skey, w.timestamp())
	params := make(map[string]interface{})
	params["BaseRequest"] = w.baseRequest
	data, err := w.post(uri, params)
	if err != nil {
		log.Error("init post fail", err)
		return err
	}

	var res InitResult
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}

	w.user = res.User
	log.Infof("My name: %s, remarkName(%s), uin(%d)", w.user.UserName,
		w.user.RemarkName, w.user.Uin)
	// change RemarkName to robotName, for monitor
	if w.robotName != "" {
		w.user.RemarkName = w.robotName
	}
	w.syncKey = res.SyncKey

	if res.BaseResponse.Ret != 0 {
		log.Warning(errInit, "errMsg:", res.BaseResponse.ErrMsg)
	} else {
		// update Contacts
		for _, contact := range res.ContactList {
			w.updateContacts(contact)
		}
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
	for _, host := range wxHosts {
		uri := "https://" + host + "/cgi-bin/mmwebwx-bin/synccheck"
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
			//log.Warning("sync check fail", err)
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
	uri := w.baseURI + "/webwxstatusnotify?lang=zh_CN&pass_ticket=" +
		w.loginRes.PassTicket
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
	uri := fmt.Sprintf("%s/webwxgetcontact?sid=%s&skey=%s&pass_ticket=%s",
		w.baseURI, w.loginRes.Wxsid, w.loginRes.Skey, w.loginRes.PassTicket)
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
		w.updateContacts(contact)
	}

	return nil
}

func (w *Wecat) updateContacts(contact Contact) {
	if contact.NickName == "" {
		if contact.RemarkName != "" {
			contact.NickName = contact.RemarkName
		} else {
			contact.NickName = contact.UserName
		}
	}
	w.contacts[contact.UserName] = contact
	if contact.UserName[:2] == "@@" {
		//if _, ok := weGroups[contact.NickName]; !ok {
		weGroups[contact.NickName] = contact.UserName
		//}
	}
}

func (w *Wecat) WxSync() (*Message, error) {
	uri := fmt.Sprintf("%s/webwxsync?sid=%s&skey=%s&pass_ticket=%s",
		w.baseURI, w.loginRes.Wxsid, w.loginRes.Skey, w.loginRes.PassTicket)
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
	log.Info(desc)
	if err := f(); err != nil {
		log.Error("FAIL, exit now", err)
		os.Exit(1)
	}

	log.Infof("SUCCESS, use time: %.3f seconds",
		time.Now().Sub(start).Seconds())
}

func (w *Wecat) SendGroupMessage(message string, to string) error {
	if toGrp, ok := weGroups[to]; ok {
		log.Info("SendGroupMsg:", toGrp, "--->", message)
		return w.SendMessage(message, toGrp)
	}
	return errNoGroup
}

func (w *Wecat) SendMessage(message string, to string) error {
	uri := w.baseURI + "/webwxsendmsg?pass_ticket=" + w.loginRes.PassTicket
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
	msg["Status"] = 3
	msg["ImgStatus"] = 1
	params["Msg"] = msg
	params["Scene"] = 0
	data, err := w.post(uri, params)
	if err != nil {
		return err
	}
	var res SendMessageResult

	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}
	if retC := res.BaseResponse.Ret; retC != 0 {
		log.Errorf("SendMessage Retcode: %d,%s", retC, res.BaseResponse.ErrMsg)
		return fmt.Errorf("SendMessage Retcode: %d", retC)
	}
	return nil
}

func (w *Wecat) RegisterHandle(cmd string, cmdFunc HandlerFunc) error {
	cmd = strings.ToLower(cmd)
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

func unicodeTrim(ss string) string {
	ru := []rune(ss)
	for i := 0; i < len(ru); i++ {
		if !unicode.IsSpace(ru[i]) {
			return string(ru[i:])
		}
	}
	return ""
}

func (w *Wecat) handle(msg *Message) error {
	for _, contact := range msg.ModContactList {
		if _, ok := w.contacts[contact.UserName]; !ok {
			w.updateContacts(contact)
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
					log.Info("[*] ", w.getNickName(m.FromUserName), ": ", content)
					cmds := strings.Split(unicodeTrim(content), ",")
					if len(cmds) == 0 {
						return nil
					}
					//log.Info("cmds", cmds)
					cmds[0] = strings.ToLower(cmds[0])
					if cmdFunc, ok := handlers[strings.Trim(cmds[0], " \t")]; ok {
						//println("cmd", cmds[0], "argc:", len(cmds[1:]))
						reply := cmdFunc(cmds[1:])
						if reply != "" {
							if err := w.SendMessage(reply, m.FromUserName); err != nil {
								return err
							}
							log.Info("[#] ", w.user.NickName, ": ", reply)
						}
					} else if w.auto {
						reply, err := w.getTulingReply(m.Content, m.FromUserName)
						if err != nil {
							return err
						}

						if err := w.SendMessage(reply, m.FromUserName); err != nil {
							return err
						}
						// copy to test群
						//w.SendGroupMessage(reply, "test群")
						log.Info("[#] ", w.user.NickName, ": ", reply)
					}
				} else {
					log.Info("From group: ", w.getNickName(m.FromUserName))
					contents := strings.Split(m.Content, ":<br/>")
					log.Info("[*] ", w.getNickName(contents[0]), ": ", contents[1])
				}
			} else {
				if m.FromUserName != w.user.UserName {
					log.Info("[*] ", w.getNickName(m.FromUserName), ": ", m.Content)
					cmds := strings.Split(unicodeTrim(m.Content), ",")
					if len(cmds) == 0 {
						return nil
					}
					cmds[0] = strings.ToLower(cmds[0])
					if cmdFunc, ok := handlers[strings.Trim(cmds[0], " \t")]; ok {
						//println("cmd", cmds[0], "argc:", len(cmds[1:]))
						reply := cmdFunc(cmds[1:])
						if reply != "" {
							if err := w.SendMessage(reply, m.FromUserName); err != nil {
								return err
							}
							log.Info("[#] ", w.user.NickName, ": ", reply)
						}
					} else if !w.cfg.Tuling.GroupOnly {
						if w.auto {
							reply, err := w.getTulingReply(m.Content, m.FromUserName)
							if err != nil {
								return err
							}

							if err := w.SendMessage(reply, m.FromUserName); err != nil {
								return err
							}
							log.Info("[#] ", w.user.NickName, ": ", reply)
						}
					}

				} else {
					switch m.Content {
					case "退下":
						w.auto = false
					case "来人":
						w.auto = true
					default:
						log.Info("[#] ", w.user.NickName, ": ", m.Content)
					}
				}
			}
		case 51:
			if m.Content == "" {
				log.Info("sync ok")
			} else {
				log.Info("sync ok, 最近联系的联系人:", m.StatusNotifyUserName)
				log.Info("content-->", m.Content)
			}
		case 9999: // SYSNOTICE
		case 10000: //system message
		case 10002: // revoke message
		}
	}

	return nil
}

func (w *Wecat) Dail() error {
	for {
		retcode, selector := w.SyncCheck()
		switch retcode {
		case 1100: //未登录提示
			log.Error("logout with phone, bye")
			w.bConnected = false
			return nil
		case 1101: //未检测到登陆？
			log.Error("login web wecat at other palce, bye")
			w.bConnected = false
			return nil
		case 1102: //cookie值无效
			// web wecat wanna login
			log.Warning("cookie值无效")
		case 0:
			switch selector {
			case 2:
				msg, err := w.WxSync()
				if err != nil {
					log.Warning(err)
				}

				if err := w.handle(msg); err != nil {
					log.Error(err)
				}
			case 0:
				time.Sleep(time.Second)
			case 7: // Enter/Leave chat Room
			case 6, 4:
				if msg, err := w.WxSync(); err == nil {
					w.handle(msg)
				}
				time.Sleep(time.Second)
			}
		default:
			log.Error("unknow code", retcode)
		}
	}
}

// Start   test purpose
//	start one session started with GenQrcode,Login
func (w *Wecat) Start() {
	w.run("[*] get uuid ...", w.GetUUID)
	w.run("[*] generate qrcode ...", w.GenQrcode)
	w.run("[*] login ...", w.Login)
	w.run("[*] init wecat ...", w.Init)
	w.run("[*] open status notify ...", w.StatusNotify)
	w.run("[*] get contact ...", w.GetContact)
	/*
		for _, cc := range w.contacts {
			if cc.VerifyFlag&8 == 0 {
				log.Infof("%s,(%s),(%s)\n", cc.UserName, cc.NickName, cc.RemarkName)
			} else {
				log.Infof("pub(%s),(%s),(%s)\n", cc.UserName, cc.NickName, cc.RemarkName)
			}
		}
	*/
	w.run("[*] dail sync message ...", w.Dail)
	/*
		w.run("[*] PushLogin ...", w.PushLogin)
		w.run("[*] dail sync after PushLogin message ...", w.Dail)
	*/
}

// Connect Trylogin without Qrcode
func (w *Wecat) Connect() error {
	if w.loginRes.Wxuin != "" {
		if err := w.PushLogin(); err != nil {
			return err
		}
	} else {
		if err := w.GetUUID(); err != nil {
			return err
		}
		if err := w.GenQrcode(); err != nil {
			return err
		}
		if err := w.Login(); err != nil {
			return err
		}
	}

	if err := w.Init(); err == nil {
		log.Info("wxInit ok")
	} else {
		return err
	}
	if err := w.StatusNotify(); err != nil {
		return err
	}
	w.GetContact()
	return nil
}

func (w *Wecat) IsConnected() bool {
	return w.bConnected
}

func (w *Wecat) timestamp() string {
	return to.String(time.Now().Unix())
}

func (w *Wecat) get(uri string) ([]byte, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Referer", wxReferer)
	req.Header.Add("User-agent", wxUserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func buildJson(data map[string]interface{}) ([]byte, error) {
	buf := bytes.NewBufferString("")
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(&data); err != nil {
		return nil, err
	} else {
		return buf.Bytes(), nil
	}
}

func (w *Wecat) post(uri string, params map[string]interface{}) ([]byte, error) {
	//data, err := json.Marshal(params)
	data, err := buildJson(params)
	if err != nil {
		return nil, err
	}

	body := bytes.NewBuffer(data)
	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Referer", wxReferer)
	req.Header.Add("User-agent", wxUserAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

//	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
func init() {
	var format = logging.MustStringFormatter(
		`%{color}%{time:01-02 15:04:05}  ▶ %{level:.4s} %{color:reset} %{message}`,
	)

	logback := logging.NewLogBackend(os.Stderr, "", 0)
	logfmt := logging.NewBackendFormatter(logback, format)
	logging.SetBackend(logfmt)
}
