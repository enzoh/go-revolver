/**
 * File        : streamstore.go
 * Description : High-level stream store interface.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 */

package streamstore

import (
	"errors"
	"io"
	"sort"
	"sync"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	"github.com/enzoh/go-logging"
)

// A thread-safe collection of peer-stream pairs.
type Streamstore interface {

	// Add a stream to a stream store.
	Add(peer.ID, net.Stream) bool

	// Apply a function to every stream in a stream store except those
	// specified in a sorted exclude list.
	Apply(func(peer.ID, io.Writer) error, peer.IDSlice) map[peer.ID]chan error

	// Get the capacity of a stream store.
	Capacity() int

	// Get the peers associated with a stream store.
	Peers() []peer.ID

	// Remove all streams from a stream store.
	Purge()

	// Remove a stream from a stream store.
	Remove(peer.ID)

	// Get the size of a stream store.
	Size() int
}

type streamstore struct {
	capacity    int
	data        map[peer.ID]peerctx
	txQueueSize int
	*logging.Logger
	*sync.Mutex
}

type peerctx struct {
	notify chan struct{}
	queue  chan transaction
	stream net.Stream
}

type transaction struct {
	query  func(peer.ID, io.Writer) error
	result map[peer.ID]chan error
	*sync.Mutex
}

// Create a stream store.
func New(capacity int, txQueueSize int) Streamstore {
	return &streamstore{
		capacity,
		make(map[peer.ID]peerctx),
		txQueueSize,
		logging.MustGetLogger("streamstore"),
		&sync.Mutex{},
	}
}

// Add a stream to a stream store.
func (ss *streamstore) Add(peerId peer.ID, stream net.Stream) bool {
	ss.Lock()
	defer ss.Unlock()
	pid := peerId
	ctx, exists := ss.data[pid]
	if exists {
		ss.Debug("Removing", pid, "from stream store")
		close(ctx.notify)
		ctx.stream.Close()
		delete(ss.data, pid)
	} else if ss.Capacity() <= ss.Size() {
		ss.Debug("Cannot add", pid, "to stream store")
		return false
	}
	ctx = peerctx{
		make(chan struct{}),
		make(chan transaction, ss.txQueueSize),
		stream,
	}
	go func() {
		for {
			select {
			case <-ctx.notify:
				return
			case tx := <-ctx.queue:
				ss.Debug("Processing transaction for", pid)
				err := tx.query(pid, ctx.stream)
				ss.Debug("Recording result for", pid)
				tx.Lock()
				tx.result[pid] <- err
				tx.Unlock()
			}
		}
	}()
	ss.Debug("Adding", pid, "to stream store")
	ss.data[pid] = ctx
	return true
}

// Apply a function to every stream in a stream store except those specified in
// a sorted exclude list.
func (ss *streamstore) Apply(f func(peer.ID, io.Writer) error, exclude peer.IDSlice) map[peer.ID]chan error {
	ss.Lock()
	defer ss.Unlock()
	tx := transaction{
		f,
		make(map[peer.ID]chan error),
		&sync.Mutex{},
	}
	var group sync.WaitGroup
	for peerId, peerCtx := range ss.data {
		i := sort.Search(len(exclude), func(i int) bool {
			return exclude[i] >= peerId
		})
		if i < len(exclude) && exclude[i] == peerId {
			continue
		}
		pid := peerId
		ctx := peerCtx
		group.Add(1)
		go func() {
			defer group.Done()
			ss.Debug("Preparing result for", pid)
			tx.Lock()
			tx.result[pid] = make(chan error, 1)
			tx.Unlock()
			ss.Debug("Queueing transaction for", pid)
			select {
			case ctx.queue <- tx:
			default:
				ss.Debug("Cannot queue transaction for", pid)
				tx.Lock()
				tx.result[pid] <- errors.New("transaction queue is full")
				tx.Unlock()
			}
		}()
	}
	group.Wait()
	return tx.result
}

// Get the capacity of a stream store.
func (ss *streamstore) Capacity() int {
	return ss.capacity
}

// Get the peers associated with a stream store.
func (ss *streamstore) Peers() []peer.ID {
	ss.Lock()
	defer ss.Unlock()
	peers := make([]peer.ID, ss.Size())
	i := 0
	for peerId := range ss.data {
		peers[i] = peerId
		i++
	}
	return peers
}

// Remove all streams from a stream store.
func (ss *streamstore) Purge() {
	ss.Lock()
	defer ss.Unlock()
	for peerId, ctx := range ss.data {
		pid := peerId
		ss.Debug("Removing", pid, "from stream store")
		close(ctx.notify)
		ctx.stream.Close()
		delete(ss.data, pid)
	}
}

// Remove a stream from a stream store.
func (ss *streamstore) Remove(peerId peer.ID) {
	ss.Lock()
	defer ss.Unlock()
	pid := peerId
	ctx, exists := ss.data[pid]
	if exists {
		ss.Debug("Removing", pid, "from stream store")
		close(ctx.notify)
		ctx.stream.Close()
		delete(ss.data, pid)
	}
}

// Get the size of a stream store.
func (ss *streamstore) Size() int {
	return len(ss.data)
}
