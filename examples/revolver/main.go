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
	"math/big"
	"sync"
	"time"

	"github.com/dfinity/go-dfinity-p2p/client"
	"github.com/multiformats/go-multiaddr"
	"github.com/whyrusleeping/go-logging"
	"golang.org/x/crypto/sha3"
)

func main() {

	// Parse command-line arguments.
	argClients := flag.Int("clients", 1, "Number of clients.")
	argDial := flag.String("dial", "", "Address of seed node.")
	argDisableNATPortMap := flag.Bool("disable-nat", false, "Disable port-mapping in NAT devices?")
	argKBucketSize := flag.Int("k-bucket-size", 8, "K-bucket size.")
	argLogLevel := flag.String("log-level", "NOTICE", "Log level.")
	argListen := flag.String("listen", "0.0.0.0", "IP address to listen on.")
	argPort := flag.Int("port", 4000, "Port number to listen on.")
	argRandomSeed := flag.String("random-seed", "", "32-byte hex-encoded random seed.")
	argSampleSize := flag.Int("sample-size", 4, "Number of peers to distribute per request.")
	argStreams := flag.Int("streams", 8, "Number of streams per client.")
	flag.Parse()

	// Create logger.
	logger := logging.MustGetLogger("main")

	// Set log level.
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

	// Set seed node.
	var seedNodes []multiaddr.Multiaddr
	if len(*argDial) != 0 {
		seedAddress, err := multiaddr.NewMultiaddr(*argDial)
		if err != nil {
			logger.Critical("Cannot create multiaddr", err)
			return
		}
		seedNodes = []multiaddr.Multiaddr{seedAddress}
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

		configs[i].DisableNATPortMap = *argDisableNATPortMap
		configs[i].KBucketSize = *argKBucketSize
		configs[i].ListenIP = *argListen
		configs[i].ListenPort = uint16(*argPort + i)
		configs[i].LogLevel = logLevel
		configs[i].RandomSeed = *argRandomSeed
		configs[i].SampleSize = *argSampleSize
		configs[i].SeedNodes = seedNodes
		configs[i].StreamstoreCapacity = *argStreams

		if len(*argRandomSeed) == 0 {
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
		go launch(configs[i], group, logger)
	}

	// Wait for clients to complete.
	group.Wait()

}

func launch(config *p2p.Config, group *sync.WaitGroup, logger *logging.Logger) {

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
	go send(client, logger)
	go receive(client, logger)

	// Hang forever.
	select {}

}

func send(client p2p.Client, logger *logging.Logger) {

	for {

		// Sleep for five seconds.
		time.Sleep(5 * time.Second)

		// Generate a random artifact.
		n, err := rand.Int(rand.Reader, big.NewInt(4096))
		if err != nil {
			logger.Warning("Cannot generate random integer", err)
			break
		}
		data := make([]byte, n.Int64())
		_, err = rand.Read(data)
		if err != nil {
			logger.Warning("Cannot generate random data", err)
			break
		}

		// Timestamp the artifact.
		timestamp := make([]byte, 8)
		binary.BigEndian.PutUint64(timestamp, uint64(time.Now().UnixNano()))
		data = append(data, timestamp...)

		// Send the artifact and timestamp.
		hash := sha3.Sum256(data)
		logger.Infof(
			"Sending %d-byte artifact with hash <%s>",
			len(data),
			hex.EncodeToString(hash[:4]),
		)
		client.Send() <- data

	}

}

func receive(client p2p.Client, logger *logging.Logger) {

	for {

		// Receive an artifact.
		data := <-client.Receive()
		hash := sha3.Sum256(data)

		// Record the latency.
		var latency time.Duration
		if len(data) >= 8 {
			latency = time.Since(time.Unix(0, int64(binary.BigEndian.Uint64(data[len(data)-8:]))))
		}
		logger.Infof(
			"Receiving %d-byte artifact with hash <%s> and latency %s",
			len(data),
			hex.EncodeToString(hash[:4]),
			latency,
		)

		// Broadcast the artifact.
		client.Send() <- data

	}

}
