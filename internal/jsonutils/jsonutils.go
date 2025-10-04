// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package jsonutils

import (
	"encoding/json"

	"github.com/matheusd/gorpcbench/rpcbench"
)

type Command int

const (
	CmdNop Command = iota
	CmdAdd
	CmdMultTree
	CmdToHex
)

type Message struct {
	Command Command         `json:"command"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type OutMessage struct {
	Command Command `json:"command"`
	Payload any     `json:"payload,omitempty"`
}

type AddRequest struct {
	A int64 `json:"a"`
	B int64 `json:"b"`
}

type AddResponse struct {
	Res int64 `json:"res"`
}

type MultTreeRequest struct {
	Mult int64                  `json:"mult"`
	Tree *rpcbench.TreeNodeImpl `json:"tree"`
}
