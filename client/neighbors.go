/**
 * File        : neighbors.go
 * Description : Sorting algorithm.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"sort"

	"github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p-kbucket/keyspace"
	"github.com/libp2p/go-libp2p-peer"
)

type keyref struct {
	key keyspace.Key
	ref int
}

// Find the nearest neighbors of the client.
func (client *client) neighbors(peers []peer.ID) ([]peer.ID, []peer.ID) {

	var xs []keyref

	for i := range peers {

		x := keyref{
			client.key.Space.Key(kbucket.ConvertPeerID(peers[i])),
			i,
		}

		if client.key.Equal(x.key) {
			continue
		}

		j := sort.Search(len(xs), func(k int) bool {
			return client.key.Space.Less(x.key, xs[k].key)
		})

		xs = append(xs, x)
		copy(xs[j+1:], xs[j:])
		xs[j] = x

	}

	j := sort.Search(len(xs), func(k int) bool {
		return client.key.Space.Less(client.key, xs[k].key)
	})

	ys := make([]keyref, len(xs))
	copy(ys[0:], xs[j:])
	copy(ys[len(xs)-j:], xs[0:])

	m := len(ys) / 2
	a := ys[0:m]
	b := ys[m:]

	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}

	l := make([]peer.ID, len(a))
	for i := range l {
		l[i] = peers[a[i].ref]
	}

	r := make([]peer.ID, len(b))
	for i := range r {
		r[i] = peers[b[i].ref]
	}

	return l, r

}
