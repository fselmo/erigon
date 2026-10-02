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
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// exceptionSubstrings and exceptionPatterns map erigon's error messages to
// the exception names that execution-spec-tests fixtures expect. They copy
// ErigonExceptionMapper, the mapping EEST's consume uses for erigon, from
// packages/testing/src/execution_testing/client_clis/clis/erigon.py in
// ethereum/execution-specs at e35787f5c67dd55c3a6874755b4b6a54bd6f255f
// (forks/amsterdam); keep the two in sync.
var exceptionSubstrings = []struct{ name, substring string }{
	{"TransactionException.SENDER_NOT_EOA", "sender not an eoa"},
	{"TransactionException.INITCODE_SIZE_EXCEEDED", "max initcode size exceeded"},
	{"TransactionException.INSUFFICIENT_ACCOUNT_FUNDS", "insufficient funds for gas * price + value"},
	{"TransactionException.NONCE_IS_MAX", "nonce has max value"},
	{"TransactionException.INTRINSIC_GAS_TOO_LOW", "intrinsic gas too low"},
	{"TransactionException.INTRINSIC_GAS_BELOW_FLOOR_GAS_COST", "intrinsic gas too low"},
	{"TransactionException.INSUFFICIENT_MAX_FEE_PER_GAS", "fee cap less than block base fee"},
	{"TransactionException.PRIORITY_GREATER_THAN_MAX_FEE_PER_GAS", "tip higher than fee cap"},
	{"TransactionException.INSUFFICIENT_MAX_FEE_PER_BLOB_GAS", "max fee per blob gas too low"},
	{"TransactionException.NONCE_MISMATCH_TOO_LOW", "nonce too low"},
	{"TransactionException.NONCE_MISMATCH_TOO_HIGH", "nonce too high"},
	{"TransactionException.GAS_ALLOWANCE_EXCEEDED", "gas limit reached"},
	{"TransactionException.INVALID_CHAINID", "invalid chain id for signer"},
	{"TransactionException.INVALID_SIGNATURE_VRS", "invalid transaction v, r, s values"},
	{"TransactionException.TYPE_3_TX_PRE_FORK", "blob txn is not supported by signer"},
	{"TransactionException.TYPE_3_TX_INVALID_BLOB_VERSIONED_HASH", "invalid blob versioned hash, must start with VERSIONED_HASH_VERSION_KZG"},
	{"TransactionException.TYPE_3_TX_BLOB_COUNT_EXCEEDED", "blob transaction has too many blobs"},
	{"TransactionException.TYPE_3_TX_ZERO_BLOBS", "a blob stx must contain at least one blob"},
	{"TransactionException.TYPE_3_TX_WITH_FULL_BLOBS", "rlp: expected String or Byte"},
	{"TransactionException.TYPE_3_TX_CONTRACT_CREATION", "wrong size for To: 0"},
	{"TransactionException.TYPE_3_TX_MAX_BLOB_GAS_ALLOWANCE_EXCEEDED", "blobs/blobgas exceeds max"},
	{"TransactionException.TYPE_4_EMPTY_AUTHORIZATION_LIST", "SetCodeTransaction without authorizations is invalid"},
	{"TransactionException.TYPE_4_TX_CONTRACT_CREATION", "wrong size for To: 0"},
	{"TransactionException.TYPE_4_TX_PRE_FORK", "setCode tx is not supported by signer"},
	{"BlockException.INVALID_DEPOSIT_EVENT_LAYOUT", "could not parse requests logs"},
	{"BlockException.SYSTEM_CONTRACT_EMPTY", "Syscall failure: Empty Code at"},
	{"BlockException.SYSTEM_CONTRACT_CALL_FAILED", "Unprecedented Syscall failure"},
	{"BlockException.INVALID_REQUESTS", "invalid requests root hash in header"},
	{"BlockException.INVALID_BLOCK_HASH", "invalid block hash"},
	{"BlockException.RLP_BLOCK_LIMIT_EXCEEDED", "block exceeds max rlp size"},
	{"BlockException.INVALID_BASEFEE_PER_GAS", "invalid block: invalid baseFee"},
	{"BlockException.INVALID_BLOCK_TIMESTAMP_OLDER_THAN_PARENT", "invalid block: timestamp older than parent"},
	{"BlockException.INVALID_BLOCK_NUMBER", "invalid block number"},
	{"BlockException.EXTRA_DATA_TOO_BIG", "invalid block: extra-data longer than 32 bytes"},
	{"BlockException.INVALID_GASLIMIT", "invalid block: invalid gas limit"},
	{"BlockException.INVALID_STATE_ROOT", "invalid block: wrong trie root"},
	{"BlockException.INVALID_RECEIPTS_ROOT", "receiptHash mismatch"},
	{"BlockException.INVALID_LOG_BLOOM", "invalid bloom"},
	{"BlockException.INCORRECT_BLOCK_FORMAT", "invalid block access list"},
	{"BlockException.GAS_USED_OVERFLOW", "block gas used overflow"},

	// Not in EEST's mapper yet: block import rejects a pre-Amsterdam header
	// that carries a block access list hash before computing the hash the
	// fixture expects to mismatch.
	{"BlockException.INVALID_BLOCK_HASH", "unexpected bal hash"},
}

var exceptionPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	// In-range r that is not an x-coordinate on the curve: the range check
	// passes and libsecp256k1 recovery itself fails.
	{"TransactionException.INVALID_SIGNATURE_VRS", regexp.MustCompile(`recovery failed`)},
	{"BlockException.INVALID_BAL_HASH", regexp.MustCompile(`invalid block access list|block access list mismatch`)},
	{"BlockException.INVALID_BLOCK_ACCESS_LIST", regexp.MustCompile(`invalid block access list|block access list mismatch`)},
	{"BlockException.INCORRECT_BLOCK_FORMAT", regexp.MustCompile(`invalid block access list`)},
	{"BlockException.BLOCK_ACCESS_LIST_GAS_LIMIT_EXCEEDED", regexp.MustCompile(`block access list too large`)},
	{"TransactionException.GAS_LIMIT_EXCEEDS_MAXIMUM", regexp.MustCompile(`gas limit too high`)},
	{"BlockException.INCORRECT_BLOB_GAS_USED", regexp.MustCompile(`blobGasUsed by execution: \d+, in header: \d+`)},
	{"BlockException.INCORRECT_EXCESS_BLOB_GAS", regexp.MustCompile(`invalid excessBlobGas: have \d+, want \d+`)},
	{"BlockException.INVALID_GAS_USED", regexp.MustCompile(`gas used by execution: \w+, in header: \w+`)},
	{"BlockException.INVALID_GAS_USED_ABOVE_LIMIT", regexp.MustCompile(`invalid gasUsed: have \d+, gasLimit \d+`)},
}

// exceptionNames returns the exception names that msg maps to.
func exceptionNames(msg string) []string {
	var names []string
	for _, m := range exceptionSubstrings {
		if strings.Contains(msg, m.substring) {
			names = append(names, m.name)
		}
	}
	for _, m := range exceptionPatterns {
		if m.pattern.MatchString(msg) && !slices.Contains(names, m.name) {
			names = append(names, m.name)
		}
	}
	return names
}

// checkException checks that the client rejected a block for the reason the
// fixture expects: err must map to one of the names in expected, which may
// list several separated by "|". An empty expected accepts any rejection.
func checkException(expected string, err error) error {
	if expected == "" {
		return nil
	}
	if err == nil {
		return fmt.Errorf("block rejected without an error, expected %s", expected)
	}
	actual := exceptionNames(err.Error())
	for want := range strings.SplitSeq(expected, "|") {
		if slices.Contains(actual, strings.TrimSpace(want)) {
			return nil
		}
	}
	if len(actual) == 0 {
		return fmt.Errorf("expected %s, got an error that maps to no exception: %w", expected, err)
	}
	return fmt.Errorf("expected %s, got %s: %w", expected, strings.Join(actual, "|"), err)
}
