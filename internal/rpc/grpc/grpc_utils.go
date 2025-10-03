// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package grpc

import (
	"errors"

	"github.com/matheusd/gorpcbench/rpcbench"
)

func treeToGrpc(t *rpcbench.TreeNodeImpl, g *TreeNode) {
	g.Value = t.Value
	g.Children = make([]*TreeNode, len(t.Children))
	for i := range t.Children {
		g.Children[i] = new(TreeNode)
		treeToGrpc(&t.Children[i], g.Children[i])
	}
}

func grpcToTree(g *TreeNode, t *rpcbench.TreeNodeImpl) error {
	if len(g.Children) != len(t.Children) {
		return errors.New("wrong number of children")
	}

	t.Value = g.Value
	for i := range t.Children {
		if err := grpcToTree(g.Children[i], &t.Children[i]); err != nil {
			return err
		}
	}
	return nil
}
