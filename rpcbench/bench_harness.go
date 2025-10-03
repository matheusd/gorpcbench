// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcbench

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"testing"
	"time"
)

type ClientCall int

const (
	ClientCallNop ClientCall = iota
	ClientCallAdd
	ClientCallTreeMult
	ClientCallToHex
)

func (cc ClientCall) String() string {
	switch cc {
	case ClientCallNop:
		return "nop"
	case ClientCallAdd:
		return "add"
	case ClientCallTreeMult:
		return "tree"
	case ClientCallToHex:
		return "hex"
	default:
		panic("unknown cc")
	}
}

func ClientCallMatrix() []ClientCall {
	return []ClientCall{ClientCallNop, ClientCallAdd, ClientCallTreeMult, ClientCallToHex}
}

type testTreePair struct {
	src, tgt TreeNode
}

type benchClient struct {
	c           Client
	rng         *rand.Rand
	rngReader   io.Reader
	testTrees   []testTreePair
	hexInBuf    []byte
	hexOutBuf   []byte
	hexCheckBuf []byte
}

type clientsHarness struct {
	clients []benchClient
}

func newClientHarness(ctx context.Context, saddr string, fac RPCFactory, nbClients int) (*clientsHarness, error) {
	ch := &clientsHarness{
		clients: make([]benchClient, 0, nbClients),
	}

	for range nbClients {
		c, err := fac.NewClient(ctx, saddr)
		if err != nil {
			return nil, err
		}

		rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))
		var chachaseed [32]byte
		binary.LittleEndian.PutUint64(chachaseed[:], rng.Uint64())
		rngReader := rand.NewChaCha8(chachaseed)

		// Initialize test trees.
		testTrees := make([]testTreePair, 6)

		// First one is shallow (only a single element).

		// Second one is deep and narrow.
		node := &testTrees[1].src
		for range 64 {
			node.Children = make([]TreeNode, 1)
			node = &node.Children[0]
		}
		deepClone(&testTrees[1].src, &testTrees[1].tgt)

		// Third one is broad, but shallow.
		testTrees[2].src.Children = make([]TreeNode, 64)
		deepClone(&testTrees[2].src, &testTrees[2].tgt)

		// Fourth is dense (deep and broad).
		makeDenseTreeNode(&testTrees[3].src, 5, 5)
		deepClone(&testTrees[3].src, &testTrees[3].tgt)

		// Fifth and sixth are random.
		makeRandomTree(&testTrees[4].src, rng, 6, 6)
		deepClone(&testTrees[4].src, &testTrees[4].tgt)
		makeRandomTree(&testTrees[5].src, rng, 6, 6)
		deepClone(&testTrees[5].src, &testTrees[5].tgt)

		const hexMaxSize = 8

		bcli := benchClient{
			c:           c,
			rng:         rng,
			rngReader:   rngReader,
			testTrees:   testTrees,
			hexInBuf:    make([]byte, hexMaxSize*1024),
			hexCheckBuf: make([]byte, hexMaxSize*1024),
			hexOutBuf:   make([]byte, hexMaxSize*2*1024),
		}
		ch.clients = append(ch.clients, bcli)
	}

	return ch, nil
}

type serverHarness struct {
	s    Server
	addr string
}

func newServerHarness(ctx context.Context, t testing.TB, fac RPCFactory) (*serverHarness, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	s, err := fac.NewServer(l)
	if err != nil {
		return nil, err
	}

	sh := &serverHarness{
		s:    s,
		addr: l.Addr().String(),
	}

	runChan := make(chan error, 1)
	go func() { runChan <- s.Run(ctx) }()
	t.Cleanup(func() {
		select {
		case runErr := <-runChan:
			if runErr != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("Error running server: %v", runErr)
				if !t.Failed() {
					t.FailNow()
				}
			}
		case <-time.After(time.Second):
			t.Error("Timed out waiting for server Run() to finish")
		}
	})

	return sh, nil
}
