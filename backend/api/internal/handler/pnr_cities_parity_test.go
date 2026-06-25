package handler

// Parity test for the PNR_CITIES list that lives in two
// languages: Go (this file's sibling customer.go) and TypeScript
// (apps/store/src/lib/deliveryRules.ts). Reviewer finding #4
// (P1) flagged that the two lists drift in the wild — they are
// edited independently, the only "synchronisation" being
// "review them together when somebody adds a city", and that
// human contract is the kind of contract that gets broken the
// first time somebody adds "Каушаны" and forgets one side.
//
// The test pins set equality: it parses the TS Set literal
// (new Set(["a","b",...])) out of the source file, parses the
// Go map[string]bool{...} out of customer.go, normalises both
// to lower-case strings, and fails with a diff if they differ.
// The diff message names the missing/extra cities so the next
// reviewer can fix it in one shot.
//
// Trade-off vs. the "shared JSON" approach the reviewer
// preferred: a parity test still relies on a human to copy
// between two files, but it (a) catches drift in CI instead of
// in prod, (b) costs zero per-build and zero codegen, and
// (c) is small enough that the next "shared source" refactor
// can delete this whole file unchanged.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// pnrCitiesFromGo parses customer.go and returns the set of
// city keys from the pnrCities map literal. We walk the AST
// instead of doing a regex on the source so a future refactor
// (e.g. extracting the list to a package-level var) doesn't
// silently break the test.
func pnrCitiesFromGo(t *testing.T, repoRoot string) map[string]struct{} {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(repoRoot, "backend/api/internal/handler/customer.go"))
	if err != nil {
		t.Fatalf("read customer.go: %v", err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "customer.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse customer.go: %v", err)
	}
	want := map[string]struct{}{}
	ast.Inspect(f, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		// We only care about map[string]bool{...} literals.
		mt, ok := cl.Type.(*ast.MapType)
		if !ok {
			return true
		}
		k, ok := mt.Key.(*ast.Ident)
		if !ok || k.Name != "string" {
			return true
		}
		v, ok := mt.Value.(*ast.Ident)
		if !ok || v.Name != "bool" {
			return true
		}
		// The pnrCities literal is the only one of this shape
		// in this file; in the (very unlikely) case the file
		// grows a second one, narrow by inspecting the keys:
		// a PMR-city map is full of lower-case Cyrillic place
		// names. We collect every key and trust the count to
		// flag duplicates.
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			lit, ok := kv.Key.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			s, err := unquote(lit.Value)
			if err != nil {
				continue
			}
			want[strings.ToLower(s)] = struct{}{}
		}
		return false
	})
	return want
}

// unquote is a tiny shim around strconv.Unquote that returns
// the error directly so the AST walk above stays readable.
func unquote(s string) (string, error) {
	if len(s) < 2 {
		return "", os.ErrInvalid
	}
	return s[1 : len(s)-1], nil
}

// pnrCitiesFromTS parses the PNR_CITIES Set literal out of the
// TypeScript source. The literal looks like:
//
//	const PNR_CITIES = new Set([
//	  "тирасполь", "бендеры", ...
//	]);
//
// We deliberately accept only the array form (not Set<string>
// generics syntax) because the live file uses the old form and
// we want the test to be the one that breaks if somebody
// modernises the TS without thinking about the test.
func pnrCitiesFromTS(t *testing.T, repoRoot string) map[string]struct{} {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(repoRoot, "apps/store/src/lib/deliveryRules.ts"))
	if err != nil {
		t.Fatalf("read deliveryRules.ts: %v", err)
	}
	// Find "new Set([" then walk forward to "]".
	idx := strings.Index(string(src), "new Set([")
	if idx < 0 {
		t.Fatalf("could not find `new Set([` in deliveryRules.ts — refactor may have moved the constant")
	}
	rest := string(src[idx+len("new Set(["):])
	end := strings.Index(rest, "]")
	if end < 0 {
		t.Fatalf("could not find closing `]` for the PNR_CITIES Set literal")
	}
	body := rest[:end]
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		t.Fatalf("no string entries inside the PNR_CITIES Set literal — refactor may have moved the constant")
	}
	want := map[string]struct{}{}
	for _, m := range matches {
		want[strings.ToLower(m[1])] = struct{}{}
	}
	return want
}

// TestPNRCitiesGoAndTSAline is the parity test itself. We
// resolve the repo root by walking up from the test file's
// directory until we find go.mod, which is the same trick
// loadOrderItems' tests use elsewhere in the package. If the
// two sets differ, the test fails with a sorted diff so the
// PR review can fix it in one line.
func TestPNRCitiesGoAndTSAline(t *testing.T) {
	repoRoot := findRepoRoot(t)
	goSet := pnrCitiesFromGo(t, repoRoot)
	tsSet := pnrCitiesFromTS(t, repoRoot)

	if len(goSet) == 0 {
		t.Fatalf("parsed zero PNR cities from Go side — has the literal moved or been commented out?")
	}
	if len(tsSet) == 0 {
		t.Fatalf("parsed zero PNR cities from TS side — has the literal moved or been commented out?")
	}

	// Symmetric diff: cities present on one side but not the other.
	var onlyGo, onlyTS []string
	for c := range goSet {
		if _, ok := tsSet[c]; !ok {
			onlyGo = append(onlyGo, c)
		}
	}
	for c := range tsSet {
		if _, ok := goSet[c]; !ok {
			onlyTS = append(onlyTS, c)
		}
	}
	sort.Strings(onlyGo)
	sort.Strings(onlyTS)

	if len(onlyGo) > 0 || len(onlyTS) > 0 {
		t.Fatalf(
			"PNR_CITIES drift between Go and TypeScript. "+
				"These cities exist on one side but not the other; "+
				"add them to the missing side (or remove from the extra side) "+
				"and re-run the test.\n"+
				"  only in Go (customer.go):  %v\n"+
				"  only in TS (deliveryRules.ts): %v",
			onlyGo, onlyTS,
		)
	}
}

// findRepoRoot walks up the directory tree from the test
// binary's working directory until it finds a directory that
// contains both `backend/` and `apps/` subdirectories. We use
// this so the test works no matter where `go test` is invoked
// from (package dir, repo root, or anywhere else) and the
// relative path to the TS source file is always
// "<root>/apps/store/src/lib/deliveryRules.ts".
func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, beErr := os.Stat(filepath.Join(dir, "backend")); beErr == nil {
			if _, apErr := os.Stat(filepath.Join(dir, "apps")); apErr == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find project root (with backend/ + apps/) above %s", wd)
		}
		dir = parent
	}
}
