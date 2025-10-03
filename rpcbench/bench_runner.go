// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcbench

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime"
	"testing"
)

type RPCSystem struct {
	Name   string
	Initer FactoryIniter
	Notes  string
}

type BenchCase struct {
	Sys      *RPCSystem
	Call     ClientCall
	Parallel bool
}

func (bc BenchCase) Name() string {
	if !bc.Parallel {
		return fmt.Sprintf("sequential/%s/%s", bc.Call, bc.Sys.Name)
	}
	return fmt.Sprintf("parallel/%s/%s", bc.Call, bc.Sys.Name)
}

func makeCall(ctx context.Context, bc BenchCase, bcli *benchClient) (int, error) {
	switch bc.Call {
	case ClientCallNop:
		return 0, bcli.c.Nop(ctx)

	case ClientCallAdd:
		a, b := bcli.rng.Int64(), bcli.rng.Int64()
		res, err := bcli.c.Add(ctx, a, b)
		if err != nil {
			return 0, err
		}
		if res != a+b {
			return 0, fmt.Errorf("wrong result: got %v, want %v", res, a+b)
		}
		return 0, nil

	case ClientCallTreeMult:
		// Pick a random tree and mult and populate the target tree.
		bcli.chosenTestTree = bcli.rng.IntN(len(bcli.testTrees))
		// bcli.chosenTestTree = 3
		mult := bcli.rng.Int64()
		pair := &bcli.testTrees[bcli.chosenTestTree]
		populateWithRand(&pair.tgt, bcli.rng)

		// Execute the call.
		res, err := bcli.c.MultTreeValues(ctx, mult, bcli.fillTreeArgs)
		if err != nil {
			return 0, err
		}

		// Check if result matches expected.
		if !treeMatchesForMult(res, &pair.tgt, mult) {
			fmt.Println("res")
			PrintTreeNode(res, 1, "")
			fmt.Println("target")
			PrintTreeNode(&pair.tgt, mult, "")
			return 0, errors.New("mismatch in request and response trees")
		}

		return 0, nil

	case ClientCallToHex:
		size := bcli.rng.IntN(len(bcli.hexInBuf))
		bcli.rngReader.Read(bcli.hexInBuf[:size])
		err := bcli.c.ToHex(ctx, bcli.hexInBuf[:size], bcli.hexOutBuf[:size*2])
		if err != nil {
			return 0, err
		}

		if _, err := hex.Decode(bcli.hexCheckBuf, bcli.hexOutBuf[:size*2]); err != nil {
			return 0, fmt.Errorf("unable to decode hex out buf: %v", err)
		}
		if !bytes.Equal(bcli.hexCheckBuf[:size], bcli.hexInBuf[:size]) {
			return 0, fmt.Errorf("mismatch in request and response hex")
		}
		return len(bcli.hexInBuf) + len(bcli.hexOutBuf), nil

	default:
		return 0, fmt.Errorf("unknown call in makeCall(): %d", bc.Call)
	}
}

func runSequentialBench(b *testing.B, bc BenchCase) error {
	fac := bc.Sys.Initer()

	ctx := b.Context()
	sh, err := newServerHarness(ctx, b, fac)
	if err != nil {
		return err
	}

	ch, err := newClientHarness(ctx, sh.addr, fac, 1)
	if err != nil {
		return err
	}

	b.ReportAllocs()

	var N, totalBytes int64
	for b.Loop() {
		if bytes, err := makeCall(ctx, bc, ch.clients[0]); err != nil {
			return err
		} else {
			totalBytes += int64(bytes)
		}
		N++
	}

	b.SetBytes(totalBytes / N)

	return nil
}

func runParallelBench(b *testing.B, bc BenchCase) error {
	fac := bc.Sys.Initer()

	ctx := b.Context()
	sh, err := newServerHarness(ctx, b, fac)
	if err != nil {
		return err
	}

	nbClients := runtime.GOMAXPROCS(0)
	ch, err := newClientHarness(ctx, sh.addr, fac, nbClients)
	if err != nil {
		return err
	}

	clients := make(chan *benchClient, nbClients)
	for i := range ch.clients {
		clients <- ch.clients[i]
	}

	type procTotals struct {
		N          int64
		totalBytes int64
	}
	totalsChan := make(chan procTotals, nbClients)

	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		var N, totalBytes int64
		c := <-clients
		for p.Next() {
			if bytes, err := makeCall(ctx, bc, c); err != nil {
				b.Fatal(err)
			} else {
				totalBytes += int64(bytes)
			}
			N++
		}
		totalsChan <- procTotals{N: N, totalBytes: totalBytes}
	})

	var N int64
	var totalBytes int64
	for range nbClients {
		tot := <-totalsChan
		N += tot.N
		totalBytes += tot.totalBytes
	}
	b.SetBytes(totalBytes / N)

	return nil

}

func RunCase(b *testing.B, bc BenchCase) error {
	if !bc.Parallel {
		return runSequentialBench(b, bc)
	}

	return runParallelBench(b, bc)
}
