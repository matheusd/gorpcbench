// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package websocket

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/matheusd/gorpcbench/internal/binutils"
	"github.com/matheusd/gorpcbench/internal/jsonutils"
	"github.com/matheusd/gorpcbench/rpcbench"
)

type wsClient struct {
	conn   *websocket.Conn
	tree   rpcbench.TreeNodeImpl
	aux    []byte
	c      net.Conn
	reader *bufio.Reader
	writer *bufio.Writer

	isJson bool
	msg    jsonutils.Message
	outMsg jsonutils.OutMessage
}

func (c *wsClient) Nop(ctx context.Context) error {
	if c.isJson {
		c.outMsg.Command = jsonutils.CmdNop
		c.outMsg.Payload = nil
		if err := c.conn.WriteJSON(c.outMsg); err != nil {
			return fmt.Errorf("unable to write JSON nop: %v", err)
		}

		if err := c.conn.ReadJSON(&c.msg); err != nil {
			return fmt.Errorf("unable to read JSON nop: %v", err)
		}

		return nil
	}

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
	if c.isJson {
		c.outMsg.Command = jsonutils.CmdAdd
		c.outMsg.Payload = jsonutils.AddRequest{A: a, B: b}
		if err := c.conn.WriteJSON(c.outMsg); err != nil {
			return 0, fmt.Errorf("unable to write JSON add: %v", err)
		}

		var res jsonutils.AddResponse
		if err := c.conn.ReadJSON(&res); err != nil {
			return 0, fmt.Errorf("unable to read JSON add: %v", err)
		}

		return res.Res, nil
	}

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

	if c.isJson {
		c.outMsg.Command = jsonutils.CmdMultTree
		c.outMsg.Payload = jsonutils.MultTreeRequest{Mult: mult, Tree: tree}
		if err := c.conn.WriteJSON(c.outMsg); err != nil {
			return nil, fmt.Errorf("unable to write JSON tree: %v", err)
		}
		tree.Reset()

		if err := c.conn.ReadJSON(tree); err != nil {
			return nil, fmt.Errorf("unable to read JSON tree: %v", err)
		}

		return tree, nil
	}

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
	if c.isJson {
		c.outMsg.Command = jsonutils.CmdToHex
		c.outMsg.Payload = in

		if err := c.conn.WriteJSON(c.outMsg); err != nil {
			return fmt.Errorf("unable to write JSON hex: %v", err)
		}

		var hexOut []byte
		if err := c.conn.ReadJSON(&hexOut); err != nil {
			return fmt.Errorf("unable to read JSON hex: %v", err)
		}
		copy(out, hexOut) // Json cannot decode directly into out.

		return nil
	}

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

func newWSClient(ctx context.Context, addr string, isJson bool) (*wsClient, error) {
	url := "ws://" + addr
	var header http.Header
	if isJson {
		header = make(http.Header)
		header.Add("Content-Type", "text/json")
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, header)
	if err != nil {
		return nil, err
	}

	return &wsClient{
		conn:   conn,
		aux:    make([]byte, 8),
		reader: bufio.NewReaderSize(nil, rpcbench.MaxHexEncodeSize*2),
		writer: bufio.NewWriterSize(nil, rpcbench.MaxHexEncodeSize*2),
		isJson: isJson,
	}, nil
}
