// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package xmpp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redbluescreen/sbrwxmpp/chatlog"
	"github.com/redbluescreen/sbrwxmpp/cmdhook"
	"github.com/redbluescreen/sbrwxmpp/tls"
	xmlstream "github.com/redbluescreen/sbrwxmpp/xmlstream2"
)

type XmppClient struct {
	tcpConn       net.Conn
	tlsConn       *tls.Conn
	tlsConfig     *tls.Config
	authenticated bool
	stream        *xmlstream.ElementStream
	logger        *log.Logger
	JID           string
	server        *XmppServer
	streamEnd     chan struct{}
	streamClosed  uint32
	webhook       *cmdhook.CmdHook
}

func (c *XmppClient) closeConn() {
	if c.tlsConn != nil {
		c.tlsConn.Close()
	} else {
		c.tcpConn.Close()
	}
}

func (c *XmppClient) CloseError(err string) {
	c.write("<stream:error>" + err + "</stream:error>")
	c.Close()
}

func (c *XmppClient) Close() {
	atomic.StoreUint32(&c.streamClosed, 1)
	c.write("</stream:stream>")
	select {
	case <-c.streamEnd:
	case <-time.After(1 * time.Second):
		c.closeConn()
	}
}

func (c *XmppClient) HandleConnection() {
	defer func() {
		c.server.RemoveClient(c)
		c.closeConn()
		c.logger.Println("Connection closed")
	}()
	stream, err := xmlstream.NewStream(c.tcpConn)
	if err != nil {
		c.logger.Printf("error creating xml stream: %v", err)
		return
	}
	c.handleRootElement(stream)
	c.stream = stream
	for {
		e, err := c.stream.NextChild()
		if err == xmlstream.NoMoreChildrenError {
			c.logger.Printf("XML stream ended")
			if atomic.LoadUint32(&c.streamClosed) != 0 {
				c.streamEnd <- struct{}{}
			} else {
				c.write("</stream:stream>")
			}
			return
		}
		if err != nil {
			c.logger.Printf("error getting next child: %v", err)
			return
		}

		err = c.handleXmlElement(e)
		if err != nil {
			c.logger.Printf("error handling element: %v", err)
			return
		}
	}
}

func (c *XmppClient) handleRootElement(e *xmlstream.ElementStream) {
	c.logger.Printf("Received stream start")
	var from string
	for _, attr := range e.Attr {
		if attr.Name.Local == "from" {
			from = attr.Value
			break
		}
	}
	c.sendStreamStart(from)
	if c.tlsConn == nil {
		c.sendPreTLSStreamFeatures()
	} else {
		c.sendPostTLSStreamFeatures()
	}
}

func (c *XmppClient) handleXmlElement(e xmlstream.Element) error {
	switch e.Name.Local {
	case "starttls":
		if c.tlsConn == nil {
			return c.doTLS()
		}
	case "iq":
		c.handleIq(e)
	case "presence":
		if c.authenticated {
			c.handlePresence(e)
		}
	case "message":
		if c.authenticated {
			c.handleMessage(e)
		}
	default:
		c.logger.Println("received unknown xml element")
		c.logger.Printf("%#v\n", e)
	}
	return nil
}

func (c *XmppClient) handleIq(e xmlstream.Element) {
	id := e.GetAttr("id")
	typ := e.GetAttr("type")
	if id == "" || (typ != "get" && typ != "set") {
		// TODO: return iq error
		c.logger.Println("iq error: id or type invalid")
		return
	}
	for _, el := range e.Children {
		if el.Name.Local == "query" && el.Name.Space == "jabber:iq:auth" {
			c.logger.Println("Received authentication IQ")
			if typ == "get" {
				s := "<iq type='result' id='%v'><query xmlns='jabber:iq:auth'><username/><password/><resource/></query></iq>"
				c.write(fmt.Sprintf(s, id))
			} else {
				uc, _ := el.GetChild("username")
				pc, _ := el.GetChild("password")
				rc, _ := el.GetChild("resource")
				user, err := c.server.DB.GetUser(uc.Text)
				if err != nil {
					c.logger.Printf("error getting user: %v", err)
					s := "<iq type='error' id='%v'><error type='cancel'>" +
						"<internal-server-error xmlns='urn:ietf:params:xml:ns:xmpp-stanzas'/>" +
						"</error></iq>"
					c.write(fmt.Sprintf(s, id))
					continue
				}
				if !bytes.Equal(user.Password, []byte(pc.Text)) {
					s := "<iq type='error' id='%v'><error code='401' type='auth'>" +
						"<not-authorized xmlns='urn:ietf:params:xml:ns:xmpp-stanzas'/>" +
						"</error></iq>"
					c.write(fmt.Sprintf(s, id))
					continue
				}
				c.JID = uc.Text + "@" + c.server.Config.Domain + "/" + rc.Text
				c.logger.Printf("JID set to %v", c.JID)
				c.server.Lock()
				for _, cl := range c.server.Clients {
					// Intentionally BareJidMatch, we don't support multiple
					// resources
					if BareJidMatch(cl.JID, c.JID) {
						cl.logger.Printf("Kicking client because of JID conflict")
						cl.CloseError("<conflict xmlns='urn:ietf:params:xml:ns:xmpp-streams'/>")
					}
				}
				c.server.Unlock()
				s := "<iq type='result' id='%v'/>"
				c.write(fmt.Sprintf(s, id))
				c.authenticated = true
			}
		} else {
			c.logger.Println("Received unknown IQ")
			s := "<iq type='error' id='%v' from='%v'><error type='cancel'><service-unavailable xmlns='urn:ietf:params:xml:ns:xmpp-stanzas'/></error></iq>"
			c.write(fmt.Sprintf(s, id, c.server.Config.Domain))
		}
	}
}

