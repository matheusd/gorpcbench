// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package http1

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/matheusd/gorpcbench/internal/binutils"
)

type http1Server struct {
	l       net.Listener
	skipLog bool
	mux     http.ServeMux
}

func (s *http1Server) handleNop(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *http1Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") == "application/json" {
		w.WriteHeader(http.StatusBadRequest) // Not implemented.
		return
	}

	// Assume binary encoding.
	reader := bufio.NewReader(r.Body)
	var a, b int64
	var err error
	aux := make([]byte, 8)
	if a, err = binutils.ReadInt64(reader, aux); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if b, err = binutils.ReadInt64(reader, aux); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err = binutils.WriteInt64(w, aux, a+b); err != nil {
		if !s.skipLog {
			log.Printf("Unable to write response to add(): %v", err)
		}
		return
	}
}

func (s *http1Server) handleMultTree(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") == "application/json" {
		w.WriteHeader(http.StatusBadRequest) // Not implemented.
		return
	}

	// Enabling full duplex avoids having to read the entire structure in
	// memory.
	rpc := http.NewResponseController(w)
	if err := rpc.EnableFullDuplex(); err != nil {
		if !s.skipLog {
			log.Printf("Unable to enable full duplex: %v", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Assume binary encoding.
	reader := bufio.NewReader(r.Body)
	writer := bufio.NewWriter(w)
	aux := make([]byte, 8)
	if err := binutils.DoMultTreeRequest(reader, writer, aux); err != nil {
		if !s.skipLog {
			log.Printf("Unable to write response to multTree(): %v", err)
		}
		return
	}
	if err := writer.Flush(); err != nil {
		if !s.skipLog {
			log.Printf("Unable to flush response to multTree(): %v", err)
		}
		return

	}
}

func (s *http1Server) handleToHex(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") == "application/json" {
		w.WriteHeader(http.StatusBadRequest) // Not implemented.
		return
	}

	// Enabling full duplex avoids having to read the entire structure in
	// memory.
	rpc := http.NewResponseController(w)
	if err := rpc.EnableFullDuplex(); err != nil {
		if !s.skipLog {
			log.Printf("Unable to enable full duplex: %v", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	enc := hex.NewEncoder(w)
	_, err := io.Copy(enc, r.Body)
	if err != nil {
		if !s.skipLog {
			log.Printf("Unable to write response to toHex(): %v", err)
		}
	}
}

func (s *http1Server) Run(ctx context.Context) error {
	var hs http.Server

	hs.Addr = s.l.Addr().String()
	hs.Handler = &s.mux
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

func newHttp1Server(l net.Listener) *http1Server {
	s := &http1Server{l: l, skipLog: true}
	s.mux.HandleFunc("/nop", s.handleNop)
	s.mux.HandleFunc("/add", s.handleAdd)
	s.mux.HandleFunc("/multTree", s.handleMultTree)
	s.mux.HandleFunc("/toHex", s.handleToHex)
	return s
}
