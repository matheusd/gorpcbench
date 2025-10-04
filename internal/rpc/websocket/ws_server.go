// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package websocket

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/matheusd/gorpcbench/internal/binutils"
	"github.com/matheusd/gorpcbench/internal/jsonutils"
	"github.com/matheusd/gorpcbench/rpcbench"
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
	readHexBuf := make([]byte, rpcbench.MaxHexEncodeSize)
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

func (s *wsServer) runJsonConn(conn *websocket.Conn) error {
	var msg jsonutils.Message
	var addReq jsonutils.AddRequest
	var addRes jsonutils.AddResponse
	var multReq jsonutils.MultTreeRequest
	toHexInBuf := make([]byte, rpcbench.MaxHexEncodeSize)
	toHexOutBuf := make([]byte, rpcbench.MaxHexEncodeSize*2)

	for {
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}

		switch msg.Command {
		case jsonutils.CmdNop:
			if err := conn.WriteJSON(msg); err != nil {
				return err
			}

		case jsonutils.CmdAdd:
			if err := json.Unmarshal(msg.Payload, &addReq); err != nil {
				return err
			}
			addRes.Res = addReq.A + addReq.B
			if err := conn.WriteJSON(addRes); err != nil {
				return err
			}

		case jsonutils.CmdMultTree:
			if err := json.Unmarshal(msg.Payload, &multReq); err != nil {
				return err
			}
			multReq.Tree.Mult(multReq.Mult)

			if err := conn.WriteJSON(multReq.Tree); err != nil {
				return err
			}

		case jsonutils.CmdToHex:
			if err := json.Unmarshal(msg.Payload, &toHexInBuf); err != nil {
				return err
			}
			n := hex.Encode(toHexOutBuf, toHexInBuf)
			if err := conn.WriteJSON(toHexOutBuf[:n]); err != nil {
				return err
			}
		}
	}
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
		ReadBufferSize:  rpcbench.MaxHexEncodeSize * 2,
		WriteBufferSize: rpcbench.MaxHexEncodeSize * 2,
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
