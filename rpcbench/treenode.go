package rpcbench

import (
	"fmt"
	"math/rand/v2"
)

// TreeNodeImpl is a simple nested data structure. It implements TreeNode.
type TreeNodeImpl struct {
	Value    int64
	Children []TreeNodeImpl
}

func (t *TreeNodeImpl) SetValue(v int64) {
	t.Value = v
}

func (t *TreeNodeImpl) GetValue() int64 {
	return t.Value
}

func (t *TreeNodeImpl) InitChildren(n int) {
	if cap(t.Children) >= n {
		t.Children = t.Children[:n]
	} else {
		t.Children = make([]TreeNodeImpl, n)
	}
}

func (t *TreeNodeImpl) ChildrenCount() int {
	return len(t.Children)
}

func (t *TreeNodeImpl) Child(i int) TreeNode {
	return &t.Children[i]
}

// Reset truncates all children, but keeps their storage around.
func (t *TreeNodeImpl) Reset() {
	for i := range t.Children {
		t.Children[i].Reset()
	}
	t.Children = t.Children[:0]
}

// TotalNodes returns the total number of nodes in this tree.
func (tn *TreeNodeImpl) TotalNodes() int {
	sum := 1
	for i := range tn.Children {
		sum += tn.Children[i].TotalNodes()
	}
	return sum
}

// Mult multiplies every value of the tree by mult.
func (tn *TreeNodeImpl) Mult(mult int64) {
	tn.Value *= mult
	for i := range tn.Children {
		tn.Children[i].Mult(mult)
	}
}

// NewTreeNode returns a new TreeNodeImpl as a TreeNode.
func NewTreeNode() TreeNode {
	return &TreeNodeImpl{}
}

func copyLayout(src TreeNode, tgt *TreeNodeImpl) {
	tgt.Children = make([]TreeNodeImpl, src.ChildrenCount())
	for i := range src.ChildrenCount() {
		copyLayout(src.Child(i), &tgt.Children[i])
	}
}

func populateWithRand(tn TreeNode, rng *rand.Rand) {
	tn.SetValue(rng.Int64())
	for i := range tn.ChildrenCount() {
		populateWithRand(tn.Child(i), rng)
	}
}

func copyTree(from, to TreeNode) {
	to.SetValue(from.GetValue())
	to.InitChildren(from.ChildrenCount())
	for i := range from.ChildrenCount() {
		copyTree(from.Child(i), to.Child(i))
	}
}

func makeDenseTreeNode(src TreeNode, density int, level int) {
	if level <= 0 {
		return
	}
	src.InitChildren(density)
	for i := range density {
		makeDenseTreeNode(src.Child(i), density, level-1)
	}
}

func makeRandomTree(src TreeNode, rng *rand.Rand, level, branching int) {
	if level <= 0 || branching <= 0 {
		return
	}
	c := rng.IntN(branching)
	src.InitChildren(c)
	for i := range c {
		cb := branching - rng.IntN(branching/2)
		makeRandomTree(src.Child(i), rng, level-1, cb)
	}
}

func treeMatchesForMult(src TreeNode, tgt *TreeNodeImpl, mult int64) bool {
	if tgt.Value*mult != src.GetValue() {
		fmt.Println("wrong value")
		return false
	}
	if len(tgt.Children) != src.ChildrenCount() {
		fmt.Println("wrong child count", len(tgt.Children), src.ChildrenCount())
		return false
	}
	for i := range tgt.Children {
		if !treeMatchesForMult(src.Child(i), &tgt.Children[i], mult) {
			fmt.Println("wrong children", i)
			return false
		}
	}
	return true
}

func PrintTreeNode(t TreeNode, mult int64, prefix string) {
	fmt.Println(prefix, t.GetValue()*mult)
	for i := range t.ChildrenCount() {
		PrintTreeNode(t.Child(i), mult, prefix+"    ")
	}
}
