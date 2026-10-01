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
	"os"
	"sync"

	"github.com/erigontech/erigon/common"
	"github.com/erigontech/erigon/node/ethconfig"
)

// executionPathLine is one block's execution path as written to stderr.
type executionPathLine struct {
	Event  string      `json:"event"`
	Block  uint64      `json:"block"`
	Hash   common.Hash `json:"hash"`
	Path   string      `json:"path"`
	Reason string      `json:"reason"`
	// Scheduler is "bal" when the access list seeds the parallel executor and
	// "optimistic" when it does not; absent for the legacy serial executor.
	Scheduler string `json:"scheduler,omitempty"`
}

// executionPathReporter returns a reporter that writes each executed block's
// path to stderr as one JSON line. With --exec.serial on, the single worker
// is the switch's doing, so its reason reads "disabled".
func executionPathReporter(execSerial bool) func(ethconfig.BlockExecutionPath) {
	var mu sync.Mutex
	return func(p ethconfig.BlockExecutionPath) {
		line := executionPathLine{
			Event:  "balExecution",
			Block:  p.Number,
			Hash:   p.Hash,
			Path:   p.Path,
			Reason: p.Reason,
		}
		if execSerial && line.Reason == "single-worker" {
			line.Reason = "disabled"
		}
		if p.Reason != "serial-executor" {
			line.Scheduler = "optimistic"
			if p.AccessList {
				line.Scheduler = "bal"
			}
		}
		out, err := json.Marshal(line)
		if err != nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		_, _ = os.Stderr.Write(append(out, '\n'))
	}
}
