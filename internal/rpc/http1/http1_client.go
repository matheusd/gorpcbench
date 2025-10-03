// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package http1

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/matheusd/gorpcbench/internal/binutils"
	"github.com/matheusd/gorpcbench/rpcbench"
)

type http1Client struct {
	hc     http.Client
	isJson bool
	tree   rpcbench.TreeNodeImpl

	aux        []byte
	bodyWriter bytes.Buffer
	bodyReader bytes.Reader

	nopURL  string
	addURL  string
	treeURL string
	hexURL  string
}

func (c *http1Client) Nop(ctx context.Context) error {
	_, err := c.hc.Get(c.nopURL)
	return err
}

func (c *http1Client) Add(ctx context.Context, a int64, b int64) (int64, error) {
	if c.isJson {
		panic("todo")
	}

	c.bodyWriter.Reset()
	if err := binutils.WriteInt64(&c.bodyWriter, c.aux, a); err != nil {
		return 0, err
	}
	if err := binutils.WriteInt64(&c.bodyWriter, c.aux, b); err != nil {
		return 0, err
	}

	r, err := c.hc.Post(c.addURL, "application/octet-stream", &c.bodyWriter)
	if err != nil {
		return 0, err
	}
	defer r.Body.Close()

	return binutils.ReadInt64(r.Body, c.aux)
}

func (c *http1Client) MultTreeValues(ctx context.Context, mult int64, fillArgs func(rpcbench.TreeNode)) (rpcbench.TreeNode, error) {
	tree := &c.tree
	tree.Reset()
	fillArgs(tree)
	if c.isJson {
		panic("todo")
	}

	c.bodyWriter.Reset()
	if err := binutils.WriteMultTreeRequest(&c.bodyWriter, c.aux, mult, tree); err != nil {
		return nil, err
	}

	r, err := c.hc.Post(c.treeURL, "application/octet-stream", &c.bodyWriter)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	reader := bufio.NewReader(r.Body)

	err = binutils.ReadMultTreeReponse(reader, c.aux, tree)
	if err != nil {
		return nil, fmt.Errorf("error reading reply from server: %v", err)
	}
	return tree, nil
}

func (c *http1Client) ToHex(ctx context.Context, in, out []byte) error {
	if c.isJson {
		panic("todo")
	}

	c.bodyReader.Reset(in)
	r, err := c.hc.Post(c.hexURL, "application/octet-stream", &c.bodyReader)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if _, err := io.ReadFull(r.Body, out); err != nil {
		return err
	}

	return nil
}

func newHttp1Client(_ context.Context, addr string) (*http1Client, error) {
	dialerCtx := func(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
		return dialer.DialContext
	}

	hc := http.Client{
		// Create one transport per client because http1 doesn't
		// multiplex concurrent requests in parallel test settings.
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: dialerCtx(&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}),
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	return &http1Client{
		hc:      hc,
		aux:     make([]byte, 8),
		nopURL:  "http://" + addr + "/nop",
		addURL:  "http://" + addr + "/add",
		treeURL: "http://" + addr + "/multTree",
		hexURL:  "http://" + addr + "/toHex",
	}, nil
}
