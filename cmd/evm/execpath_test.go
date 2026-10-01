// Copyright 2026 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/common"
	"github.com/erigontech/erigon/node/ethconfig"
)

func TestExecutionPathReporterLines(t *testing.T) {
	hash := common.HexToHash("0xabcdef")
	for _, tc := range []struct {
		name       string
		execSerial bool
		path       ethconfig.BlockExecutionPath
		want       string
	}{
		{
			name: "parallel with access list",
			path: ethconfig.BlockExecutionPath{Number: 12, Hash: hash, Path: "parallel", AccessList: true},
			want: `{"event":"balExecution","block":12,"hash":"0x0000000000000000000000000000000000000000000000000000000000abcdef","path":"parallel","reason":"","scheduler":"bal"}`,
		},
		{
			name: "parallel without access list",
			path: ethconfig.BlockExecutionPath{Number: 12, Hash: hash, Path: "parallel"},
			want: `{"event":"balExecution","block":12,"hash":"0x0000000000000000000000000000000000000000000000000000000000abcdef","path":"parallel","reason":"","scheduler":"optimistic"}`,
		},
		{
			name:       "single worker under --exec.serial",
			execSerial: true,
			path:       ethconfig.BlockExecutionPath{Number: 12, Hash: hash, Path: "sequential", Reason: "single-worker", AccessList: true},
			want:       `{"event":"balExecution","block":12,"hash":"0x0000000000000000000000000000000000000000000000000000000000abcdef","path":"sequential","reason":"disabled","scheduler":"bal"}`,
		},
		{
			name: "single worker by configuration",
			path: ethconfig.BlockExecutionPath{Number: 12, Hash: hash, Path: "sequential", Reason: "single-worker", AccessList: true},
			want: `{"event":"balExecution","block":12,"hash":"0x0000000000000000000000000000000000000000000000000000000000abcdef","path":"sequential","reason":"single-worker","scheduler":"bal"}`,
		},
		{
			name: "serial executor",
			path: ethconfig.BlockExecutionPath{Number: 12, Hash: hash, Path: "sequential", Reason: "serial-executor"},
			want: `{"event":"balExecution","block":12,"hash":"0x0000000000000000000000000000000000000000000000000000000000abcdef","path":"sequential","reason":"serial-executor"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			executionPathReporter(&out, tc.execSerial)(tc.path)
			require.Equal(t, tc.want+"\n", out.String())
		})
	}
}
