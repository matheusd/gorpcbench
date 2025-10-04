// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package gocapnp

import (
	context "context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/sourcegraph/conc/pool"
)

type gocapnpServer struct {
	l       net.Listener
	skipLog bool
}

func (s *gocapnpServer) Nop(context.Context, API_nop) error {
	if !s.skipLog {
		log.Println("Got nop() call")
	}
	return nil
}

func (s *gocapnpServer) Add(_ context.Context, call API_add) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	res.SetRes(call.Args().A() + call.Args().B())
	return nil
}

func (s *gocapnpServer) MultTree(_ context.Context, call API_multTree) error {
	tree, err := call.Args().Tree()
	if err != nil {
		return err
	}

	multTree(call.Args().Mult(), tree)

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	// Unfortunately, this doesn't work for large structs (bug), so we need
	// to manually copy.
	/*
		if err := res.SetRes(tree); err != nil {
			log.Println("setRes returned error", err)
			return err
		}
	*/
	resTree, err := res.NewRes()
	if err != nil {
		return err
	}
	copyTree(tree, resTree)

	return nil
}

func (s *gocapnpServer) ToHex(_ context.Context, call API_toHex) error {
	in, err := call.Args().In()
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	out := make([]byte, len(in)*2)
	hex.Encode(out, in)
	if err := res.SetOut(out); err != nil {
		return err
	}

	return nil
}

func (s *gocapnpServer) runConn(ctx context.Context, c net.Conn) error {
	// Cast the server as a capability that can be served through bootstrap.
	client := API_ServerToClient(s)

	// Convert net.Conn into a capnp Conn.
	conn := rpc.NewConn(rpc.NewStreamTransport(c), &rpc.Options{
		BootstrapClient: capnp.Client(client),
	})
	defer conn.Close()

	// Wait for connection to abort.
	select {
	case <-conn.Done():
		return nil
	case <-ctx.Done():
		return conn.Close()
	}
}

func (s *gocapnpServer) Run(ctx context.Context) error {
	g := pool.New().WithContext(ctx).WithCancelOnError().WithFirstError()

	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return s.l.Close()
	})

	g.Go(func(ctx context.Context) error {
		var acceptErr error
		connPool := pool.New().WithContext(ctx).WithCancelOnError().WithFirstError()
		for {
			c, err := s.l.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					acceptErr = err
				}
				break
			}

			if !s.skipLog {
				log.Printf("Accepted connection from %s", c.RemoteAddr())
			}
			connPool.Go(func(ctx context.Context) error { return s.runConn(ctx, c) })
		}

		waitErr := connPool.Wait()
		switch {
		case acceptErr != nil:
			return fmt.Errorf("server Accept() errored: %w", acceptErr)
		case waitErr != nil:
			return fmt.Errorf("conn wait() errored: %w", waitErr)
		default:
			return nil
		}
	})

	err := g.Wait()
	if errors.Is(err, context.Canceled) {
		err = nil
	}
	return err

}

func newGoCapnpServer(l net.Listener) (*gocapnpServer, error) {
	return &gocapnpServer{
		l:       l,
		skipLog: true,
	}, nil

}
