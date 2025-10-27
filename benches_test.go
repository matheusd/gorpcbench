// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	"github.com/matheusd/gorpcbench/rpcbench"
)

func BenchmarkRPC(b *testing.B) {
	matrix := fullTestMatrix()

	for _, bc := range matrix {
		b.Run(bc.Name(), func(b *testing.B) {
			err := rpcbench.RunCase(b, bc)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}
