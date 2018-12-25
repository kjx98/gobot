package gobot

import (
	"encoding/json"
	"fmt"
)

//tuling
type News struct {
	Article   string `json:"article"`
	Source    string `json:"source"`
	Icon      string `json:"icon"`
	DetailURL string `json:"detailurl"`
}

type Menu struct {
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Info      string `json:"info"`
	DetailURL string `json:"detailurl"`
}

type Reply struct {
	Code int         `json:"code"`
	Text string      `json:"text"` //100000
	URL  string      `json:"url"`  //200000
	List interface{} `json:"list"` //302000 []News 308000 []Menu
}

var tulingURL = "http://www.tuling123.com/openapi/api"

func (w *Wecat) getTulingReply(msg string, uid string) (string, error) {
	params := make(map[string]interface{})
	params["userid"] = uid
	params["key"] = w.cfg.Tuling.KeyAPI
	params["info"] = msg

	body, err := w.post(w.cfg.Tuling.URL, params)

	if err != nil {
		return "", err
	}

	var reply Reply

	if err := json.Unmarshal(body, &reply); err != nil {
		return "", err
	}

	switch reply.Code {
	case 100000:
		return reply.Text, nil
	case 200000:
		return reply.Text + " " + reply.URL, nil
	case 302000:
		var res string
		news := reply.List.([]News)
		for _, n := range news {
			res += fmt.Sprintf("%s\n%s\n", n.Article, n.DetailURL)
		}

		return res, nil
	case 308000:
		var res string
		menu := reply.List.([]Menu)
		for _, m := range menu {
			res += fmt.Sprintf("%s\n%s\n%s\n", m.Name, m.Info, m.DetailURL)
		}
		return res, nil
	default:
		return reply.Text, nil
	}

	// return "哦", nil
}
