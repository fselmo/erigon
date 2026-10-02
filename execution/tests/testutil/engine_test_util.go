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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/erigontech/erigon/common"
	"github.com/erigontech/erigon/common/hexutil"
	"github.com/erigontech/erigon/execution/engineapi"
	"github.com/erigontech/erigon/execution/engineapi/engine_block_downloader"
	enginetypes "github.com/erigontech/erigon/execution/engineapi/engine_types"
	"github.com/erigontech/erigon/execution/execmodule/chainreader"
	"github.com/erigontech/erigon/execution/p2p"
	"github.com/erigontech/erigon/node/ethconfig"
	"github.com/erigontech/erigon/rpc"
)

// An EngineTest checks a chain delivered as engine API calls: a
// blockchain_tests_engine fixture. Genesis, post-state checks and runner
// settings are shared with BlockTest.
type EngineTest struct {
	BlockTest
	payloads []enginePayload
}

type enginePayload struct {
	Params            []json.RawMessage `json:"params"`
	NewPayloadVersion string            `json:"newPayloadVersion"`
	FcuVersion        string            `json:"forkchoiceUpdatedVersion"`
	// ValidationError and ErrorCode mark a payload the client must reject,
	// with an invalid status or an error from the call.
	ValidationError string `json:"validationError"`
	ErrorCode       string `json:"errorCode"`
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (et *EngineTest) UnmarshalJSON(in []byte) error {
	if err := et.BlockTest.UnmarshalJSON(in); err != nil {
		return err
	}
	var fixture struct {
		Payloads []enginePayload `json:"engineNewPayloads"`
	}
	if err := json.Unmarshal(in, &fixture); err != nil {
		return err
	}
	et.payloads = fixture.Payloads
	return nil
}

// RunCLI runs the test on a fresh execution module. Each payload and
// forkchoice update is a direct call to an engine server built on that
// module, at the version the fixture names; nothing listens on a port.
func (et *EngineTest) RunCLI() error {
	m, err := et.newTester(nil)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := et.validateGenesis(m); err != nil {
		return err
	}
	// With no peers, a payload whose parent is unknown is answered SYNCING, as
	// on an isolated node.
	noPeers := p2p.NewBackwardBlockDownloader(m.Log, nil, nil, p2p.NewPeerTracker(m.Log, nil), m.Dirs.Tmp)
	srv := engineapi.NewEngineServer(
		m.Log,
		m.ChainConfig,
		m.ExecModule,
		engine_block_downloader.NewEngineBlockDownloader(m.Ctx, m.Log, m.ExecModule, m.BlockReader, m.DB, m.ChainConfig, ethconfig.Defaults.Sync, noPeers),
		false, /* caplin */
		false, /* internalCL */
		false, /* proposing */
		true,  /* consuming */
		nil,   /* txPool */
		nil,   /* blobGetter */
		ethconfig.Defaults.FcuTimeout,
		ethconfig.Defaults.MaxReorgDepth,
	)
	if len(et.payloads) > 0 {
		if err := forkchoiceUpdated(m.Ctx, srv, m.Genesis.Hash(), et.payloads[0].FcuVersion); err != nil {
			return fmt.Errorf("forkchoice update to genesis: %w", err)
		}
	}
	chain := chainreader.NewChainReaderEth1(m.ChainConfig, m.ExecModule, 0)
	for i, payload := range et.payloads {
		if err := payload.send(m.Ctx, srv, chain); err != nil {
			return fmt.Errorf("payload %d: %w", i, err)
		}
	}
	m.ExecModule.WaitIdle(m.Ctx)

	tx, err := m.DB.BeginTemporalRo(m.Ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return et.validateHeadAndPostState(m, tx)
}

// send delivers the payload through engine_newPayloadV<n> and, if it is valid,
// makes it the head through engine_forkchoiceUpdatedV<n>.
func (p *enginePayload) send(ctx context.Context, srv *engineapi.EngineServer, chain chainreader.ChainReaderWriterEth1) error {
	var (
		payload           enginetypes.ExecutionPayload
		blobHashes        []common.Hash
		parentBeaconRoot  common.Hash
		executionRequests []hexutil.Bytes
	)
	targets := []any{&payload, &blobHashes, &parentBeaconRoot, &executionRequests}
	for i, param := range p.Params {
		if i >= len(targets) {
			return fmt.Errorf("unexpected param %d", i)
		}
		if err := json.Unmarshal(param, targets[i]); err != nil {
			return fmt.Errorf("decode param %d: %w", i, err)
		}
	}

	parentUnknown := func() bool { return chain.GetHeaderByHash(ctx, payload.ParentHash) == nil }
	status, err := retrySyncing(ctx, parentUnknown, func() (*enginetypes.PayloadStatus, error) {
		switch p.NewPayloadVersion {
		case "1":
			return srv.NewPayloadV1(ctx, &payload)
		case "2":
			return srv.NewPayloadV2(ctx, &payload)
		case "3":
			return srv.NewPayloadV3(ctx, &payload, blobHashes, &parentBeaconRoot)
		case "4":
			return srv.NewPayloadV4(ctx, &payload, blobHashes, &parentBeaconRoot, executionRequests)
		case "5":
			return srv.NewPayloadV5(ctx, &payload, blobHashes, &parentBeaconRoot, executionRequests)
		default:
			return nil, fmt.Errorf("unsupported newPayload version %q", p.NewPayloadVersion)
		}
	})
	expectInvalid := p.ValidationError != "" || p.ErrorCode != ""
	switch {
	case err != nil && p.ErrorCode != "":
		return checkErrorCode(p.ErrorCode, err)
	case err != nil && expectInvalid:
		return checkException(p.ValidationError, err)
	case err != nil:
		return err
	case status.Status != enginetypes.ValidStatus && expectInvalid:
		return checkException(p.ValidationError, statusError("payload", status))
	case status.Status != enginetypes.ValidStatus:
		return statusError("payload", status)
	case expectInvalid:
		return fmt.Errorf("payload is valid, expected %s%s", p.ValidationError, p.ErrorCode)
	}
	return forkchoiceUpdated(ctx, srv, payload.BlockHash, p.FcuVersion)
}

func forkchoiceUpdated(ctx context.Context, srv *engineapi.EngineServer, head common.Hash, version string) error {
	state := &enginetypes.ForkChoiceState{HeadHash: head}
	status, err := retrySyncing(ctx, nil, func() (*enginetypes.PayloadStatus, error) {
		var (
			r   *enginetypes.ForkChoiceUpdatedResponse
			err error
		)
		switch version {
		case "1":
			r, err = srv.ForkchoiceUpdatedV1(ctx, state, nil)
		case "2":
			r, err = srv.ForkchoiceUpdatedV2(ctx, state, nil)
		case "3":
			r, err = srv.ForkchoiceUpdatedV3(ctx, state, nil)
		case "4":
			r, err = srv.ForkchoiceUpdatedV4(ctx, state, nil, nil)
		default:
			return nil, fmt.Errorf("unsupported forkchoiceUpdated version %q", version)
		}
		if err != nil {
			return nil, err
		}
		return r.PayloadStatus, nil
	})
	if err != nil {
		return err
	}
	if status.Status != enginetypes.ValidStatus {
		return statusError("forkchoice", status)
	}
	return nil
}

// checkErrorCode checks that a call the fixture expects to fail with a
// JSON-RPC error returned that error's code. As in EEST's consume, the code
// alone decides; the payload's validation error is not checked.
func checkErrorCode(expected string, err error) error {
	var rpcErr rpc.Error
	if !errors.As(err, &rpcErr) {
		return fmt.Errorf("expected error code %s, got an error without a code: %w", expected, err)
	}
	if code := strconv.Itoa(rpcErr.ErrorCode()); code != expected {
		return fmt.Errorf("expected error code %s, got %s: %w", expected, code, err)
	}
	return nil
}

func statusError(call string, status *enginetypes.PayloadStatus) error {
	if status.ValidationError == nil || status.ValidationError.Error() == nil {
		return fmt.Errorf("%s status %s", call, status.Status)
	}
	return fmt.Errorf("%s status %s: %w", call, status.Status, status.ValidationError.Error())
}

// retrySyncing repeats call while the server answers SYNCING, which it does
// while the execution module is still busy with the previous request. A
// SYNCING answer is final once stuck, if given, reports true.
func retrySyncing(ctx context.Context, stuck func() bool, call func() (*enginetypes.PayloadStatus, error)) (*enginetypes.PayloadStatus, error) {
	for {
		status, err := call()
		if err != nil || status.Status != enginetypes.SyncingStatus {
			return status, err
		}
		if stuck != nil && stuck() {
			return status, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}
