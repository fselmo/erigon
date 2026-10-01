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

	"github.com/erigontech/erigon/execution/tests/testutil"
)

// The two fixtures carry the same block. In wrong_list, the block is expected
// invalid and only its rlp_decoded access list differs, so it is rejected only
// if the block test delivers that list with the block.
func TestBlockTestDeliversInvalidBlockAccessList(t *testing.T) {
	if testing.Short() {
		t.Skip("long-running test")
	}
	for _, name := range []string{"valid", "wrong_list"} {
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("testdata", "delivered_access_list", name+".json"))
			require.NoError(t, err)
			var tests map[string]*testutil.BlockTest
			require.NoError(t, json.Unmarshal(src, &tests))
			require.NoError(t, tests[name].Run(t))
		})
	}
}
