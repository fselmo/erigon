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

package executiontests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/common/dbg"
	"github.com/erigontech/erigon/execution/tests/testutil"
	"github.com/erigontech/erigon/node/ethconfig"
)

// The two fixtures carry the same valid block. In mismatched_list, one value
// in its delivered access list is changed, so the list's hash differs from the
// header's. The block test drops that list, as sync drops a peer's, and the
// block imports and runs without it, reported as "bad-access-list".
func TestBlockTestDropsMismatchedBlockAccessList(t *testing.T) {
	if testing.Short() {
		t.Skip("long-running test")
	}
	previousParallel := dbg.Exec3Parallel
	dbg.Exec3Parallel = true
	t.Cleanup(func() { dbg.Exec3Parallel = previousParallel })

	for _, tc := range []struct {
		name       string
		accessList bool
		reason     string
	}{
		{"valid", true, ""},
		{"mismatched_list", false, "bad-access-list"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("testdata", "delivered_access_list", tc.name+".json"))
			require.NoError(t, err)
			var tests map[string]*testutil.BlockTest
			require.NoError(t, json.Unmarshal(src, &tests))
			var paths []ethconfig.BlockExecutionPath
			bt := tests[tc.name]
			bt.ExecWorkers = 2
			bt.ExecutionPathReporter = func(p ethconfig.BlockExecutionPath) { paths = append(paths, p) }
			require.NoError(t, bt.Run(t))
			require.Len(t, paths, 1)
			require.Equal(t, "parallel", paths[0].Path)
			require.Equal(t, tc.accessList, paths[0].AccessList)
			require.Equal(t, tc.reason, paths[0].Reason)
		})
	}
}
