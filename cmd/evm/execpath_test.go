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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/cmd/utils/cmdtest"
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
			name:       "dropped access list under --exec.serial",
			execSerial: true,
			path:       ethconfig.BlockExecutionPath{Number: 12, Hash: hash, Path: "sequential", Reason: "bad-access-list"},
			want:       `{"event":"balExecution","block":12,"hash":"0x0000000000000000000000000000000000000000000000000000000000abcdef","path":"sequential","reason":"bad-access-list","scheduler":"optimistic"}`,
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

// Both runners print a line per executed block with --bal-report, and none
// without it.
func TestBALReportFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for testing.Short")
	}
	for _, runner := range []struct {
		command string
		fixture string
	}{
		{"blocktest", "../../execution/tests/testdata/delivered_access_list/valid.json"},
		{"enginetest", "../../execution/tests/testdata/engine/bal_cross_block_ripemd160_state_leak.json"},
	} {
		for _, balReport := range []bool{true, false} {
			args := []string{runner.command, "--jsonout"}
			if balReport {
				args = append(args, "--bal-report")
			}
			t.Run(fmt.Sprintf("%s bal-report=%v", runner.command, balReport), func(t *testing.T) {
				tt := cmdtest.NewTestCmd(t, nil)
				tt.Run("evm-test", append(args, runner.fixture)...)
				stdout := tt.Output()
				tt.WaitExit()
				require.Equal(t, 0, tt.ExitStatus())

				var results []testResult
				require.NoError(t, json.Unmarshal(stdout, &results))
				require.NotEmpty(t, results)
				for _, r := range results {
					require.True(t, r.Pass, r.Name)
				}

				lines := bytes.Count(tt.Stderr(), []byte(`"event":"balExecution"`))
				if balReport {
					require.Positive(t, lines)
				} else {
					require.Zero(t, lines)
				}
			})
		}
	}
}
