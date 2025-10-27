// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package tcp

import (
	"bufio"
	"context"
	"io"
	"net"

	"github.com/matheusd/gorpcbench/internal/binutils"
	"github.com/matheusd/gorpcbench/rpcbench"
)

type tcpClient struct {
	aux    []byte
	c      net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	tree   rpcbench.TreeNodeImpl
}

func (c *tcpClient) Nop(ctx context.Context) error {
	if err := c.writer.WriteByte(cmdNop); err != nil {
		return err
	}

	if err := c.writer.Flush(); err != nil {
		return err
	}

	_, err := c.reader.ReadByte()
	return err
}

func (c *tcpClient) Add(ctx context.Context, a int64, b int64) (int64, error) {
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

	return binutils.ReadInt64(c.reader, c.aux)
}

func (c *tcpClient) MultTreeValues(ctx context.Context, mult int64, fillArgs func(rpcbench.TreeNode)) (rpcbench.TreeNode, error) {
	tree := &c.tree
	tree.Reset()
	fillArgs(tree)
	if err := c.writer.WriteByte(cmdMultTree); err != nil {
		return nil, err
	}

	if err := binutils.WriteMultTreeRequest(c.writer, c.aux, mult, tree); err != nil {
		return nil, err
	}

	if err := c.writer.Flush(); err != nil {
		return nil, err
	}

	if err := binutils.ReadMultTreeReponse(c.reader, c.aux, tree); err != nil {
		return nil, err
	}
	return tree, nil
}

func (c *tcpClient) ToHex(ctx context.Context, in, out []byte) error {
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
	_, err := io.ReadFull(c.reader, out)
	return err
}

func newTCPClient(ctx context.Context, addr string) (*tcpClient, error) {
	// Try to connect.
	var dc net.Dialer
	c, err := dc.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		c.Close()
	}()
	return &tcpClient{
		c:      c,
		reader: bufio.NewReaderSize(c, rpcbench.MaxHexEncodeSize*2),
		writer: bufio.NewWriterSize(c, rpcbench.MaxHexEncodeSize*2),
		aux:    make([]byte, 8),
	}, nil
}
