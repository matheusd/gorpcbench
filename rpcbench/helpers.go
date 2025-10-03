// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcbench

import "math/rand/v2"

func populateWithRand(tn *TreeNode, rng *rand.Rand) {
	tn.Value = rng.Int64()
	for i := range tn.Children {
		populateWithRand(&tn.Children[i], rng)
	}
}

func deepClone(src, tgt *TreeNode) {
	tgt.Value = src.Value
	tgt.Children = make([]TreeNode, len(src.Children))
	for i := range src.Children {
		deepClone(&src.Children[i], &tgt.Children[i])
	}
}

func copyValues(src, tgt *TreeNode) {
	tgt.Value = src.Value
	for i := range src.Children {
		copyValues(&src.Children[i], &tgt.Children[i])
	}
}

func makeDenseTreeNode(src *TreeNode, density int, level int) {
	if level <= 0 {
		return
	}
	src.Children = make([]TreeNode, density)
	for i := range density {
		makeDenseTreeNode(&src.Children[i], density, level-1)
	}
}

func makeRandomTree(src *TreeNode, rng *rand.Rand, level, branching int) {
	if level <= 0 || branching <= 0 {
		return
	}
	c := rng.IntN(branching)
	src.Children = make([]TreeNode, c)
	for i := range src.Children {
		cb := branching - rng.IntN(branching/2)
		makeRandomTree(&src.Children[i], rng, level-1, cb)
	}
}

func treeMatchesForMult(src, tgt *TreeNode, mult int64) bool {
	if tgt.Value != src.Value*mult {
		return false
	}
	if len(tgt.Children) != len(src.Children) {
		return false
	}
	for i := range src.Children {
		if !treeMatchesForMult(&src.Children[i], &tgt.Children[i], mult) {
			return false
		}
	}
	return true
}
