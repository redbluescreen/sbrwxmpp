// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package chatlog

import (
	"encoding/xml"
	"fmt"
	"time"
)

type chatMsgType uint

func (t chatMsgType) String() string {
	switch t {
	case chatMsgTypeGlobal:
		return "global"
	case chatMsgTypeEvent:
		return "event"
	case chatMsgTypeWhisper:
		return "whisper"
	case chatMsgTypeGroup:
		return "group"
	default:
		return "unknown"
	}
}

const (
	chatMsgTypeGlobal  chatMsgType = 0
	chatMsgTypeEvent   chatMsgType = 1
	chatMsgTypeWhisper chatMsgType = 3
	chatMsgTypeGroup   chatMsgType = 8
	// chatMsgTypeAnnounce = 2
)

type chatMsg struct {
	XMLName xml.Name `xml:"ChatMsg"`
	Type    uint     `xml:",attr"`
	From    string
	Message string `xml:"Msg"`
}

type messageDocument struct {
	Timestamp time.Time
	From      string
	Type      string
	Message   string
}

func ProcessMessage(body string) {
	msg := chatMsg{}
	err := xml.Unmarshal([]byte(body), &msg)
	if err != nil {
		return
	}
	fmt.Printf("%#v\n", msg)
}
