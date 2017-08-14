/**
 * File        : main.go
 * Description : Example client.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"sync"
	"time"

	"github.com/dfinity/go-dfinity-p2p/artifact"
	"github.com/dfinity/go-dfinity-p2p/client"
	"github.com/multiformats/go-multiaddr"
	"github.com/whyrusleeping/go-logging"
)

func main() {

	// Parse command-line arguments.
	argAnalyticsInterval := flag.Duration("analytics-interval", 10*time.Second, "Time between analytics reports.")
	argAnalyticsURL := flag.String("analytics-url", "http://127.0.0.1:8080/report", "URL to send analytics reports.")
	argBucketSize := flag.Int("bucket-size", 16, "Size of Kademlia buckets.")
	argClients := flag.Int("clients", 1, "Number of clients.")
	argConnections := flag.Int("connections", 8, "Number of connections per client.")
	argDial := flag.String("dial", "", "Address of seed node.")
	argDisableAnalytics := flag.Bool("disable-analytics", false, "Disable analytics?")
	argDisableNATPortMap := flag.Bool("disable-nat", false, "Disable port-mapping in NAT devices?")
	argIP := flag.String("ip", "0.0.0.0", "IP address to listen on.")
	argLogLevel := flag.String("log-level", "INFO", "Log level.")
	argPort := flag.Int("port", 4000, "Port number to listen on.")
	argRandomSeed := flag.String("random-seed", "", "32-byte hex-encoded random seed.")
	argReceiveOnly := flag.Bool("receive-only", false, "Only receive and rebroadcast artifacts?")
	argSampleSize := flag.Int("sample-size", 6, "Number of peers to distribute per request.")
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

	// Set a seed node.
	var seedNodes []multiaddr.Multiaddr
	if *argDial != "" {
		address, err := multiaddr.NewMultiaddr(*argDial)
		if err != nil {
			logger.Critical("Cannot create multiaddr", err)
			return
		}
		seedNodes = []multiaddr.Multiaddr{address}
	}

	// Create client configurations.
	var err error
	configs := make([]*p2p.Config, *argClients)
	for i := 0; i < *argClients; i++ {

		configs[i], err = p2p.DefaultConfig()
		if err != nil {
			logger.Critical("Cannot create client configuration", err)
			return
		}

		configs[i].AnalyticsIterationInterval = *argAnalyticsInterval
		configs[i].AnalyticsURL = *argAnalyticsURL
		configs[i].DisableAnalytics = *argDisableAnalytics
		configs[i].DisableNATPortMap = *argDisableNATPortMap
		configs[i].KBucketSize = *argBucketSize
		configs[i].ListenIP = *argIP
		configs[i].ListenPort = uint16(*argPort + i)
		configs[i].LogLevel = logLevel
		configs[i].RandomSeed = *argRandomSeed
		configs[i].SampleSize = *argSampleSize
		configs[i].SeedNodes = seedNodes
		configs[i].StreamstoreCapacity = *argConnections

		if *argRandomSeed == "" {
			data := make([]byte, 32)
			_, err = rand.Read(data)
			if err != nil {
				logger.Critical("Cannot generate random data", err)
				return
			}
			configs[i].RandomSeed = hex.EncodeToString(data)
		}

	}

	// Lanuch clients.
	group := &sync.WaitGroup{}
	group.Add(*argClients)
	for i := 0; i < *argClients; i++ {
		go launch(configs[i], group, logger, *argReceiveOnly)
	}

	// Wait for clients to complete.
	group.Wait()

}

func launch(config *p2p.Config, group *sync.WaitGroup, logger *logging.Logger, receiveOnly bool) {

	// Decrement the wait group counter on exit.
	defer group.Done()

	// Create a p2p client.
	client, shutdown, err := config.New()
	if err != nil {
		logger.Critical("Cannot create p2p client", err)
		return
	}
	defer shutdown()

	// Send and receive artifacts.
	if !receiveOnly {
		go send(client, logger)
	}
	go receive(client, logger)

	// Hang forever.
	select {}

}

func send(client p2p.Client, logger *logging.Logger) {

	for {

		// Wait.
		time.Sleep(time.Minute)

		// Create a 1Mb artifact and timestamp it.
		size := 1000000
		data := make([]byte, size)
		rand.Read(data)
		binary.BigEndian.PutUint64(data, uint64(time.Now().UnixNano()))
		object := artifact.FromBytes(data)

		// Log the details of this artifact.
		checksum := object.Checksum()
		code := hex.EncodeToString(checksum[:4])
		logger.Infof("Sending %d byte artifact with checksum %s", size, code)

		// Broadcast the artifact.
		client.Send() <- object

	}

}

func receive(client p2p.Client, logger *logging.Logger) {

	for {

		// Receive an artifact and extract data from it.
		object := <-client.Receive()
		data, err := artifact.ToBytes(object)
		if err != nil {
			logger.Warning("Cannot extract data from artifact", err)
			continue
		}

		// Log the details of this artifact.
		size := object.Size()
		checksum := object.Checksum()
		code := hex.EncodeToString(checksum[:4])
		latency := time.Duration(0)
		if len(data) >= 8 {
			latency = time.Since(time.Unix(0, int64(binary.BigEndian.Uint64(data[:8]))))
		}
		logger.Infof("Receiving %d byte artifact with checksum %s and latency %s", size, code, latency)

		// Create an artifact from the data.
		object = artifact.FromBytes(data)

		// Broadcast the artifact.
		client.Send() <- object

	}

}