func (c *XmppClient) handlePresence(e xmlstream.Element) {
	to := e.GetAttr("to")
	typ := e.GetAttr("type")
	if to != "" {
		toBare := strings.Split(to, "/")[0]
		if typ == "" {
			c.logger.Println("Handling presence as groupchat 1.0 join")
			c.server.Lock()
			var joinedRoom *XmppRoom
			wasAdded := false
			for _, room := range c.server.Rooms {
				if strings.EqualFold(room.JID, toBare) {
					c.logger.Println("Added client to room " + toBare)
					room.AddMember(c)
					joinedRoom = room
					wasAdded = true
					break
				}
			}
			if !wasAdded {
				c.logger.Println("Created room " + toBare)
				joinedRoom = &XmppRoom{
					JID:     toBare,
					Members: []*XmppClient{c},
				}
				c.server.Rooms = append(c.server.Rooms, joinedRoom)
			}
			c.server.Unlock()
			for _, member := range joinedRoom.Members {
				str := "<presence from='%v' to='%v'>" +
					"<x xmlns='http://jabber.org/protocol/muc#user'>" +
					"<item affiliation='member' role='participant'/>"
				if member == c {
					str += "<status code='110'/>"
				}
				str += "</x></presence>"
				nick := strings.Split(member.JID, "@")[0]
				c.write(fmt.Sprintf(str, XMLEscape(toBare+"/"+nick), XMLEscape(c.JID)))
			}
			for _, member := range joinedRoom.Members {
				if member == c {
					continue
				}
				str := "<presence from='%v' to='%v'>" +
					"<x xmlns='http://jabber.org/protocol/muc#user'>" +
					"<item affiliation='member' role='participant'/></x></presence>"
				member.write(fmt.Sprintf(str, XMLEscape(to), XMLEscape(member.JID)))
			}
		}
		if typ == "unavailable" {
			c.logger.Println("Handling presence as groupchat 1.0 leave")
			c.server.Lock()
			for _, room := range c.server.Rooms {
				if room.JID == toBare {
					room.RemoveMember(c)
					c.logger.Println("Removed client from room " + toBare)
					break
				}
			}
			c.server.Unlock()
		}
	} else {
		c.logger.Println("Adding client to available clients")
		c.server.AddClient(c)
	}
}

func (c *XmppClient) handleMessage(e xmlstream.Element) {
	c.logger.Printf("Handling message: %#v\n", e)
	body, ok := e.GetChild("body")
	if ok {
		chatlog.ProcessMessage(body.Text)
		if c.webhook.ProcessMessage(c.JID, body.Text) {
			return
		}
	}
	e.SetAttr("from", c.JID)
	c.server.RouteMessage(e)
}

func (c *XmppClient) write(str string) {
	c.logger.Printf("SEND: %v\n", str)
	if c.tlsConn != nil {
		c.tlsConn.Write([]byte(str))
	} else {
		c.tcpConn.Write([]byte(str))
	}
}

func (c *XmppClient) doTLS() error {
	c.write("<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>")
	c.tlsConn = tls.Server(c.tcpConn, c.tlsConfig)
	stream, err := xmlstream.NewStream(c.tlsConn)
	if err != nil {
		return fmt.Errorf("error creating xml stream: %v", err)
	}
	c.handleRootElement(stream)
	c.stream = stream
	return nil
}

func (c *XmppClient) sendStreamStart(toJid string) {
	id := RandomStringSecure(10)
	c.logger.SetPrefix("[" + id + "] ")
	t := xml.Header + "<stream:stream " +
		"from='%v' " +
		"id='%v' " +
		"version='1.0' " +
		"xml:lang='en' " +
		"xmlns='jabber:client' " +
		"xmlns:stream='http://etherx.jabber.org/streams'>"
	c.write(fmt.Sprintf(t, c.server.Config.Domain, id))
}

func (c *XmppClient) sendPreTLSStreamFeatures() {
	t := "<stream:features>" +
		"<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'><required/></starttls>" +
		"</stream:features>"
	c.write(t)
}

func (c *XmppClient) sendPostTLSStreamFeatures() {
	t := "<stream:features>" +
		"<auth xmlns='http://jabber.org/features/iq-auth'/>" +
		"</stream:features>"
	c.write(t)
}

func (c *XmppClient) Write(m string) {
	c.write(m)
}

func (c *XmppClient) SendXML(e xmlstream.Element) {
	c.write(e.AsString())
}
