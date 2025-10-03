// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package websocket

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/matheusd/gorpcbench/internal/binutils"
)

const (
	cmdNop      byte = 1
	cmdAdd      byte = 2
	cmdMultTree byte = 3
	cmdToHex    byte = 4
)

type wsServer struct {
	l        net.Listener
	skipLog  bool
	upgrader *websocket.Upgrader
}

func (s *wsServer) runBinaryConn(conn *websocket.Conn) error {
	nopReplyBuf := []byte{cmdNop}
	aux := make([]byte, 8)
	reader := &bufio.Reader{}
	readHexBuf := make([]byte, 10*1024)
	writeHexBuf := make([]byte, len(readHexBuf)*2)
	for {
		// Read message from client
		_, rawReader, err := conn.NextReader()
		if err != nil {
			return fmt.Errorf("error obtaining reader: %w", err)
		}
		reader.Reset(rawReader)

		cmd, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("error reading cmd: %w", err)
		}

		writer, err := conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			return fmt.Errorf("error obtaining writer: %w", err)
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

				hex.Encode(writeHexBuf, buf[:n])
				writer.Write(writeHexBuf[:n*2])
				size -= int64(n)
			}

		}

		if err := writer.Close(); err != nil {
			return fmt.Errorf("error closing writer: %w", err)
		}
	}
}

func (s *wsServer) runJsonConn(_ *websocket.Conn) error {
	return errors.New("unimplemented")
}

func (s *wsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if !s.skipLog {
			log.Println("Upgrade error:", err)
		}
		return
	}
	defer conn.Close()

	isJson := r.Header.Get("Content-Type") == "text/json"

	if isJson {
		err = s.runJsonConn(conn)
	} else {
		err = s.runBinaryConn(conn)
	}

	if err != nil && !s.skipLog && !errors.Is(err, io.EOF) {
		log.Printf("Websocket conn errored: %v", err)
	}
}

func (s *wsServer) Run(ctx context.Context) error {
	var hs http.Server

	hs.Addr = s.l.Addr().String()
	hs.Handler = s
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		hs.Shutdown(shutCtx)
		cancel()
	}()

	err := hs.Serve(s.l)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	return err
}

func newWSServer(l net.Listener) *wsServer {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for simplicity
		},
	}

	return &wsServer{
		l:        l,
		skipLog:  true,
		upgrader: &upgrader,
	}
}
