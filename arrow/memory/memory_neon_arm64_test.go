// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file to you under the
// Apache License, Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build arm64 && !noasm

package memory_test

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/internal/testutils"
)

func TestNEONAssemblyTracebacks(t *testing.T) {
	// This is a regression test for GH-983. _memset_neon used to modify the
	// stack pointer in a way that the assembler didn't see, leading to
	// broken tracebacks in CPU profiler samples when the function is
	// running.

	buf := make([]byte, 1<<20)
	traces := testutils.CPUProfileUntilSampled(t, "memory._memset_neon", func() {
		memory.Set(buf, 0x1f)
	})
	testutils.AssertAssemblyTraceback(t, traces, "memory._memset_neon", []string{
		"github.com/apache/arrow-go/v18/arrow/memory._memset_neon",
		"github.com/apache/arrow-go/v18/arrow/memory.memory_memset_neon",
		"github.com/apache/arrow-go/v18/arrow/memory.Set",
		"github.com/apache/arrow-go/v18/arrow/memory_test.TestNEONAssemblyTracebacks",
	})
}
