package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"time"

	"github.com/dfinity/go-revolver/artifact"
	"github.com/dfinity/go-revolver/p2p"
	"github.com/enzoh/go-logging"
)

func main() {

	argLevel := flag.String("log-level", "info", "Log level.")
	argPort := flag.Uint("port", 0, "Port number to listen on.")
	argRandomSeed := flag.String("random-seed", "", "32-byte hex-encoded random seed.")
	argSeedNode := flag.String("seed-node", "", "Address of seed node.")
	flag.Parse()

	var level logging.Level
	logger := logging.MustGetLogger("main")
	level, err := logging.LogLevel(*argLevel)
	if err != nil {
		logger.Critical("Invalid log level:", err)
		return
	}
	logging.SetLevel(level, "main")

	config := p2p.DefaultConfig()
	config.DisableAnalytics = true
	config.LogLevel = *argLevel
	config.Port = (uint16(*argPort))
	config.RandomSeed = *argRandomSeed
	if *argSeedNode != "" {
		config.SeedNodes = []string{*argSeedNode}
	}

	client, shutdown, err := config.New()
	if err != nil {
		logger.Critical("Cannot create network client:", err)
		return
	}
	defer shutdown()

	for {
		if client.StreamCount() > 0 {
			break
		}
		time.Sleep(time.Second)
	}

	go func() {
		for {
			time.Sleep(time.Second)
			data := make([]byte, 32)
			_, err := rand.Read(data)
			if err != nil {
				logger.Error("Cannot generate random data:", err)
				continue
			}
			hash := sha256.Sum256(data)
			logger.Debugf("Sending artifact <%s>", hex.EncodeToString(hash[:])[0:8])
			object, err := artifact.FromBytes(data, false)
			if err != nil {
				logger.Error("Cannot create artifact:", err)
				continue
			}
			client.Send(object)
		}
	}()

	go func() {
		for {
			object := client.Receive()
			data, err := artifact.ToBytes(object)
			if err != nil {
				logger.Error("Cannot read artifact:", err)
				continue
			}
			hash := sha256.Sum256(data)
			logger.Debugf("Receiving artifact <%s>", hex.EncodeToString(hash[:])[0:8])
		}
	}()

	select {}

}
