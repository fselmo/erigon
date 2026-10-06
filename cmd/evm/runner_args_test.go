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
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/cmd/utils/cmdtest"
)

// Each runner runs the fixtures under every path argument, files and
// directories alike, not only the first.
func TestRunnersRunEveryPathArgument(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for testing.Short")
	}
	const (
		blockFixtures  = "../../execution/tests/testdata/delivered_access_list/"
		engineFixtures = "../../execution/tests/testdata/engine/"
		stateFixtures  = "../../execution/tests/test-corners/state/"
	)
	for _, tc := range []struct {
		command string
		paths   []string
		want    int
	}{
		{"blocktest", []string{blockFixtures + "valid.json", blockFixtures + "mismatched_list.json"}, 2},
		// The directory holds the file and one more fixture.
		{"enginetest", []string{engineFixtures + "bal_cross_block_ripemd160_state_leak.json", engineFixtures}, 3},
		{"statetest", []string{stateFixtures + "CallNonExistingAccount.json", stateFixtures + "eip2681-max-sender-nonce.json"}, 2},
	} {
		t.Run(tc.command, func(t *testing.T) {
			tt := cmdtest.NewTestCmd(t, nil)
			tt.Run("evm-test", append([]string{tc.command, "--jsonout"}, tc.paths...)...)
			stdout := tt.Output()
			tt.WaitExit()
			require.Equal(t, 0, tt.ExitStatus())

			var results []testResult
			require.NoError(t, json.Unmarshal(stdout, &results))
			require.Len(t, results, tc.want)
			for _, r := range results {
				require.True(t, r.Pass, r.Name)
			}
		})
	}
}

// A path argument that does not exist fails the run before any fixture runs,
// instead of being skipped.
func TestRunnersRejectMissingPathArgument(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for testing.Short")
	}
	const missing = "../../execution/tests/testdata/does-not-exist.json"
	for _, tc := range []struct {
		command string
		path    string
	}{
		{"blocktest", "../../execution/tests/testdata/delivered_access_list/valid.json"},
		{"enginetest", "../../execution/tests/testdata/engine/bal_cross_block_ripemd160_state_leak.json"},
		{"statetest", "../../execution/tests/test-corners/state/CallNonExistingAccount.json"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			tt := cmdtest.NewTestCmd(t, nil)
			tt.Run("evm-test", tc.command, "--jsonout", tc.path, missing)
			stdout := tt.Output()
			tt.WaitExit()
			require.NotEqual(t, 0, tt.ExitStatus())
			require.Empty(t, stdout)
			require.Contains(t, tt.StderrText(), "does-not-exist.json")
		})
	}
}

// A fixture that cannot be decoded fails on its own with the decoder's error;
// the other fixtures in its file run as usual.
func TestRunnersRunFixturesBesideAnUnreadableOne(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for testing.Short")
	}
	for _, tc := range []struct {
		command string
		fixture string
	}{
		{"blocktest", "../../execution/tests/testdata/delivered_access_list/valid.json"},
		{"enginetest", "../../execution/tests/testdata/engine/bal_cross_block_ripemd160_state_leak.json"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			data, err := os.ReadFile(tc.fixture)
			require.NoError(t, err)
			var fixtures map[string]map[string]any
			require.NoError(t, json.Unmarshal(data, &fixtures))
			require.Len(t, fixtures, 1)
			var fixture map[string]any
			for _, f := range fixtures {
				fixture = f
			}
			unreadable := maps.Clone(fixture)
			unreadable["genesisBlockHeader"] = map[string]any{"gasLimit": "0xzz"}
			data, err = json.Marshal(map[string]any{"a": fixture, "b": unreadable, "c": fixture})
			require.NoError(t, err)
			path := filepath.Join(t.TempDir(), "batch.json")
			require.NoError(t, os.WriteFile(path, data, 0o644))

			tt := cmdtest.NewTestCmd(t, nil)
			tt.Run("evm-test", tc.command, "--jsonout", path)
			stdout := tt.Output()
			tt.WaitExit()
			require.Equal(t, 0, tt.ExitStatus(), tt.StderrText())

			var results []testResult
			require.NoError(t, json.Unmarshal(stdout, &results))
			require.Len(t, results, 3)
			for i, name := range []string{"a", "b", "c"} {
				r := results[i]
				require.Equal(t, name, r.Name)
				require.NotNil(t, r.Rejections, name)
				require.Empty(t, *r.Rejections, name)
				if name == "b" {
					require.False(t, r.Pass)
					require.Contains(t, r.Error, `invalid hex or decimal integer "0xzz"`)
					continue
				}
				require.True(t, r.Pass, r.Error)
			}
		})
	}
}

// A fixture file that is not a JSON object fails the run.
func TestRunnersRejectFixtureFileThatIsNotAnObject(t *testing.T) {
	for _, command := range []string{"blocktest", "enginetest"} {
		t.Run(command, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "array.json")
			require.NoError(t, os.WriteFile(path, []byte("[1,2]"), 0o644))

			tt := cmdtest.NewTestCmd(t, nil)
			tt.Run("evm-test", command, "--jsonout", path)
			stdout := tt.Output()
			tt.WaitExit()
			require.NotEqual(t, 0, tt.ExitStatus())
			require.Empty(t, stdout)
			require.Contains(t, tt.StderrText(), "cannot unmarshal array")
		})
	}
}
