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

package engineapitester

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/erigontech/erigon/common"
	"github.com/erigontech/erigon/common/log/v3"
	"github.com/erigontech/erigon/execution/types"
)

// EngineTestDefinition is a blockchain_tests_engine fixture. Unlike an
// engine-x fixture it carries its own genesis instead of naming a shared
// pre-allocation, and its post-state lists every account.
type EngineTestDefinition struct {
	Fork          Fork                    `json:"network"`
	Genesis       EngineTestGenesis       `json:"genesisBlockHeader"`
	Pre           types.GenesisAlloc      `json:"pre"`
	PostState     *EngineXPostStateDiff   `json:"postState"`
	LastBlockHash *common.Hash            `json:"lastblockhash"`
	NewPayloads   []EngineXTestNewPayload `json:"engineNewPayloads"`
}

// EngineTestGenesis is a fixture's genesis block header: the genesis it
// describes and the hash it must have.
type EngineTestGenesis struct {
	types.Genesis
	Hash common.Hash
}

func (g *EngineTestGenesis) UnmarshalJSON(input []byte) error {
	if err := json.Unmarshal(input, &g.Genesis); err != nil {
		return err
	}
	var header struct {
		Hash common.Hash `json:"hash"`
	}
	if err := json.Unmarshal(input, &header); err != nil {
		return err
	}
	g.Hash = header.Hash
	return nil
}

// RunEngineTest runs one blockchain_tests_engine fixture on its own in-process
// node: it sends each payload and forkchoice update through the node's engine
// API at the versions the fixture names, then checks the head and post-state.
// The node and its temp directory are removed before returning.
func RunEngineTest(ctx context.Context, logger log.Logger, test EngineTestDefinition, opts ...EngineXTestRunnerOption) (err error) {
	preAllocHash := PreAllocHash(test.Genesis.Hash.Hex())
	runner := &EngineXTestRunner{
		ctx:    ctx,
		logger: logger,
		preAllocs: map[PreAllocHash]*PreAlloc{
			preAllocHash: {Genesis: test.Genesis.Genesis, Alloc: test.Pre},
		},
		testers: make(map[Fork]map[PreAllocHash]testerEntry),
	}
	for _, opt := range opts {
		opt(runner)
	}
	defer func() {
		err = errors.Join(err, runner.Close())
	}()
	tester, err := runner.getOrCreateTester(test.Fork, preAllocHash)
	if err != nil {
		return err
	}
	if got := tester.GenesisBlock.Hash(); got != test.Genesis.Hash {
		return fmt.Errorf("genesis block hash mismatch: want %s, got %s", test.Genesis.Hash, got)
	}
	return runner.execute(ctx, tester, EngineXTestDefinition{
		Fork:          test.Fork,
		PreAllocHash:  preAllocHash,
		LastBlockHash: test.LastBlockHash,
		PostStateDiff: test.PostState,
		NewPayloads:   test.NewPayloads,
	})
}
