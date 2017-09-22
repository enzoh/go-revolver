/**
 * File        : main.go
 * Description : Network topology inspector.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Report struct {
	Addrs     []string
	ClusterID int
	Network   string
	NodeID    string
	Peers     int
	ProcessID int
	Streams   []string
	Timestamp int64
	UserData  string
	Version   string
}

func main() {

	log.Println("Parsing command-line arguments ...")
	port := flag.Uint64("port", 8080, "Port that the network topology inspector listens on.")
	ttl := flag.Duration("ttl", 90*time.Second, "Time until analytics reports are discarded.")
	flag.Parse()

	reports := make(map[string]Report)
	lock := &sync.Mutex{}

	log.Println("Registering request handlers ...")
	http.HandleFunc("/", index())
	http.HandleFunc("/graph", graph(reports, lock, *ttl))
	http.HandleFunc("/report", report(reports, lock))

	log.Println("Registering termination signal handler ...")
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Listening on port %d ...\n", *port)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
		if err != nil {
			log.Printf("Cannot listen on port %d: \033[1;31m%s\033[0m\n", *port, err.Error())
			shutdown <- syscall.SIGTERM
		}
	}()

	sig := <-shutdown
	log.Printf("Receiving termination signal %s ...\n", sig.String())

}

// Serve the index page.
func index() http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Write(HTML)
	}

}

// Serve the graph data.
func graph(reports map[string]Report, lock *sync.Mutex, ttl time.Duration) http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {

		lock.Lock()
		defer lock.Unlock()

		nodes := make([]map[string]interface{}, 0)
		links := make([]map[string]interface{}, 0)

		threshold := time.Now().Add(-ttl).Unix()

		for id, report := range reports {

			if report.Timestamp < threshold {
				delete(reports, id)
				continue
			}

			nodes = append(nodes, map[string]interface{}{
				"Addrs":     report.Addrs,
				"ClusterID": report.ClusterID,
				"Network":   report.Network,
				"NodeID":    report.NodeID,
				"Peers":     report.Peers,
				"ProcessID": report.ProcessID,
				"Streams":   len(report.Streams),
				"Timestamp": report.Timestamp,
				"UserData":  report.UserData,
				"Version":   report.Version,
			})

			for i := range report.Streams {
				links = append(links, map[string]interface{}{
					"source": report.NodeID,
					"target": report.Streams[i],
				})
			}

		}

		data, err := json.Marshal(map[string]interface{}{
			"nodes": nodes,
			"links": links,
		})
		if err != nil {
			log.Printf("Cannot encode graph: \033[1;31m%s\033[0m\n", err.Error())
			http.Error(resp, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}

		header := resp.Header()
		header.Set("Access-Control-Allow-Origin", "http://localhost:8080")
		header.Set("Content-Type", "application/json")

		resp.Write(data)

	}

}

// Report connections.
func report(reports map[string]Report, lock *sync.Mutex) http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {

		lock.Lock()
		defer lock.Unlock()

		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()

		var report Report
		err := decoder.Decode(&report)
		if err != nil {
			log.Printf("Cannot decode report: \033[1;31m%s\033[0m\n", err.Error())
			http.Error(resp, "400 Bad Request", http.StatusBadRequest)
			return
		}

		report.Timestamp = time.Now().Unix()
		reports[report.NodeID] = report

	}

}
