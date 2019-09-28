// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strings"

	bolt "go.etcd.io/bbolt"

	"github.com/redbluescreen/sbrwxmpp/api"
	"github.com/redbluescreen/sbrwxmpp/certgen"
	pconfig "github.com/redbluescreen/sbrwxmpp/config"
	"github.com/redbluescreen/sbrwxmpp/db"
	"github.com/redbluescreen/sbrwxmpp/tls"
	"github.com/redbluescreen/sbrwxmpp/xmpp"
)

var defaultConfig = `# Remove localhost to make the server listen publicly
addr = "localhost:5222"
# cert = "cert.pem"
# certkey = "key.pem"

# Change domain to the public address of the server
domain = "localhost"

[api]
addr = "localhost:8087"
key = "<<APIKEY>>"`

func main() {
	config, err := pconfig.LoadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("No configuration found, generating")
			genConfig := strings.ReplaceAll(defaultConfig, "<<APIKEY>>", xmpp.RandomStringSecure(32))
			err = ioutil.WriteFile("sbrwxmpp.toml", []byte(genConfig), os.ModePerm)
			if err != nil {
				log.Fatalf("Failed to save config: %v\n", err)
			}
			config, err = pconfig.LoadConfig()
			if err != nil {
				log.Fatalf("Failed to read config: %v\n", err)
			}
		} else {
			log.Fatalf("Failed to read config: %v\n", err)
		}
	}

	// TODO: logger init based on log config
	logger := log.New(os.Stderr, "[server] ", log.LstdFlags)

	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		logger.Fatalf("Failed to listen: %v\n", err)
	}

	if config.Cert == "" || config.CertKey == "" {
		logger.Print("No certificate specified, using selfsigned certificate")
		config.Cert = path.Join("sbrwxmpp-certs", config.Domain+".crt")
		config.CertKey = path.Join("sbrwxmpp-certs", config.Domain+".key")
		if _, err := os.Stat(config.Cert); os.IsNotExist(err) {
			logger.Printf("No certificate found for %v, generating new", config.Domain)
			os.Mkdir("sbrwxmpp-certs", 0600)
			err = certgen.GenerateCertificate("sbrwxmpp-certs", config.Domain)
			if err != nil {
				logger.Fatalf("Failed to generate certificate: %v\n", err)
			}
		}
	}

	cert, err := tls.LoadX509KeyPair(config.Cert, config.CertKey)
	if err != nil {
		logger.Fatalf("Failed to load certs: %v\n", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// Just in case, game doesn't like data being split
		// into multiple records
		DynamicRecordSizingDisabled: true,
	}

	bdb, err := bolt.Open("sbrwxmpp.db", 0600, nil)
	if err != nil {
		logger.Fatalf("Failed to open DB: %v\n", err)
	}
	db := &db.DB{bdb}
	err = db.Initialize()
	if err != nil {
		logger.Fatalf("Failed to initialize DB: %v\n", err)
	}

	server := &xmpp.XmppServer{
		Logger: logger,
		DB:     db,
		Config: config,
	}

	apiSrv := api.Server{
		XMPP:   server,
		DB:     db,
		Config: config,
		Logger: log.New(os.Stderr, "[api] ", log.LstdFlags),
	}
	go apiSrv.Run()
	logger.Print("Server running!")
	server.Run(ln, tlsConfig)
}
