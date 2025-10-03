// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package binutils contains some useful helpers for creating a :qa
package binutils

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/matheusd/gorpcbench/rpcbench"
)

// ReadInt64 reads an int64 from a reader, using aux as temp buffer.
func ReadInt64(r io.Reader, aux []byte) (int64, error) {
	if _, err := io.ReadFull(r, aux); err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(aux)), nil
}

// WriteInt64 writes an int64 to a writer, using aux as a temp buffer.
func WriteInt64(w io.Writer, aux []byte, v int64) error {
	binary.LittleEndian.PutUint64(aux, uint64(v))
	_, err := w.Write(aux)
	return err
}

// {read,write}Tree always reads/writes in the same order and there's no
// multiplexing so we always expect the same sequence coming back.

// WriteTree writes the given tree to a writer, in binary format.
func WriteTree(w io.Writer, aux []byte, tn *rpcbench.TreeNodeImpl) error {
	if err := WriteInt64(w, aux, tn.Value); err != nil {
		return err
	}

	for i := range tn.Children {
		err := WriteTree(w, aux, &tn.Children[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadTree reads the given tree from a reader, in binary format.
func ReadTree(r io.Reader, aux []byte, tn *rpcbench.TreeNodeImpl) error {
	var err error
	tn.Value, err = ReadInt64(r, aux)
	if err != nil {
		return fmt.Errorf("error reading value: %v", err)
	}
	for i := range tn.Children {
		err := ReadTree(r, aux, &tn.Children[i])
		if err != nil {
			return fmt.Errorf("error reading children: %v", err)
		}
	}
	return nil
}

// WriteMultTreeRequest writes the request to the MultTreeValues request in
// binary format.
func WriteMultTreeRequest(w io.Writer, aux []byte, mult int64, tree *rpcbench.TreeNodeImpl) error {
	if err := WriteInt64(w, aux, mult); err != nil {
		return err
	}

	if err := WriteInt64(w, aux, int64(tree.TotalNodes())); err != nil {
		return err
	}

	if err := WriteTree(w, aux, tree); err != nil {
		return err
	}

	return nil
}

// DoMultTreeRequest performs the MultTreeValues request between the binary
// read/writer.
func DoMultTreeRequest(r io.Reader, w io.Writer, aux []byte) error {
	var err error
	var mult int64
	if mult, err = ReadInt64(r, aux); err != nil {
		return fmt.Errorf("could not read mult: %v", err)
	}

	var totalNodes int64
	if totalNodes, err = ReadInt64(r, aux); err != nil {
		return fmt.Errorf("could not read totalNodes: %v", err)
	}

	// There's no need to actually decode the tree: we can
	// process in-line in the same order as it was received.
	var val int64
	for i := range totalNodes {
		if val, err = ReadInt64(r, aux); err != nil {
			return fmt.Errorf("unable to read value %d/%d: %v", i, totalNodes, err)
		}
		val *= mult
		if err = WriteInt64(w, aux, val); err != nil {
			return fmt.Errorf("unable to write value %d/%d: %v", i, totalNodes, err)
		}
	}

	return nil

}

// ReadMultTreeReponse reads the response of a binary execution of
// MultTreeValues, as produced by DoMultTreeRequest.
func ReadMultTreeReponse(r io.Reader, aux []byte, tn *rpcbench.TreeNodeImpl) error {
	return ReadTree(r, aux, tn)
}
