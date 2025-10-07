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

// MaxHexEncodeSize is the maximum size of a toHex message. This can be
// considered as the max message size in a particular RPC implementation.
const MaxHexEncodeSize = 128 * 1024

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

type benchClient struct {
	c              Client
	rng            *rand.Rand
	rngReader      io.Reader
	testTrees      []TreeNodeImpl
	chosenTestTree int
	hexInBuf       []byte
	hexOutBuf      []byte
	hexCheckBuf    []byte
	fillTreeArgs   func(node TreeNode) // Storing here avoids one alloc per call.
}

func (bcli *benchClient) fillRequestTree(node TreeNode) {
	tgt := &bcli.testTrees[bcli.chosenTestTree]
	copyTree(tgt, node)
}

type clientsHarness struct {
	clients []*benchClient
}

func newClientHarness(ctx context.Context, saddr string, fac RPCFactory, nbClients int) (*clientsHarness, error) {
	ch := &clientsHarness{
		clients: make([]*benchClient, 0, nbClients),
	}

	for i := range nbClients {
		c, err := fac.NewClient(ctx, saddr)
		if err != nil {
			return nil, err
		}

		// Deterministic rng per client.
		rng := rand.New(rand.NewPCG(0x01020304, uint64(i)))
		var chachaseed [32]byte
		binary.LittleEndian.PutUint64(chachaseed[:], rng.Uint64())
		rngReader := rand.NewChaCha8(chachaseed)

		// Initialize test trees.
		testTrees := make([]TreeNodeImpl, 6)

		// First one is only a single element.

		// Second one is deep and narrow.
		var node TreeNode = &testTrees[1]
		for range 64 {
			node.InitChildren(1)
			node = node.Child(0)
		}

		// Third one is broad, but shallow.
		testTrees[2].InitChildren(64)

		// Fourth is dense (deep and broad).
		makeDenseTreeNode(&testTrees[3], 5, 6)

		// Fifth and sixth are random.
		makeRandomTree(&testTrees[4], rng, 8, 5)
		makeRandomTree(&testTrees[5], rng, 8, 5)

		bcli := &benchClient{
			c:           c,
			rng:         rng,
			rngReader:   rngReader,
			testTrees:   testTrees,
			hexInBuf:    make([]byte, MaxHexEncodeSize),
			hexCheckBuf: make([]byte, MaxHexEncodeSize),
			hexOutBuf:   make([]byte, MaxHexEncodeSize*2),
		}
		bcli.fillTreeArgs = bcli.fillRequestTree
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
