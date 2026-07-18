package embeddedmodels

import (
	"testing"

	"github.com/hybridgroup/yzma/pkg/download"
)

func TestLlamaReleaseURL_WindowsCUDA(t *testing.T) {
	arch, err := download.ParseArch("amd64")
	if err != nil {
		t.Fatal(err)
	}
	os, err := download.ParseOS("windows")
	if err != nil {
		t.Fatal(err)
	}
	prcssr, err := download.ParseProcessor("cuda")
	if err != nil {
		t.Fatal(err)
	}

	_, filename, err := llamaReleaseURL(arch, os, prcssr, "b9935")
	if err != nil {
		t.Fatal(err)
	}
	want := "llama-b9935-bin-win-cuda-13.3-x64.zip"
	if filename != want {
		t.Fatalf("got %q want %q", filename, want)
	}
}

func TestWindowsCudaArtifactNames(t *testing.T) {
	if got := windowsCudaLlamaZip("b9935", "13.3"); got != "llama-b9935-bin-win-cuda-13.3-x64.zip" {
		t.Fatalf("unexpected llama zip name: %s", got)
	}
	if got := windowsCudaCudartZip("13.3"); got != "cudart-llama-bin-win-cuda-13.3-x64.zip" {
		t.Fatalf("unexpected cudart zip name: %s", got)
	}
}
