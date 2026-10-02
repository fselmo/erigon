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
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/execution/tests/testutil"
)

const engineFixture = "testdata/engine/bal_cross_block_ripemd160_state_leak.json"

func loadEngineTest(t *testing.T, edit func(fixture map[string]any)) *testutil.EngineTest {
	t.Helper()
	src, err := os.ReadFile(engineFixture)
	require.NoError(t, err)
	var fixtures map[string]map[string]any
	require.NoError(t, json.Unmarshal(src, &fixtures))
	fixture := fixtures["bal_cross_block_ripemd160_state_leak"]
	if edit != nil {
		edit(fixture)
	}
	src, err = json.Marshal(fixture)
	require.NoError(t, err)
	var test testutil.EngineTest
	require.NoError(t, json.Unmarshal(src, &test))
	return &test
}

func payload(fixture map[string]any, i int) map[string]any {
	return fixture["engineNewPayloads"].([]any)[i].(map[string]any)
}

func payloadParams(fixture map[string]any, i int) map[string]any {
	return payload(fixture, i)["params"].([]any)[0].(map[string]any)
}

// deliverBlock0AccessList makes block 1 deliver block 0's list: well formed,
// but not the one its header commits to.
func deliverBlock0AccessList(fixture map[string]any) {
	payloadParams(fixture, 1)["blockAccessList"] = payloadParams(fixture, 0)["blockAccessList"]
}

func TestEngineTest(t *testing.T) {
	if testing.Short() {
		t.Skip("long-running test")
	}

	t.Run("valid chain", func(t *testing.T) {
		require.NoError(t, loadEngineTest(t, nil).RunCLI())
	})

	t.Run("wrong access list", func(t *testing.T) {
		test := loadEngineTest(t, deliverBlock0AccessList)
		require.ErrorContains(t, test.RunCLI(), "payload 1: payload status INVALID")
	})

	// The same block, expected invalid: with CheckExceptions it passes only
	// when the fixture names the reason erigon rejects it for.
	for _, tc := range []struct {
		expected string
		check    bool
		err      string
	}{
		{"BlockException.INVALID_BLOCK_HASH", true, ""},
		{"TransactionException.INSUFFICIENT_ACCOUNT_FUNDS", true, "payload 1: expected TransactionException.INSUFFICIENT_ACCOUNT_FUNDS, got BlockException.INVALID_BLOCK_HASH"},
		{"TransactionException.INSUFFICIENT_ACCOUNT_FUNDS", false, ""},
	} {
		t.Run(fmt.Sprintf("wrong access list expecting %s check=%v", tc.expected, tc.check), func(t *testing.T) {
			test := loadEngineTest(t, func(fixture map[string]any) {
				deliverBlock0AccessList(fixture)
				payload(fixture, 1)["validationError"] = tc.expected
				fixture["lastblockhash"] = payloadParams(fixture, 0)["blockHash"]
				delete(fixture, "postState") // the fixture's is block 1's
			})
			test.CheckExceptions = tc.check
			err := test.RunCLI()
			if tc.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.err)
			}
		})
	}

	// A node with no peers cannot fetch a missing parent; the test must fail
	// with SYNCING rather than wait or crash.
	t.Run("unknown parent", func(t *testing.T) {
		test := loadEngineTest(t, func(fixture map[string]any) {
			fixture["engineNewPayloads"] = fixture["engineNewPayloads"].([]any)[1:]
		})
		require.ErrorContains(t, test.RunCLI(), "payload 0: payload status SYNCING")
	})
}
