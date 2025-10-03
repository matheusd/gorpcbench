// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package tcp

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/matheusd/gorpcbench/internal/binutils"
	"github.com/sourcegraph/conc/pool"
)

type tcpServer struct {
	l       net.Listener
	skipLog bool
}

func (s *tcpServer) runConn(ctx context.Context, c net.Conn) error {
	nopReplyBuf := []byte{cmdNop}

	aux := make([]byte, 8)
	reader := bufio.NewReader(c)
	writer := bufio.NewWriter(c)

	readHexBuf := make([]byte, 8*1024)
	hexEnc := hex.NewEncoder(writer)

	for ctx.Err() == nil {
		cmd, err := reader.ReadByte()
		if err != nil {
			if !s.skipLog {
				log.Printf("TCP Read error: %v", err)
			}
			if errors.Is(err, io.EOF) {
				// EOF here means remote or local is winding
				// down.
				return nil
			}
			return err
		}

		switch cmd {
		case cmdNop:
			if !s.skipLog {
				log.Printf("Nop() called")
			}
			_, err = writer.Write(nopReplyBuf)

		case cmdAdd:
			var a, b int64
			if a, err = binutils.ReadInt64(reader, aux); err != nil {
				return err
			}
			if b, err = binutils.ReadInt64(reader, aux); err != nil {
				return err
			}

			if err = binutils.WriteInt64(writer, aux, a+b); err != nil {
				return err
			}

		case cmdMultTree:
			if err = binutils.DoMultTreeRequest(reader, writer, aux); err != nil {
				return err
			}

		case cmdToHex:
			var size int64
			if size, err = binutils.ReadInt64(reader, aux); err != nil {
				return err
			}

			for size > 0 {
				buf := readHexBuf[min(len(readHexBuf), int(size)):]
				n, err := reader.Read(buf)
				if err != nil {
					return err
				}

				if _, err := hexEnc.Write(buf[:n]); err != nil {
					return err
				}

				size -= int64(n)
			}

		}

		if err := writer.Flush(); err != nil {
			return err
		}

		if err != nil {
			if !s.skipLog {
				log.Printf("TCP Write error: %v", err)
			}
			return err
		}
	}

	return ctx.Err()
}

func (s *tcpServer) Run(ctx context.Context) error {
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

func newTCPServer(l net.Listener) *tcpServer {
	return &tcpServer{l: l, skipLog: true}
}
