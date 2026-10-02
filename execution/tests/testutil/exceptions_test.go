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

package testutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/rpc"
)

func TestCheckException(t *testing.T) {
	stateRoot := errors.New("insertion failed for block 1, code: BadBlock err: invalid block: wrong trie root")
	for _, tc := range []struct {
		name     string
		expected string
		err      error
		want     string
	}{
		{"match", "BlockException.INVALID_STATE_ROOT", stateRoot, ""},
		{"any of several", "BlockException.INVALID_GASLIMIT|BlockException.INVALID_STATE_ROOT", stateRoot, ""},
		{"nothing expected", "", stateRoot, ""},
		{"wrong reason", "BlockException.INVALID_GASLIMIT", stateRoot, "expected BlockException.INVALID_GASLIMIT, got BlockException.INVALID_STATE_ROOT: insertion failed"},
		{"unmapped", "BlockException.INVALID_STATE_ROOT", errors.New("something else"), "expected BlockException.INVALID_STATE_ROOT, got an error that maps to no exception: something else"},
		{"no error", "BlockException.INVALID_STATE_ROOT", nil, "block rejected without an error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkException(tc.expected, tc.err)
			if tc.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.want)
			}
		})
	}
}

func TestCheckErrorCode(t *testing.T) {
	invalidParams := &rpc.InvalidParamsError{Message: "blockAccessList missing"}
	require.NoError(t, checkErrorCode("-32602", invalidParams))
	require.ErrorContains(t, checkErrorCode("-38003", invalidParams), "expected error code -38003, got -32602")
	require.ErrorContains(t, checkErrorCode("-32602", errors.New("plain")), "got an error without a code")
}
