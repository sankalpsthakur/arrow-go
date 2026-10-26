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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var objdumpInsn = regexp.MustCompile(`\.s:(\d+)\s+0x[0-9a-f]+\s+([0-9a-f]{8})\s+(.*)$`)

// objdumpViolation reports GH-983 problems in a go tool objdump instruction:
// writes to SP that the assembler cannot see when they are encoded as WORD,
// and writes to Go's reserved arm64 registers (R18 platform, R27 REGTMP, R28 g).
func objdumpViolation(insn string) string {
	insn = strings.TrimSpace(insn)
	if insn == "" || insn == "?" || strings.HasPrefix(insn, "RET") {
		return ""
	}
	fields := strings.Fields(insn)
	if len(fields) == 0 {
		return ""
	}
	mnem := fields[0]
	dest := insn
	if i := strings.LastIndex(insn, ","); i >= 0 {
		dest = strings.TrimSpace(insn[i+1:])
	}
	switch dest {
	case "R18", "R27", "R28", "g":
		return "writes reserved register " + dest
	case "RSP":
		return "writes SP"
	}
	if strings.Contains(insn, "RSP") && (strings.Contains(mnem, ".W") || strings.Contains(mnem, ".P")) {
		return "SP writeback"
	}
	return ""
}

func TestGoObjdumpDecodesHiddenSPAndR28(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bad.s")
	obj := filepath.Join(dir, "bad.o")
	if err := os.WriteFile(src, []byte(`TEXT ·dummy(SB), 516, $0
	WORD $0xa9bf7bfd
	WORD $0xaa0003fc
	RET
`), 0o644); err != nil {
		t.Fatal(err)
	}
	asm := exec.Command("go", "tool", "asm", "-o", obj, src)
	asm.Env = append(os.Environ(), "GOARCH=arm64")
	if out, err := asm.CombinedOutput(); err != nil {
		t.Fatalf("go tool asm: %v\n%s", err, out)
	}
	dump, err := exec.Command("go", "tool", "objdump", obj).CombinedOutput()
	if err != nil {
		t.Fatalf("go tool objdump: %v\n%s", err, dump)
	}
	text := string(dump)
	if !strings.Contains(text, "RSP") || !strings.Contains(text, "R28") {
		t.Fatalf("objdump did not decode hidden SP/R28 writes:\n%s", text)
	}
	foundSP, foundG := false, false
	for _, line := range strings.Split(text, "\n") {
		m := objdumpInsn.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		v := objdumpViolation(strings.TrimSpace(m[3]))
		if strings.Contains(v, "SP") {
			foundSP = true
		}
		if strings.Contains(v, "R28") {
			foundG = true
		}
	}
	if !foundSP || !foundG {
		t.Fatalf("did not classify objdump output as SP/R28 writes (sp=%v g=%v):\n%s", foundSP, foundG, text)
	}
}

func TestObjdumpViolationDetectsGH983(t *testing.T) {
	cases := []struct {
		insn string
		want string
	}{
		{"STP.W (R29, R30), -16(RSP)", "SP writeback"},
		{"LDP.P 16(RSP), (R29, R30)", "SP writeback"},
		{"SUB $16, RSP, RSP", "writes SP"},
		{"ADD $16, RSP, RSP", "writes SP"},
		{"MOVD R0, R28", "writes reserved register R28"},
		{"VMOV V2.H[1], R18", "writes reserved register R18"},
		{"MOVD R0, R27", "writes reserved register R27"},
		{"MOVD R28, R0", ""},
		{"SUB $400, RSP, R9", ""},
		{"MOVD 8(RSP), R0", ""},
		{"VMOV V2.H[1], R7", ""},
		{"ADD R1, R0, R8", ""},
	}
	for _, tc := range cases {
		got := objdumpViolation(tc.insn)
		if tc.want == "" {
			if got != "" {
				t.Errorf("%q: unexpected violation %q", tc.insn, got)
			}
			continue
		}
		if !strings.Contains(got, tc.want) {
			t.Errorf("%q: got %q, want substring %q", tc.insn, got, tc.want)
		}
	}
}

var wordHex = regexp.MustCompile(`WORD\s+\$0x([0-9a-fA-F]+)`)

type wordSite struct {
	file string
	line int
	enc  uint32
}

func TestArm64WORDInstructionsAreProfilerSafe(t *testing.T) {
	root := moduleRoot(t)
	var sites []wordSite
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "_lib", "vendor", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_arm64.s") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNo := 0
		for sc.Scan() {
			lineNo++
			m := wordHex.FindStringSubmatch(sc.Text())
			if m == nil {
				continue
			}
			enc, err := strconv.ParseUint(m[1], 16, 32)
			if err != nil {
				return fmt.Errorf("%s:%d: %w", rel, lineNo, err)
			}
			sites = append(sites, wordSite{file: rel, line: lineNo, enc: uint32(enc)})
		}
		return sc.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) == 0 {
		t.Fatal("no WORD encodings found in *_arm64.s; scanner likely walked the wrong tree")
	}

	const chunkSize = 400
	for i := 0; i < len(sites); i += chunkSize {
		end := i + chunkSize
		if end > len(sites) {
			end = len(sites)
		}
		checkWORDChunk(t, sites[i:end])
	}
}

func checkWORDChunk(t *testing.T, sites []wordSite) {
	t.Helper()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "words.s")
	objPath := filepath.Join(dir, "words.o")

	var b strings.Builder
	b.WriteString("TEXT ·dummy(SB), 516, $0\n")
	for _, s := range sites {
		fmt.Fprintf(&b, "\tWORD $%#010x\n", s.enc)
	}
	b.WriteString("\tRET\n")
	if err := os.WriteFile(srcPath, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	asm := exec.Command("go", "tool", "asm", "-o", objPath, srcPath)
	asm.Env = append(os.Environ(), "GOARCH=arm64")
	if out, err := asm.CombinedOutput(); err != nil {
		t.Fatalf("go tool asm: %v\n%s", err, out)
	}
	dump, err := exec.Command("go", "tool", "objdump", objPath).CombinedOutput()
	if err != nil {
		t.Fatalf("go tool objdump: %v\n%s", err, dump)
	}

	for _, line := range strings.Split(string(dump), "\n") {
		m := objdumpInsn.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		dummyLine, _ := strconv.Atoi(m[1])
		text := strings.TrimSpace(m[3])
		if dummyLine < 2 {
			continue
		}
		idx := dummyLine - 2
		if idx < 0 || idx >= len(sites) {
			continue
		}
		if v := objdumpViolation(text); v != "" {
			s := sites[idx]
			t.Errorf("%s:%d WORD $%#08x: %s (%s)", s.file, s.line, s.enc, v, text)
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		p := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(p); err == nil && strings.Contains(string(data), "module github.com/apache/arrow-go") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("module root not found from ", dir)
		}
		dir = parent
	}
}
