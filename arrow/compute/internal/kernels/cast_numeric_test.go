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

package kernels

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
)

func TestCastNumeric16BitToFloat64(t *testing.T) {
	const n = 64
	u16 := make([]uint16, n)
	i16 := make([]int16, n)
	for i := range n {
		u16[i] = uint16(i*13 + 1)
		i16[i] = int16(i*7 - 40)
	}

	gotU := make([]float64, n)
	castNumericUnsafe(arrow.UINT16, arrow.FLOAT64, arrow.Uint16Traits.CastToBytes(u16), arrow.Float64Traits.CastToBytes(gotU), n)
	for i, v := range u16 {
		if gotU[i] != float64(v) {
			t.Fatalf("uint16[%d]=%d: got %v, want %v", i, v, gotU[i], float64(v))
		}
	}

	gotI := make([]float64, n)
	castNumericUnsafe(arrow.INT16, arrow.FLOAT64, arrow.Int16Traits.CastToBytes(i16), arrow.Float64Traits.CastToBytes(gotI), n)
	for i, v := range i16 {
		if gotI[i] != float64(v) {
			t.Fatalf("int16[%d]=%d: got %v, want %v", i, v, gotI[i], float64(v))
		}
	}
}
