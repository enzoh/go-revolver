/**
 * File        : main.go
 * Description : Example client.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"os"
	"sync"
	"time"

	"github.com/dfinity/go-revolver/artifact"
	"github.com/dfinity/go-revolver/p2p"
	"github.com/enzoh/go-logging"
)

func main() {

	// Parse command-line arguments.
	argAnalyticsInterval := flag.Duration("analytics-interval", 10*time.Second, "Time between analytics reports.")
	argAnalyticsURL := flag.String("analytics-url", "http://127.0.0.1:8080/report", "URL to submit analytics reports.")
	argClients := flag.Int("clients", 1, "Number of clients.")
	argDisableNATPortMap := flag.Bool("disable-nat", false, "Disable port-mapping in NAT devices?")
	argIP := flag.String("ip", "0.0.0.0", "IP address to listen on.")
	argKBucketSize := flag.Int("k-bucket-size", 8, "Kademlia bucket size.")
	argLevel := flag.String("log-level", "debug", "Log level.")
	argPort := flag.Uint("port", 0, "Port number to listen on.")
	argRandomSeed := flag.String("random-seed", "", "32-byte hex-encoded random seed.")
	argSeedNode := flag.String("seed-node", "", "Address of seed node.")
	flag.Parse()

	// Create a logger.
	logger := logging.MustGetLogger("main")
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	format := "\033[0;37m%{time:15:04:05.000} %{color}%{level} \033[0;34m[%{module}] \033[0m%{message} \033[0;37m%{shortfile}\033[0m"
	logging.SetBackend(logging.AddModuleLevel(backend))
	logging.SetFormatter(logging.MustStringFormatter(format))

	// Set the log level.
	level, err := logging.LogLevel(*argLevel)
	if err != nil {
		logger.Critical("Invalid log level:", err)
		return
	}
	logging.SetLevel(level, "main")
	logging.SetLevel(level, "p2p")
	logging.SetLevel(level, "streamstore")

	// Create the client configurations.
	configs := make([]*p2p.Config, *argClients)
	for i := 0; i < *argClients; i++ {
		configs[i] = p2p.DefaultConfig()
		configs[i].AnalyticsInterval = *argAnalyticsInterval
		configs[i].AnalyticsURL = *argAnalyticsURL
		configs[i].DisableNATPortMap = *argDisableNATPortMap
		configs[i].IP = *argIP
		configs[i].KBucketSize = *argKBucketSize
		configs[i].LogLevel = *argLevel
		configs[i].Port = uint16(*argPort + uint(i))
		if *argSeedNode != "" {
			configs[i].SeedNodes = []string{*argSeedNode}
		}
	}

	// Set the random seed.
	if *argClients > 0 {
		configs[0].RandomSeed = *argRandomSeed
	}

	// Lanuch the clients.
	group := &sync.WaitGroup{}
	group.Add(*argClients)
	for i := 0; i < *argClients; i++ {
		go launch(configs[i], group, logger)
	}

	// Wait for the clients to complete.
	group.Wait()

}

func launch(config *p2p.Config, group *sync.WaitGroup, logger *logging.Logger) {

	// Decrement the wait group counter on exit.
	defer group.Done()

	// Create a client.
	client, shutdown, err := config.New()
	if err != nil {
		logger.Error("Cannot create p2p client:", err)
		return
	}
	defer shutdown()

	// Send and receive artifacts.
	go send(client, logger)
	go receive(client, logger)

	// Hang forever.
	select {}

}

func send(client p2p.Client, logger *logging.Logger) {

	for {

		time.Sleep(time.Second)

		data := make([]byte, 32)
		_, err := rand.Read(data)
		if err != nil {
			logger.Error("Cannot generate random bytes:", err)
			continue
		}

		object, err := artifact.FromBytes(data, false)
		if err != nil {
			logger.Error("Cannot create artifact:", err)
			continue
		}

		checksum := object.Checksum()
		logger.Debugf(
			"Sending %d byte artifact with checksum %s",
			object.Size(),
			hex.EncodeToString(checksum[:4]),
		)

		client.Send(object)

	}

}

func receive(client p2p.Client, logger *logging.Logger) {

	for {

		object := client.Receive()

		checksum := object.Checksum()
		logger.Debugf(
			"Receiving %d byte artifact with checksum %s and latency %s",
			object.Size(),
			hex.EncodeToString(checksum[:4]),
			time.Since(object.Timestamp()),
		)

		client.Send(object)

	}

}
