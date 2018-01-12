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

// Streamstore is a thread-safe collection of peer-stream pairs.
type Streamstore interface {

	// Add a stream to the stream store.  The `outgoing` flag specifies if the
	// stream is an outgoing stream.
	Add(peerID peer.ID, stream net.Stream, outgoing bool) bool

	// Remove a stream from the stream store.
	Remove(peer.ID)

	// Remove all streams from the stream store.
	Purge()

	// Apply a function to every outgoing stream in the stream store except
	// those specified in a sorted exclude list.
	Apply(func(peer.ID, io.Writer) error, peer.IDSlice) map[peer.ID]chan error

	// Get the peers associated with incoming streams.
	IncomingPeers() []peer.ID

	// Get the peers associated with the outgoing streams.
	OutgoingPeers() []peer.ID

	// Get the incoming capacity of the stream store.
	IncomingCapacity() int

	// Get the outgoing capacity of the stream store.
	OutgoingCapacity() int

	// Get the current number of incoming streams.
	IncomingSize() int

	// Get the current number of outgoing streams.
	OutgoingSize() int
}

type streamstore struct {
	incCapacity int
	outCapacity int

	incPeers map[peer.ID]incCtx
	outPeers map[peer.ID]outCtx

	txQueueSize int
	*logging.Logger
	*sync.Mutex
}

type outCtx struct {
	shutdown chan struct{}
	queue    chan transaction
	stream   net.Stream
}

type incCtx struct {
	stream net.Stream
}

type transaction struct {
	query  func(peer.ID, io.Writer) error
	result map[peer.ID]chan error
	*sync.Mutex
}

// New creates a stream store.
func New(incCapacity, outCapacity, txQueueSize int) Streamstore {
	return &streamstore{
		incCapacity: incCapacity,
		outCapacity: outCapacity,
		incPeers:    make(map[peer.ID]incCtx),
		outPeers:    make(map[peer.ID]outCtx),
		txQueueSize: txQueueSize,
		Logger:      logging.MustGetLogger("streamstore"),
		Mutex:       &sync.Mutex{},
	}
}

func (ss *streamstore) Add(pid peer.ID, stream net.Stream, outgoing bool) bool {
	ss.Lock()
	defer ss.Unlock()

	if outgoing {
		ctx, exists := ss.outPeers[pid]
		if exists {
			ss.Debug("Removing", pid, "from stream store")
			close(ctx.shutdown)
			ctx.stream.Close()
			delete(ss.outPeers, pid)
		} else {
			if ss.OutgoingSize() >= ss.OutgoingCapacity() {
				ss.Debug("Cannot add", pid, "to stream store")
				return false
			}
		}
		ctx = outCtx{
			make(chan struct{}),
			make(chan transaction, ss.txQueueSize),
			stream,
		}
		go func() {
			for {
				select {
				case <-ctx.shutdown:
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
		ss.Debug("Adding outgoing stream", pid, "to stream store")
		ss.outPeers[pid] = ctx
	} else {
		ctx, exists := ss.incPeers[pid]
		if exists {
			ss.Debug("Removing", pid, "from stream store")
			ctx.stream.Close()
			delete(ss.incPeers, pid)
		} else {
			if ss.IncomingSize() >= ss.IncomingCapacity() {
				ss.Debug("Cannot add", pid, "to stream store")
				return false
			}
		}
		ctx = incCtx{
			stream,
		}
		ss.Debug("Adding incoming stream", pid, "to stream store")
		ss.incPeers[pid] = ctx
	}
	return true
}

func (ss *streamstore) Apply(f func(peer.ID, io.Writer) error, exclude peer.IDSlice) map[peer.ID]chan error {
	ss.Lock()
	defer ss.Unlock()
	tx := transaction{
		f,
		make(map[peer.ID]chan error),
		&sync.Mutex{},
	}
	var group sync.WaitGroup
	for peerID, peerCtx := range ss.outPeers {
		i := sort.Search(len(exclude), func(i int) bool {
			return exclude[i] >= peerID
		})
		if i < len(exclude) && exclude[i] == peerID {
			continue
		}
		pid := peerID
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

func (ss *streamstore) IncomingCapacity() int {
	return ss.incCapacity
}

func (ss *streamstore) OutgoingCapacity() int {
	return ss.outCapacity
}

func (ss *streamstore) IncomingPeers() []peer.ID {
	ss.Lock()
	defer ss.Unlock()
	var peers []peer.ID
	for peerID := range ss.incPeers {
		peers = append(peers, peerID)
	}
	return peers
}

func (ss *streamstore) OutgoingPeers() []peer.ID {
	ss.Lock()
	defer ss.Unlock()
	var peers []peer.ID
	for peerID := range ss.outPeers {
		peers = append(peers, peerID)
	}
	return peers
}

func (ss *streamstore) Purge() {
	ss.Lock()
	defer ss.Unlock()

	for peerID, ctx := range ss.incPeers {
		pid := peerID
		ss.Debug("Removing incoming stream", pid, "from stream store")
		ctx.stream.Close()
	}

	for peerID, ctx := range ss.outPeers {
		pid := peerID
		ss.Debug("Removing outgoing stream", pid, "from stream store")
		close(ctx.shutdown)
		ctx.stream.Close()
	}

	ss.incPeers = make(map[peer.ID]incCtx)
	ss.outPeers = make(map[peer.ID]outCtx)
}

func (ss *streamstore) Remove(pid peer.ID) {
	ss.Lock()
	defer ss.Unlock()

	if ctx, exists := ss.outPeers[pid]; exists {
		ss.Debug("Removing outgoing stream", pid, "from stream store")
		close(ctx.shutdown)
		ctx.stream.Close()
		delete(ss.outPeers, pid)
	}

	if ctx, exists := ss.incPeers[pid]; exists {
		ss.Debug("Removing incoming stream", pid, "from stream store")
		ctx.stream.Close()
		delete(ss.incPeers, pid)
	}
}

func (ss *streamstore) IncomingSize() int {
	return len(ss.incPeers)
}

func (ss *streamstore) OutgoingSize() int {
	return len(ss.outPeers)
}
