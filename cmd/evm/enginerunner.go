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
)

var engineTestCommand = cli.Command{
	Action:    engineTestCmd,
	Name:      "enginetest",
	Usage:     "Executes the given engine API tests (blockchain_tests_engine fixtures)",
	ArgsUsage: "<path>",
	Flags: []cli.Flag{
		&JSONOutputFlag,
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

	path := ctx.Args().First()
	if path == "" {
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

	files := filter.filterFiles(collectFiles(path))
	results, err := runTestFilesParallel(files, workers, func(fname string) ([]testResult, error) {
		return runEngineTest(ctx, fname, filter)
	})
	if err != nil {
		return err
	}
	report(ctx, results)
	return nil
}

func runEngineTest(ctx *cli.Command, fname string, filter testFilter) ([]testResult, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var tests map[string]*engineapitester.EngineTest
	if err := json.Unmarshal(src, &tests); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", fname, err)
	}

	results := make([]testResult, 0, len(tests))
	for _, name := range slices.Sorted(maps.Keys(tests)) {
		if !filter.includeCase(fname, name) {
			continue
		}
		result := testResult{Name: name, Pass: true}
		if err := tests[name].RunCLI(); err != nil {
			result.Pass = false
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results, nil
}
