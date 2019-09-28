// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package cmdhook

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"

	"github.com/redbluescreen/sbrwxmpp/config"
)

type chatMsg struct {
	XMLName xml.Name `xml:"ChatMsg"`
	Message string   `xml:"Msg"`
}

type CmdHook struct {
	Client *http.Client
	Config *config.WebhookConfig
}

func (h *CmdHook) ProcessMessage(from string, body string) bool {
	if h.Config.Target == "" {
		return false
	}
	msg := chatMsg{}
	err := xml.Unmarshal([]byte(body), &msg)
	if err != nil {
		return false
	}
	if !strings.HasPrefix(msg.Message, "/") {
		return false
	}
	splits := strings.Split(strings.Split(from, "@")[0], ".")
	if len(splits) < 2 {
		return false
	}
	fromID := splits[1]
	qs := url.Values{
		"pid": {fromID},
		"cmd": {msg.Message},
	}
	req, err := http.NewRequest("POST", h.Config.Target+"?"+qs.Encode(), nil)
	if err != nil {
		return true
	}
	req.Header.Add("Authorization", h.Config.Secret)
	resp, err := h.Client.Do(req)
	if err != nil {
		return true
	}
	resp.Body.Close()
	return true
}
