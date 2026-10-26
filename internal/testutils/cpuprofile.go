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

package testutils

import (
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
	"testing"
	"time"
)

// CPUProfileUntilSampled runs work under pprof.StartCPUProfile, retrying
// with a doubled duration until the profile traces contain needle.
//
// This is the GH-983 regression check: SIGPROF must unwind through assembly
// without crashing or corrupting g, and the resulting traces must include
// the assembler routine's callers.
func CPUProfileUntilSampled(t *testing.T, needle string, work func()) string {
	t.Helper()

	duration := time.Second
	dir := t.TempDir()
	var traces []byte
	var stop func()
	defer func() {
		if stop != nil {
			stop()
		}
	}()
	for range 5 {
		f, err := os.CreateTemp(dir, "cpu.prof")
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			t.Skipf("CPU profiling is already enabled: %v", err)
		}
		stopped := false
		stop = func() {
			if stopped {
				return
			}
			stopped = true
			pprof.StopCPUProfile()
		}

		deadline := time.Now().Add(duration)
		for time.Now().Before(deadline) {
			work()
		}
		stop()
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}

		traces, err = exec.Command("go", "tool", "pprof", "-symbolize=none", "-traces", f.Name()).CombinedOutput()
		if err != nil {
			t.Fatalf("go tool pprof -traces: %v\n%s", err, traces)
		}
		if strings.Contains(string(traces), needle) {
			return string(traces)
		}
		duration *= 2
	}
	t.Fatalf("CPU profile contains no %s samples after 5 attempts", needle)
	return ""
}

// AssertAssemblyTraceback requires every sample of needle to contain the
// given frames in order, matching the GH-983 memset profiler check.
func AssertAssemblyTraceback(t *testing.T, traces, needle string, frames []string) {
	t.Helper()

	want := strings.Join(frames, "\n")
	for trace := range strings.SplitSeq(traces, "-----------") {
		if !strings.Contains(trace, needle) {
			continue
		}
		trace = strings.ReplaceAll(trace, "(inline)", "")
		trace = strings.Join(strings.Fields(trace), "\n")
		if !strings.Contains(trace, want) {
			t.Errorf("invalid %s traceback:\n%s", needle, trace)
		}
	}
}
