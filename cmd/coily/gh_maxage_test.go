package main

import (
	"reflect"
	"testing"
	"time"
)

func TestExtractGHMaxAge_Shapes(t *testing.T) {
	cases := []struct {
		name     string
		argv     []string
		wantArgv []string
		wantD    time.Duration
		wantOK   bool
	}{
		{
			name:     "absent",
			argv:     []string{"api", "/repos/o/r/issues/1"},
			wantArgv: []string{"api", "/repos/o/r/issues/1"},
			wantOK:   false,
		},
		{
			name:     "space-separated",
			argv:     []string{"--max-age", "30s", "api", "/repos/o/r/issues/1"},
			wantArgv: []string{"api", "/repos/o/r/issues/1"},
			wantD:    30 * time.Second,
			wantOK:   true,
		},
		{
			name:     "equals form",
			argv:     []string{"api", "/repos/o/r/issues/1", "--max-age=2m"},
			wantArgv: []string{"api", "/repos/o/r/issues/1"},
			wantD:    2 * time.Minute,
			wantOK:   true,
		},
		{
			name:     "zero value",
			argv:     []string{"--max-age=0", "api", "/repos/o/r/issues/1"},
			wantArgv: []string{"api", "/repos/o/r/issues/1"},
			wantD:    0,
			wantOK:   true,
		},
		{
			name:     "malformed leaves argv unchanged",
			argv:     []string{"--max-age", "foo", "api", "/repos/o/r/issues/1"},
			wantArgv: []string{"--max-age", "foo", "api", "/repos/o/r/issues/1"},
			wantOK:   false,
		},
		{
			name:     "missing value",
			argv:     []string{"api", "/repos/o/r/issues/1", "--max-age"},
			wantArgv: []string{"api", "/repos/o/r/issues/1", "--max-age"},
			wantOK:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotArgv, gotD, gotOK := extractGHMaxAge(tc.argv)
			if gotOK != tc.wantOK {
				t.Errorf("ok=%v want %v", gotOK, tc.wantOK)
			}
			if tc.wantOK && gotD != tc.wantD {
				t.Errorf("d=%v want %v", gotD, tc.wantD)
			}
			if !reflect.DeepEqual(gotArgv, tc.wantArgv) {
				t.Errorf("argv=%v want %v", gotArgv, tc.wantArgv)
			}
		})
	}
}

func TestResolveGHMaxAge_CLIWinsOverEnv(t *testing.T) {
	t.Setenv("COILY_GH_MAX_AGE", "1h")
	_, d, ok := resolveGHMaxAge([]string{"--max-age=10s", "api", "/repos/o/r/issues/1"})
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if d != 10*time.Second {
		t.Errorf("d=%v want 10s (CLI overrides env)", d)
	}
}

func TestResolveGHMaxAge_EnvFallback(t *testing.T) {
	t.Setenv("COILY_GH_MAX_AGE", "0")
	argv, d, ok := resolveGHMaxAge([]string{"api", "/repos/o/r/issues/1"})
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if d != 0 {
		t.Errorf("d=%v want 0", d)
	}
	// Env-only path: argv untouched.
	if !reflect.DeepEqual(argv, []string{"api", "/repos/o/r/issues/1"}) {
		t.Errorf("argv=%v want unchanged", argv)
	}
}

func TestResolveGHMaxAge_MalformedEnvIgnored(t *testing.T) {
	t.Setenv("COILY_GH_MAX_AGE", "garbage")
	_, _, ok := resolveGHMaxAge([]string{"api", "/repos/o/r/issues/1"})
	if ok {
		t.Errorf("malformed env: expected ok=false")
	}
}

func TestRewriteGHForRESTAndJQFile_StripsMaxAgeAndStashes(t *testing.T) {
	pendingGHMaxAge.Store(nil)
	out := rewriteGHForRESTAndJQFile([]string{"api", "/repos/o/r/issues/1", "--max-age=45s"})
	if !reflect.DeepEqual(out, []string{"api", "/repos/o/r/issues/1"}) {
		t.Errorf("argv after rewrite=%v want %v", out, []string{"api", "/repos/o/r/issues/1"})
	}
	if got := loadGHMaxAge(); got != 45*time.Second {
		t.Errorf("stashed max-age=%v want 45s", got)
	}
}

func TestLoadGHMaxAge_DefaultIsNegativeOne(t *testing.T) {
	pendingGHMaxAge.Store(nil)
	if got := loadGHMaxAge(); got != -1 {
		t.Errorf("default loadGHMaxAge=%v want -1 (no cap)", got)
	}
}

func TestGHReadCacheClassifier_SurfacesStashedMaxAge(t *testing.T) {
	d := 5 * time.Second
	pendingGHMaxAge.Store(&d)
	t.Cleanup(func() { pendingGHMaxAge.Store(nil) })
	_, gotD, ok := ghReadCacheClassifier([]string{"api", "/repos/o/r/issues/1"})
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if gotD != d {
		t.Errorf("classifier max-age=%v want %v", gotD, d)
	}
}
