package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"time"

	"github.com/dfinity/go-revolver/go-revolver-p2p"
	"github.com/enzoh/go-logging"

	"gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
)

func readLevel(name string) logging.Level {
	switch name {
	case "CRITICAL":
		return logging.CRITICAL
	case "ERROR":
		return logging.ERROR
	case "WARNING":
		return logging.WARNING
	case "NOTICE":
		return logging.NOTICE
	case "INFO":
		return logging.INFO
	default:
		return logging.DEBUG
	}
}

func main() {

	argLevel := flag.String("log-level", "INFO", "Log level.")
	argRandomSeed := flag.String("random-seed", "", "32-byte hex-encoded random seed.")
	argSeedNode := flag.String("seed-node", "", "Address of seed node.")
	flag.Parse()

	logger := logging.MustGetLogger("main")
	level := readLevel(*argLevel)
	logging.SetLevel(level, "main")

	config, err := p2p.DefaultConfig()
	if err != nil {
		logger.Critical("Cannot create networking configuration", err)
		return
	}

	config.DisableAnalytics = true
	config.LogLevel = level
	config.Port = 0

	if *argRandomSeed == "" {
		seed := make([]byte, 32)
		_, err := rand.Read(seed)
		if err != nil {
			logger.Critical("Cannot generate random seed", err)
			return
		}
		config.RandomSeed = hex.EncodeToString(seed)
	}

	if *argSeedNode != "" {
		node, err := multiaddr.NewMultiaddr(*argSeedNode)
		if err != nil {
			logger.Critical("Cannot parse address of seed node", err)
			return
		}
		config.SeedNodes = []multiaddr.Multiaddr{node}
	}

	client, shutdown, err := config.New()
	if err != nil {
		logger.Critical("Cannot create networking client", err)
		return
	}
	defer shutdown()

	go func() {
		for {
			artifact := make([]byte, 32)
			_, err := rand.Read(artifact)
			if err != nil {
				logger.Critical("Cannot generate random artifact", err)
				break
			}
			hash := sha256.Sum256(artifact)
			logger.Debugf("Sending artifact <%s>", hex.EncodeToString(hash[:])[0:8])
			client.Send() <- artifact
			time.Sleep(time.Second)
		}
	}()

	go func() {
		for {
			artifact := <-client.Receive()
			hash := sha256.Sum256(artifact)
			logger.Debugf("Receiving artifact <%s>", hex.EncodeToString(hash[:])[0:8])
		}
	}()

	select {}
}
