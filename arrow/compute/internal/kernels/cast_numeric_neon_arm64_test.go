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

package kernels

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/internal/testutils"
)

func TestNEONCastNumericCPUProfileTracebacks(t *testing.T) {
	// The uint16/int16 → float64 vector paths used WORD-encoded umov into
	// w18. R18 is reserved; SIGPROF must still observe a valid g (GH-983).
	const n = 1024
	u16 := make([]uint16, n)
	i16 := make([]int16, n)
	for i := range n {
		u16[i] = uint16(i * 13)
		i16[i] = int16(i*7 - 512)
	}
	f64 := make([]float64, n)

	traces := testutils.CPUProfileUntilSampled(t, "kernels._cast_type_numeric_neon", func() {
		castNumericUnsafe(arrow.UINT16, arrow.FLOAT64, arrow.Uint16Traits.CastToBytes(u16), arrow.Float64Traits.CastToBytes(f64), n)
		castNumericUnsafe(arrow.INT16, arrow.FLOAT64, arrow.Int16Traits.CastToBytes(i16), arrow.Float64Traits.CastToBytes(f64), n)
	})
	testutils.AssertAssemblyTraceback(t, traces, "kernels._cast_type_numeric_neon", []string{
		"github.com/apache/arrow-go/v18/arrow/compute/internal/kernels._cast_type_numeric_neon",
		"github.com/apache/arrow-go/v18/arrow/compute/internal/kernels.castNumericNeon",
		"github.com/apache/arrow-go/v18/arrow/compute/internal/kernels.TestNEONCastNumericCPUProfileTracebacks",
	})
}
