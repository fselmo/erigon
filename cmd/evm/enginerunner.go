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
	"github.com/erigontech/erigon/execution/engineapi/engineapitester"
	"github.com/erigontech/erigon/node/ethconfig"
)

var engineTestCommand = cli.Command{
	Action:    engineTestCmd,
	Name:      "enginetest",
	Usage:     "Executes the given engine API tests (blockchain_tests_engine fixtures)",
	ArgsUsage: "<path>",
	Description: "Each test runs on its own in-process Erigon node, whose engine API\n" +
		"receives the test's payloads and forkchoice updates at the versions the\n" +
		"test names. Node datadirs live under $TMPDIR; see enginextest for tips.",
	Flags: []cli.Flag{
		&ExecSerialFlag,
		&JSONOutputFlag,
		&RunFlag,
		&ExcludeFlag,
		&VerbosityFlag,
		&WorkersFlag,
	},
}

func engineTestCmd(ctx context.Context, cliCtx *cli.Command) error {
	if cliCtx.Int(VerbosityFlag.Name) > 0 {
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(cliCtx.Int(VerbosityFlag.Name)), log.StderrHandler))
	} else {
		log.Root().SetHandler(log.LvlFilterHandler(log.LvlError, log.StderrHandler))
	}

	path := cliCtx.Args().First()
	if path == "" {
		return errors.New("path argument required")
	}
	workers := cliCtx.Uint64(WorkersFlag.Name)
	if workers == 0 {
		return fmt.Errorf("--%s must be >= 1", WorkersFlag.Name)
	}
	filter, err := compileTestFilter(cliCtx.String(RunFlag.Name), cliCtx.StringSlice(ExcludeFlag.Name))
	if err != nil {
		return err
	}

	execSerial := cliCtx.Bool(ExecSerialFlag.Name)
	reportPath := executionPathReporter(execSerial)
	opts := []engineapitester.EngineXTestRunnerOption{
		engineapitester.WithEthConfigTweaker(func(cfg *ethconfig.Config) {
			if execSerial {
				cfg.Sync.ExecWorkerCount = 1
			}
			cfg.Sync.ExecutionPathReporter = reportPath
		}),
	}

	files := filter.filterFiles(collectFiles(path))
	results, err := runTestFilesParallel(files, workers, func(fname string) ([]testResult, error) {
		return runEngineTest(ctx, fname, filter, opts)
	})
	if err != nil {
		return err
	}
	report(cliCtx, results)
	return nil
}

func runEngineTest(ctx context.Context, fname string, filter testFilter, opts []engineapitester.EngineXTestRunnerOption) ([]testResult, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var tests map[string]engineapitester.EngineTestDefinition
	if err := json.Unmarshal(src, &tests); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", fname, err)
	}

	results := make([]testResult, 0, len(tests))
	for _, name := range slices.Sorted(maps.Keys(tests)) {
		if !filter.includeCase(fname, name) {
			continue
		}
		result := testResult{Name: name, Pass: true}
		if err := engineapitester.RunEngineTest(ctx, log.Root(), tests[name], opts...); err != nil {
			result.Pass = false
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results, nil
}
