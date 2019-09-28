// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package api

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/gorilla/mux"
	"github.com/redbluescreen/sbrwxmpp/config"
	"github.com/redbluescreen/sbrwxmpp/db"
	xmlstream "github.com/redbluescreen/sbrwxmpp/xmlstream2"
	"github.com/redbluescreen/sbrwxmpp/xmpp"
)

type Server struct {
	XMPP   *xmpp.XmppServer
	DB     *db.DB
	Config *config.Config
	Logger *log.Logger
}

func (s Server) Run() {
	mux := mux.NewRouter()
	mux.HandleFunc("/api/sessions", s.getSessions).Methods("GET")
	mux.HandleFunc("/api/rooms", s.getRooms).Methods("GET")
	mux.HandleFunc("/api/users/{to}/message", s.sendMessage(false)).Methods("POST")
	mux.HandleFunc("/api/rooms/{to}/message", s.sendMessage(true)).Methods("POST")
	mux.HandleFunc("/api/users", s.upsertUser).Methods("POST")
	mux.HandleFunc("/api/users/{user}", s.deleteUser).Methods("DELETE")
	mux.HandleFunc("/api/users/{user}/kick", s.kickUser).Methods("POST")
	mux.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	mux.Use(loggerMiddleware(s.Logger))
	mux.Use(authMiddleware(s.Config.API.Key))
	http.ListenAndServe(s.Config.API.Addr, mux)
}

func (s Server) getSessions(rw http.ResponseWriter, r *http.Request) {
	s.XMPP.Lock()
	sessions := make([]string, len(s.XMPP.Clients))
	for i, client := range s.XMPP.Clients {
		sessions[i] = strings.Split(client.JID, "@")[0]
	}
	s.XMPP.Unlock()
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(sessions)
}

func (s Server) getRooms(rw http.ResponseWriter, r *http.Request) {
	type roomInfo struct {
		Name    string   `json:"name"`
		Members []string `json:"members"`
	}
	s.XMPP.Lock()
	rooms := make([]roomInfo, len(s.XMPP.Rooms))
	for i, room := range s.XMPP.Rooms {
		members := make([]string, len(room.Members))
		for i, member := range room.Members {
			members[i] = strings.Split(member.JID, "@")[0]
		}
		rooms[i] = roomInfo{
			Name:    strings.Split(room.JID, "@")[0],
			Members: members,
		}
	}
	s.XMPP.Unlock()
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(rooms)
}

func (s Server) sendMessage(room bool) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		var body struct {
			From    string `json:"from"`
			Body    string `json:"body"`
			Subject string `json:"subject"`
		}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			s.Logger.Printf("error handling request: %v", err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		el := xmlstream.Element{
			Name: xml.Name{
				Local: "message",
				Space: "jabber:client",
			},
			Children: []xmlstream.Element{
				xmlstream.Element{
					Name: xml.Name{
						Local: "body",
						Space: "jabber:client",
					},
					Text: body.Body,
				},
				xmlstream.Element{
					Name: xml.Name{
						Local: "subject",
						Space: "jabber:client",
					},
					Text: body.Subject,
				},
			},
		}
		el.SetAttr("from", body.From)
		var service string
		if room {
			service = "conference." + s.Config.Domain
		} else {
			service = s.Config.Domain
		}
		el.SetAttr("to", mux.Vars(r)["to"]+"@"+service)
		if room {
			el.SetAttr("type", "groupchat")
		}
		s.XMPP.RouteMessage(el)
	}
}

func (s Server) upsertUser(rw http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		s.Logger.Printf("error handling request: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !jidNodeValid(body.Username) {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	err = s.DB.UpsertUser(db.User{
		Name:     body.Username,
		Password: []byte(body.Password),
	})
	if err != nil {
		s.Logger.Printf("error handling request: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s Server) deleteUser(rw http.ResponseWriter, r *http.Request) {
	err := s.DB.DeleteUser(mux.Vars(r)["user"])
	if err != nil {
		s.Logger.Printf("error handling request: %v", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s Server) kickUser(rw http.ResponseWriter, r *http.Request) {
	s.XMPP.Lock()
	for _, client := range s.XMPP.Clients {
		if strings.Split(client.JID, "/")[0] == mux.Vars(r)["user"]+"@"+s.Config.Domain {
			client.CloseError("<not-authorized xmlns='urn:ietf:params:xml:ns:xmpp-streams'/>")
		}
	}
	s.XMPP.Unlock()
}

func jidNodeValid(s string) bool {
	if len(s) == 0 || len(s) > 256 {
		return false
	}
	return !strings.ContainsAny(s, "\"&'/:<>@\u007F\uFFFE\uFFFF")
}
