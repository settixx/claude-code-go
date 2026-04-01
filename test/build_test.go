package test

import (
	"os"
	"os/exec"
	"testing"
)

func TestBuild_CrossCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cross-compilation in short mode")
	}

	root := projectRoot(t)

	platforms := []struct{ goos, goarch string }{
		{"darwin", "amd64"}, {"darwin", "arm64"},
		{"linux", "amd64"}, {"linux", "arm64"},
		{"windows", "amd64"}, {"windows", "arm64"},
	}

	for _, p := range platforms {
		t.Run(p.goos+"/"+p.goarch, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			outPath := tmpDir + "/ticode"
			if p.goos == "windows" {
				outPath += ".exe"
			}

			cmd := exec.Command("go", "build", "-o", outPath, "./cmd/ticode")
			cmd.Dir = root
			cmd.Env = append(os.Environ(),
				"GOOS="+p.goos,
				"GOARCH="+p.goarch,
				"CGO_ENABLED=0",
			)

			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("build failed for %s/%s:\n%s\nerror: %v", p.goos, p.goarch, out, err)
			}
		})
	}
}
