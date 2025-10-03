// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package websocket

import (
	"bufio"
	"context"
	"io"
	"net"

	"github.com/gorilla/websocket"
	"github.com/matheusd/gorpcbench/internal/binutils"
	"github.com/matheusd/gorpcbench/rpcbench"
)

type wsClient struct {
	conn   *websocket.Conn
	tree   rpcbench.TreeNodeImpl
	aux    []byte
	c      net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func (c *wsClient) Nop(ctx context.Context) error {
	rawWriter, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	c.writer.Reset(rawWriter)

	if err := c.writer.WriteByte(cmdNop); err != nil {
		return err
	}
	if err := c.writer.Flush(); err != nil {
		return err
	}
	if err := rawWriter.Close(); err != nil {
		return err
	}

	_, rawReader, err := c.conn.NextReader()
	if err != nil {
		return err
	}

	c.reader.Reset(rawReader)
	_, err = c.reader.ReadByte()
	return err
}

func (c *wsClient) Add(ctx context.Context, a int64, b int64) (int64, error) {
	rawWriter, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	c.writer.Reset(rawWriter)

	if err := c.writer.WriteByte(cmdAdd); err != nil {
		return 0, err
	}
	if err := binutils.WriteInt64(c.writer, c.aux, a); err != nil {
		return 0, err
	}
	if err := binutils.WriteInt64(c.writer, c.aux, b); err != nil {
		return 0, err
	}
	if err := c.writer.Flush(); err != nil {
		return 0, err
	}
	if err := rawWriter.Close(); err != nil {
		return 0, err
	}

	_, rawReader, err := c.conn.NextReader()
	if err != nil {
		return 0, err
	}
	c.reader.Reset(rawReader)

	return binutils.ReadInt64(c.reader, c.aux)
}

func (c *wsClient) MultTreeValues(ctx context.Context, mult int64, fillArgs func(rpcbench.TreeNode)) (rpcbench.TreeNode, error) {
	tree := &c.tree
	tree.Reset()
	fillArgs(tree)
	rawWriter, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return nil, err
	}
	c.writer.Reset(rawWriter)

	if err := c.writer.WriteByte(cmdMultTree); err != nil {
		return nil, err
	}

	if err := binutils.WriteMultTreeRequest(c.writer, c.aux, mult, tree); err != nil {
		return nil, err
	}

	if err := c.writer.Flush(); err != nil {
		return nil, err
	}
	if err := rawWriter.Close(); err != nil {
		return nil, err
	}

	_, rawReader, err := c.conn.NextReader()
	if err != nil {
		return nil, err
	}
	c.reader.Reset(rawReader)

	if err := binutils.ReadMultTreeReponse(c.reader, c.aux, tree); err != nil {
		return nil, err
	}
	return tree, nil
}

func (c *wsClient) ToHex(ctx context.Context, in, out []byte) error {
	rawWriter, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	c.writer.Reset(rawWriter)

	if err := c.writer.WriteByte(cmdToHex); err != nil {
		return err
	}
	if err := binutils.WriteInt64(c.writer, c.aux, int64(len(in))); err != nil {
		return err
	}
	if _, err := c.writer.Write(in); err != nil {
		return err
	}
	if err := c.writer.Flush(); err != nil {
		return err
	}
	if err := rawWriter.Close(); err != nil {
		return err
	}

	_, rawReader, err := c.conn.NextReader()
	if err != nil {
		return err
	}
	c.reader.Reset(rawReader)

	_, err = io.ReadFull(c.reader, out)
	return err
}

func (c *wsClient) Run(ctx context.Context) error {
	<-ctx.Done()
	return c.conn.Close()
}

func newWSClient(ctx context.Context, addr string) (*wsClient, error) {
	url := "ws://" + addr
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return nil, err
	}

	return &wsClient{
		conn:   conn,
		aux:    make([]byte, 8),
		reader: bufio.NewReader(nil),
		writer: bufio.NewWriter(nil),
	}, nil
}
