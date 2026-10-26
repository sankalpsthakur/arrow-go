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

//go:build go1.18 && arm64 && !noasm && !appengine

package kernels

import (
	"math"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"golang.org/x/sys/cpu"
)

func TestCastNumericNeon16BitToFloat64(t *testing.T) {
	if !cpu.ARM64.HasASIMD {
		t.Skip("ARM64 SIMD is not available")
	}

	// The vector loops take 16 values at a time.
	const n = 64
	u16 := make([]uint16, n)
	i16 := make([]int16, n)
	for i := 0; i < n; i++ {
		u16[i] = uint16(i * 1031)
		i16[i] = int16(i*1031 - 32768)
	}
	u16[n-1], i16[n-1] = math.MaxUint16, math.MaxInt16

	gotU := make([]float64, n)
	castNumericNeon(arrow.UINT16, arrow.FLOAT64, arrow.Uint16Traits.CastToBytes(u16), arrow.Float64Traits.CastToBytes(gotU), n)
	for i, v := range u16 {
		if gotU[i] != float64(v) {
			t.Errorf("uint16 %d at index %d: got %v", v, i, gotU[i])
		}
	}

	gotI := make([]float64, n)
	castNumericNeon(arrow.INT16, arrow.FLOAT64, arrow.Int16Traits.CastToBytes(i16), arrow.Float64Traits.CastToBytes(gotI), n)
	for i, v := range i16 {
		if gotI[i] != float64(v) {
			t.Errorf("int16 %d at index %d: got %v", v, i, gotI[i])
		}
	}
}
