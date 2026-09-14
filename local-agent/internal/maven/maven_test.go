package maven_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"oncode/local-agent/internal/build"
	"oncode/local-agent/internal/maven"
	"oncode/local-agent/internal/testrun"
)

func sampleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "sample-java-maven"))
	if _, err := os.Stat(filepath.Join(root, "pom.xml")); err != nil {
		t.Fatalf("sample project missing: %v", err)
	}
	return root
}

func requireMaven(t *testing.T, root string) {
	t.Helper()
	if _, err := maven.ResolveExecutable(root); err != nil {
		t.Skip("maven not available: ", err)
	}
	if _, err := exec.LookPath("java"); err != nil {
		// wrapper/mvn may still find java via JAVA_HOME; don't hard skip
		t.Log("java not on PATH; maven may still work")
	}
}

func TestMavenBuildAndTestSample(t *testing.T) {
	root := sampleRoot(t)
	requireMaven(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	br, err := (maven.BuildAdapter{}).Run(ctx, build.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if br.Status != "SUCCESS" {
		t.Fatalf("build status=%s exit=%d stderr=%s", br.Status, br.ExitCode, br.StderrSummary)
	}

	tr, err := (maven.TestAdapter{}).Run(ctx, testrun.Request{WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if tr.Status != "SUCCESS" {
		t.Fatalf("test status=%s exit=%d out=%s", tr.Status, tr.ExitCode, tr.StdoutSummary)
	}
	if tr.Total < 2 || tr.Failed != 0 {
		t.Fatalf("counts=%+v", tr)
	}
}

func TestDetect(t *testing.T) {
	root := sampleRoot(t)
	if !maven.Detect(root) {
		t.Fatal("expected detect")
	}
	if maven.Detect(t.TempDir()) {
		t.Fatal("empty dir should not detect")
	}
}
