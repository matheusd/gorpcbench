package mdcapnp

import (
	"github.com/matheusd/gorpcbench/rpcbench"
	ser "matheusd.com/mdcapnp/capnpser"
)

type tnBuilderAdapterAlt struct {
	tnb      treeNodeBuilder
	children []tnBuilderAdapterAlt
}

func (b *tnBuilderAdapterAlt) SetValue(v int64) {
	b.tnb.SetValue(v)
}

func (b *tnBuilderAdapterAlt) GetValue() int64 {
	s := (*ser.StructBuilder)(&b.tnb).Reader()
	return s.Int64(0)
}

func (b *tnBuilderAdapterAlt) InitChildren(n int) {
	children, err := b.tnb.NewChildren(n, n)
	if err != nil {
		panic(err)
	}
	if cap(b.children) >= n {
		b.children = b.children[:n]
	} else {
		b.children = make([]tnBuilderAdapterAlt, n)
	}
	for i := range n {
		b.children[i].tnb = children.At(i)
	}
}

func (b *tnBuilderAdapterAlt) ChildrenCount() int {
	return len(b.children)
}

func (b *tnBuilderAdapterAlt) Child(i int) rpcbench.TreeNode {
	return &b.children[i]
}

func (b *tnBuilderAdapterAlt) TotalNodes() int {
	sum := 1
	for i := range b.children {
		sum += b.children[i].TotalNodes()
	}
	return sum
}

type tnReaderAdapterAlt struct {
	tn       treeNode
	children []tnReaderAdapterAlt
}

func (b *tnReaderAdapterAlt) SetValue(v int64) {
	panic("cannot set value in reader adapter")
}

func (b *tnReaderAdapterAlt) GetValue() int64 {
	v := (*ser.Struct)(&b.tn).Int64(0)
	return v
}

func (b *tnReaderAdapterAlt) InitChildren(n int) {
	panic("cannot init children in reader")
}

func (b *tnReaderAdapterAlt) ChildrenCount() int {
	return len(b.children)
}

func (b *tnReaderAdapterAlt) Child(i int) rpcbench.TreeNode {
	return &b.children[i]
}

func (b *tnReaderAdapterAlt) TotalNodes() int {
	sum := 1
	l := b.ChildrenCount()
	for i := range l {
		sum += b.Child(i).TotalNodes()
	}
	return sum
}

func (b *tnReaderAdapterAlt) reset() error {
	children, err := b.tn.Children()
	if err != nil {
		return err
	}

	childrenLen := children.Len()
	if cap(b.children) >= childrenLen {
		b.children = b.children[:childrenLen]
	} else {
		b.children = make([]tnReaderAdapterAlt, childrenLen)
	}
	for i := range children.Len() {
		c := children.At(i)
		b.children[i].tn = c
		if err := b.children[i].reset(); err != nil {
			return err
		}
	}
	return nil
}

func newTnReaderAdapter(root treeNode) (*tnReaderAdapterAlt, error) {
	res := &tnReaderAdapterAlt{tn: root}
	return res, res.reset()
}
