// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmpp

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redbluescreen/sbrwxmpp/cmdhook"
	"github.com/redbluescreen/sbrwxmpp/config"
	"github.com/redbluescreen/sbrwxmpp/db"
	"github.com/redbluescreen/sbrwxmpp/tls"
	xmlstream "github.com/redbluescreen/sbrwxmpp/xmlstream2"
)

type XmppRoom struct {
	JID     string
	Members []*XmppClient
}

func (r *XmppRoom) RouteMessage(msg xmlstream.Element) {
	nick := strings.Split(msg.GetAttr("from"), "@")[0]
	msg.SetAttr("from", r.JID+"/"+nick)
	for _, member := range r.Members {
		msg.SetAttr("to", member.JID)
		member.SendXML(msg)
	}
}

func (r *XmppRoom) AddMember(c *XmppClient) {
	for _, member := range r.Members {
		if member == c {
			return
		}
	}
	r.Members = append(r.Members, c)
}

func (r *XmppRoom) RemoveMember(c *XmppClient) {
	for _, member := range r.Members {
		str := "<presence from='%v' to='%v' type='unavailable'>" +
			"<x xmlns='http://jabber.org/protocol/muc#user'>" +
			"<item affiliation='member' role='none'/>"
		if member == c {
			str += "<status code='110'/>"
		}
		str += "</x></presence>"
		nick := strings.Split(c.JID, "@")[0]
		member.Write(fmt.Sprintf(str, XMLEscape(r.JID+"/"+nick), XMLEscape(member.JID)))
	}
	for i, member := range r.Members {
		if member == c {
			j := len(r.Members) - 1
			r.Members[i] = r.Members[j]
			r.Members[j] = nil
			r.Members = r.Members[:j]
			return
		}
	}
}

type XmppServer struct {
	sync.Mutex
	Clients []*XmppClient
	Rooms   []*XmppRoom
	Logger  *log.Logger
	Config  *config.Config
	DB      *db.DB
}

func (s *XmppServer) Run(ln net.Listener, tlsConfig *tls.Config) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		clogger := log.New(os.Stderr, "[unknown] ", log.LstdFlags)
		clogger.Println("Accepted TCP connection")

		cl := &XmppClient{
			tcpConn:   conn,
			tlsConfig: tlsConfig,
			logger:    clogger,
			server:    s,
			streamEnd: make(chan struct{}),
			webhook: &cmdhook.CmdHook{
				Client: &http.Client{Timeout: 1 * time.Second},
				Config: &s.Config.Webhook,
			},
		}
		go cl.HandleConnection()
	}
}

func (s *XmppServer) RouteMessage(msg xmlstream.Element) {
	s.Logger.Println("Routing message")
	to := msg.GetAttr("to")
	if msg.GetAttr("type") == "groupchat" {
		s.Lock()
		for _, room := range s.Rooms {
			if strings.EqualFold(room.JID, to) {
				s.Logger.Println("Routing to room " + to)
				room.RouteMessage(msg)
				break
			}
		}
		s.Unlock()
	}
	s.Lock()
	for _, client := range s.Clients {
		if jidMatches(to, client.JID) {
			s.Logger.Println("Routing to " + to)
			client.SendXML(msg)
			break
		}
	}
	s.Unlock()
}

func (s *XmppServer) AddClient(c *XmppClient) {
	s.Lock()
	defer s.Unlock()
	for _, client := range s.Clients {
		if client == c {
			return
		}
	}
	s.Clients = append(s.Clients, c)
}

func (s *XmppServer) RemoveClient(c *XmppClient) {
	s.Lock()
	for _, room := range s.Rooms {
		room.RemoveMember(c)
	}
	for i, member := range s.Clients {
		if member == c {
			j := len(s.Clients) - 1
			s.Clients[i] = s.Clients[j]
			s.Clients[j] = nil
			s.Clients = s.Clients[:j]
			break
		}
	}
	s.Unlock()
}

func jidMatches(a, b string) bool {
	if strings.Contains(a, "/") {
		return a == b
	}
	jidParts := strings.Split(b, "/")
	return a == jidParts[0]
}

func BareJidMatch(a, b string) bool {
	pa := strings.Split(a, "/")
	pb := strings.Split(b, "/")
	return strings.EqualFold(pa[0], pb[0])
}
