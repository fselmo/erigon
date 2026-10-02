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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/urfave/cli/v3"

	"github.com/erigontech/erigon/common/log/v3"
	"github.com/erigontech/erigon/execution/tests/testutil"
	"github.com/erigontech/erigon/node/ethconfig"
)

var engineTestCommand = cli.Command{
	Action:    engineTestCmd,
	Name:      "enginetest",
	Usage:     "Executes the given engine API tests (blockchain_tests_engine fixtures)",
	ArgsUsage: "<path>...",
	Flags: []cli.Flag{
		&BALReportFlag,
		&ExecSerialFlag,
		&JSONOutputFlag,
		&JSONLOutputFlag,
		&RunFlag,
		&ExcludeFlag,
		&VerbosityFlag,
		&WorkersFlag,
	},
}

func engineTestCmd(_ context.Context, ctx *cli.Command) error {
	if ctx.Int(VerbosityFlag.Name) > 0 {
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(ctx.Int(VerbosityFlag.Name)), log.StderrHandler))
	} else {
		log.Root().SetHandler(log.LvlFilterHandler(log.LvlError, log.StderrHandler))
	}

	if !ctx.Args().Present() {
		return errors.New("path argument required")
	}
	workers := ctx.Uint64(WorkersFlag.Name)
	if workers == 0 {
		return fmt.Errorf("--%s must be >= 1", WorkersFlag.Name)
	}
	filter, err := compileTestFilter(ctx.String(RunFlag.Name), ctx.StringSlice(ExcludeFlag.Name))
	if err != nil {
		return err
	}

	reportPath := runnerExecutionPathReporter(ctx)
	files := filter.filterFiles(collectArgFiles(ctx))
	results, err := runTestFilesParallel(files, workers, func(fname string) ([]testResult, error) {
		return runEngineTest(ctx, fname, filter, reportPath)
	})
	if err != nil {
		return err
	}
	report(ctx, results)
	return nil
}

func runEngineTest(ctx *cli.Command, fname string, filter testFilter, reportPath func(ethconfig.BlockExecutionPath)) ([]testResult, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var tests map[string]*testutil.EngineTest
	if err := json.Unmarshal(src, &tests); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", fname, err)
	}

	results := make([]testResult, 0, len(tests))
	for _, name := range slices.Sorted(maps.Keys(tests)) {
		if !filter.includeCase(fname, name) {
			continue
		}
		if ctx.Bool(ExecSerialFlag.Name) {
			tests[name].ExecWorkers = 1
		}
		tests[name].ExecutionPathReporter = reportPath
		result := testResult{Name: name, Pass: true}
		if err := tests[name].RunCLI(); err != nil {
			result.Pass = false
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results, nil
}
