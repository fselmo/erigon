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

// Package stderrlog points the root logger at stderr when evm starts, so the
// env override warnings erigon logs while its packages initialize stay off
// stdout, which carries evm's results. Go initializes this package before
// common/dbg: it imports only the log package, and its import path sorts first.
package stderrlog

import "github.com/erigontech/erigon/common/log/v3"

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlWarn, log.StderrHandler))
}
