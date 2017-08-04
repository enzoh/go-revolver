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
)

type Vertex struct {
	Client string
	Peers []string
}

func main() {

	log.Println("Parsing command-line arguments ...")
	port := flag.Uint64("port", 8080, "Port that the network topology inspector listens on.")
	flag.Parse()

	dstore := make(map[string][]string)
	lock := &sync.Mutex{}

	log.Println("Registering request handlers ...")
	http.HandleFunc("/", index())
	http.HandleFunc("/graph", graph(dstore, lock))
	http.HandleFunc("/report", report(dstore, lock))

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

	sig := <- shutdown
	log.Printf("Receiving termination signal %s ...\n", sig.String())

}

// Serve the index page.
func index() http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Write(HTML)
	}

}

// Serve the graph data.
func graph(dstore map[string][]string, lock *sync.Mutex) http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {

		lock.Lock()
		defer lock.Unlock()

		var nodes []map[string]interface {}
		var links []map[string]interface {}

		for client, peers := range dstore {
			node := make(map[string]interface {})
			node["id"] = client
			nodes = append(nodes, node)
			for i := range peers {
				link := make(map[string]interface {})
				link["source"] = client
				link["target"] = peers[i]
				links = append(links, link)
			}
		}

		object := make(map[string]interface {})
		object["nodes"] = nodes
		object["links"] = links

		js, err := json.Marshal(object)
		if err != nil {
			log.Printf("Cannot encode graph: \033[1;31m%s\033[0m\n", err.Error())
			http.Error(resp, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}

		header := resp.Header()
		header.Set("Content-Type", "application/json")

		resp.Write(js)

	}

}

// Report connections.
func report(dstore map[string][]string, lock *sync.Mutex) http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {

		lock.Lock()
		defer lock.Unlock()

		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()

		var vertex Vertex
		err := decoder.Decode(&vertex)
		if err != nil {
			log.Printf("Cannot decode vertex: \033[1;31m%s\033[0m\n", err.Error())
			http.Error(resp, "400 Bad Request", http.StatusBadRequest)
			return
		}

		dstore[vertex.Client] = vertex.Peers

	}

}
