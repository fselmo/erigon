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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/common"
)

func TestEngineTestDefinitionGenesis(t *testing.T) {
	input := []byte(`{
		"network":"Amsterdam",
		"genesisBlockHeader":{
			"parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
			"coinbase":"0x0000000000000000000000000000000000000000",
			"stateRoot":"0x1c622a4d1670dc02a132af7aedad38ec75b5ac0bc52b39e5384a5a3746d93c76",
			"difficulty":"0x00",
			"number":"0x00",
			"gasLimit":"0x07270e00",
			"gasUsed":"0x00",
			"timestamp":"0x00",
			"extraData":"0x00",
			"nonce":"0x0000000000000000",
			"baseFeePerGas":"0x07",
			"blobGasUsed":"0x00",
			"excessBlobGas":"0x00",
			"requestsHash":"0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			"hash":"0x50601369124dd3b3d7d412444e97a0b2903ecd5e3f7ede163284855e6a1bfec2"
		},
		"pre":{"0x0000000000000000000000000000000000000001":{"nonce":"0x01","balance":"0x00","code":"0x","storage":{}}},
		"postState":{"0x0000000000000000000000000000000000000001":{"nonce":"0x01","balance":"0x00","code":"0x","storage":{}}},
		"lastblockhash":"0x50601369124dd3b3d7d412444e97a0b2903ecd5e3f7ede163284855e6a1bfec2",
		"engineNewPayloads":[]
	}`)
	var definition EngineTestDefinition
	require.NoError(t, json.Unmarshal(input, &definition))
	require.Equal(t, common.HexToHash("0x50601369124dd3b3d7d412444e97a0b2903ecd5e3f7ede163284855e6a1bfec2"), definition.Genesis.Hash)
	require.Equal(t, uint64(0x07270e00), definition.Genesis.GasLimit)
	require.Equal(t, uint64(7), definition.Genesis.BaseFee.Uint64())
	require.Equal(t, common.HexToHash("0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"), *definition.Genesis.RequestsHash)
	require.Len(t, definition.Pre, 1)
	require.Len(t, *definition.PostState, 1)
}
