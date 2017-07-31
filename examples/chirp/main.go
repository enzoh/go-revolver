/**
 * File        : main.go
 * Description : Chirp client.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package main

import (
	"database/sql"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dfinity/go-dfinity-p2p/client"
	_ "github.com/lib/pq"
	"github.com/gorilla/websocket"
	"github.com/multiformats/go-multiaddr"
	"github.com/whyrusleeping/go-logging"
)

type Chirp struct {
	Data string
	Nonce int
	Username string
}

func main() {

	// Parse command-line arguments.
	argDBHost      := flag.String("db-host", "127.0.0.1", "IP address or hostname of the database server.")
	argDBName      := flag.String("db-name", "chirp", "Name of the database to connect to.")
	argDBPassword  := flag.String("db-password", "chirp", "Password for accessing the database.")
	argDBPort      := flag.Uint64("db-port", 5432, "Port that the database server listens on.")
	argDBUsername  := flag.String("db-username", "chirp", "Username for accessing the database.")
	argDisableNAT  := flag.Bool("disable-nat", false, "Disable port-mapping in NAT devices?")
	argDisableUI   := flag.Bool("disable-ui", false, "Disable the UI server?")
	argEnableDB    := flag.Bool("enable-db", false, "Enable database for history retention?")
	argIndex       := flag.String("index", "", "Path to index page.")
	argLogLevel    := flag.String("log-level", "INFO", "Log level.")
	argPort        := flag.Int("port", 0, "Port that the p2p client listens on.")
	argRandomSeed  := flag.String("random-seed", "", "32-byte hex-encoded random seed.")
	argSeedAddress := flag.String("seed-address", "", "Address of seed node.")
	argUIPort         := flag.Int("ui-port", 3000, "Port that the UI server listens on.")
	flag.Parse()

	// Create a logger.
	logger := logging.MustGetLogger("main")

	// Set the log level.
	var logLevel logging.Level
	switch *argLogLevel {
	case "CRITICAL":
		logLevel = logging.CRITICAL
	case "ERROR":
		logLevel = logging.ERROR
	case "WARNING":
		logLevel = logging.WARNING
	case "NOTICE":
		logLevel = logging.NOTICE
	case "INFO":
		logLevel = logging.INFO
	default:
		logLevel = logging.DEBUG
	}
	logging.SetLevel(logLevel, "main")

	// Set the random seed.
	if *argRandomSeed == "" {
		data := make([]byte, 32)
		_, err := rand.Read(data)
		if err != nil {
			logger.Critical("Cannot generate random data", err)
			return
		}
		*argRandomSeed = hex.EncodeToString(data)
	}

	// Set the seed nodes.
	var seedNodes []multiaddr.Multiaddr
	if *argSeedAddress == "" {
		seedNodes = make([]multiaddr.Multiaddr, 2)
		seedNodes[0], _ = multiaddr.NewMultiaddr("/ip4/35.157.1.121/tcp/4000/ipfs/QmdMM3Nftnn7ih8g9buzgqxMStoC5mzNNA5krk7W9gqpLE")
		seedNodes[1], _ = multiaddr.NewMultiaddr("/ip4/13.113.68.112/tcp/4000/ipfs/QmRrqFPYzTJ9xtgCoPKMxHESF9fwH33cbADTcaEJc5UabX")
	} else {
		var err error
		seedNodes = make([]multiaddr.Multiaddr, 1)
		seedNodes[0], err = multiaddr.NewMultiaddr(*argSeedAddress)
		if err != nil {
			logger.Critical("Cannot parse seed address", err)
			return
		}
	}

	// Generate a random port.
	if *argPort == 0 {
		noise, err := rand.Int(rand.Reader, big.NewInt(16383))
		if err != nil {
			logger.Critical("Cannot generate random port", err)
			return
		}
		*argPort = int(noise.Int64()) + 49152
	}

	// Create a networking configuration.
	config, err := p2p.DefaultConfig()
	if err != nil {
		logger.Critical("Cannot create networking configuration", err)
		return
	}
	config.DisableNATPortMap = *argDisableNAT
	config.ListenPort = uint16(*argPort)
	config.LogLevel = logLevel
	config.RandomSeed = *argRandomSeed
	config.SeedNodes = seedNodes

	// Create a networking client.
	logger.Info("Creating networking client")
	client, shutdown, err := config.New()
	if err != nil {
		logger.Critical("Cannot create networking client", err)
		return
	}
	defer shutdown()

	if *argEnableDB {

		// Create a database connection.
		logger.Info("Creating database connection")
		db, err := sql.Open(
			"postgres",
			fmt.Sprintf(
				"postgres://%s:%s@%s:%d/%s?sslmode=disable",
				argDBUsername,
				argDBPassword,
				argDBHost,
				argDBPort,
				argDBName,
			),
		)
		if err != nil {
			logger.Critical("Cannot create database connection", err)
			return
		}
		defer db.Close()

		// Test the database connection.
		err = db.Ping()
		if err != nil {
			logger.Critical("Cannot ping database")
			return
		}

	}

	// Create a channel that waits for a termination signal.
	wait := make(chan os.Signal, 1)
	signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)

	if *argDisableUI {

		go func() {
			for {
				data := <-client.Receive()
				chirp := Chirp{}
				err := json.Unmarshal(data, &chirp)
				if err != nil {
					logger.Warning("Cannot decode chirp", err)
					continue
				}
				logger.Debugf("Receiving chirp: %#v", chirp)
				client.Send() <-data
			}
		}()

	} else {

		// Load the HTML content.
		var html []byte
		if *argIndex == "" {
			html = []byte(UI)
		} else {
			html, err = ioutil.ReadFile(*argIndex)
			if err != nil {
				logger.Error("Cannot read file", *argIndex, err)
			}
		}

		// Create a thread safe counter.
		var counter int
		lock := &sync.Mutex{}

		// Register the request handlers.
		http.HandleFunc("/", index(html))
		http.HandleFunc("/ws", ws(client, counter, lock, logger))

		// Start the user interface.
		logger.Info("Starting user interface")
		go func() {
			err := http.ListenAndServe(fmt.Sprintf(":%d", *argUIPort), nil)
			if err != nil {
				logger.Critical("Cannot start user interface", err)
				wait <-syscall.SIGTERM
			}
		}()

	}

	// Wait for a termination signal.
	sig := <-wait
	logger.Info("Detecting signal", sig.String())

}

// Serve the index page.
func index(html []byte) http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Write(html)
	}

}

// Exchange artifacts via websocket.
func ws(client p2p.Client, counter int, lock *sync.Mutex, logger *logging.Logger) http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {

		lock.Lock()
		defer lock.Unlock()

		if counter != 0 {
			http.Error(resp, "Forbidden", http.StatusForbidden)
			return
		}

		if req.Header.Get("Origin") != "http://" + req.Host {
			http.Error(resp, "Forbidden", http.StatusForbidden)
			return
		}

		conn, err := websocket.Upgrade(resp, req, resp.Header(), 2048, 2048)
		if err != nil {
			http.Error(resp, "Bad Request", http.StatusBadRequest)
			return
		}

		counter++
		notify := make(chan struct {}, 1)

		go func() {

			Sender:
			for {
				chirp := Chirp{}
				err := conn.ReadJSON(&chirp)
				if err != nil {
					logger.Debug("Cannot read chirp", err)
					break Sender
				}
				logger.Debugf("Sending chirp: %#v", chirp)
				data, err := json.Marshal(chirp)
				if err != nil {
					logger.Warning("Cannot encode chirp", err)
					break Sender
				}
				client.Send() <-data
			}

			notify <-struct {}{}

		}()

		go func() {

			Receiver:
			for {
				select {
				case <-notify:
					break Receiver
				case data := <-client.Receive():
					chirp := Chirp{}
					err = json.Unmarshal(data, &chirp)
					if err != nil {
						logger.Warning("Cannot decode chirp", err)
						break Receiver
					}
					logger.Debugf("Receiving chirp: %#v", chirp)
					err := conn.WriteJSON(chirp)
					if err != nil {
						logger.Debug("Cannot write chirp", err)
						break Receiver
					}
					client.Send() <-data
				}
			}

			lock.Lock()
			defer lock.Unlock()
			counter--

		}()

	}

}
