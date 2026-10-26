// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build arm64 && !noasm

package utils_test

import (
	"bytes"
	"testing"

	"github.com/apache/arrow-go/v18/internal/testutils"
	"github.com/apache/arrow-go/v18/parquet/internal/utils"
)

func TestNEONUnpack32CPUProfileTracebacks(t *testing.T) {
	// Regression for GH-983: _unpack32_neon used R28 and WORD-encoded SP
	// updates, so CGO_ENABLED=0 CPU profiling crashed in sigprof and
	// traces were truncated even when it did not crash.
	const batchSize = 512
	input := make([]byte, batchSize*4)
	for i := range input {
		input[i] = byte(i)
	}
	output := make([]uint64, batchSize)
	reader := utils.NewBitReader(bytes.NewReader(input))

	traces := testutils.CPUProfileUntilSampled(t, "utils._unpack32_neon", func() {
		reader.Reset(bytes.NewReader(input))
		_, _ = reader.GetBatch(32, output)
	})
	testutils.AssertAssemblyTraceback(t, traces, "utils._unpack32_neon", []string{
		"github.com/apache/arrow-go/v18/parquet/internal/utils._unpack32_neon",
		"github.com/apache/arrow-go/v18/parquet/internal/utils.unpack32NEON",
		"github.com/apache/arrow-go/v18/parquet/internal/utils.(*BitReader).GetBatch",
		"github.com/apache/arrow-go/v18/parquet/internal/utils_test.TestNEONUnpack32CPUProfileTracebacks",
	})
}

func TestNEONBytesToBoolsCPUProfileTracebacks(t *testing.T) {
	in := make([]byte, 4096)
	for i := range in {
		in[i] = byte(i * 3)
	}
	out := make([]bool, len(in)*8)

	traces := testutils.CPUProfileUntilSampled(t, "utils._bytes_to_bools_neon", func() {
		utils.BytesToBools(in, out)
	})
	testutils.AssertAssemblyTraceback(t, traces, "utils._bytes_to_bools_neon", []string{
		"github.com/apache/arrow-go/v18/parquet/internal/utils._bytes_to_bools_neon",
		"github.com/apache/arrow-go/v18/parquet/internal/utils.bytesToBoolsNEON",
		"github.com/apache/arrow-go/v18/parquet/internal/utils.BytesToBools",
		"github.com/apache/arrow-go/v18/parquet/internal/utils_test.TestNEONBytesToBoolsCPUProfileTracebacks",
	})
}
