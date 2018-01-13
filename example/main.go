package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/dfinity/go-revolver/artifact"
	"github.com/dfinity/go-revolver/p2p"
	"github.com/enzoh/go-logging"
)

func main() {

	// Parse command-line arguments.
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
	var level logging.Level
	logger := logging.MustGetLogger("main")
	level, err := logging.LogLevel(*argLevel)
	if err != nil {
		logger.Critical("Invalid log level:", err)
		return
	}
	logging.SetLevel(level, "main")
	logging.SetLevel(level, "p2p")

	// Create the client configurations.
	configs := make([]*p2p.Config, *argClients)
	for i := 0; i < *argClients; i++ {
		configs[i] = p2p.DefaultConfig()
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
		logger.Error("Cannot create p2p client:", err)
		return
	}
	defer shutdown()

	// Send and receive artifacts.
	go send(client, logger)
	go receive(client, logger)

	// Hang.
	// time.Sleep(5 * time.Minute)
	select {}

}

func send(client p2p.Client, logger *logging.Logger) {

	for {

		time.Sleep(time.Second)

		n, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint16))
		if err != nil {
			logger.Error("Cannot generate random integer:", err)
			continue
		}

		data := make([]byte, n.Int64())
		_, err = rand.Read(data)
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
		logger.Infof(
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
		logger.Infof(
			"Receiving %d byte artifact with checksum %s and latency %s",
			object.Size(),
			hex.EncodeToString(checksum[:4]),
			time.Since(object.Timestamp()),
		)

		client.Send(object)

	}

}
