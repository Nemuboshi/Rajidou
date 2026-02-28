package util

import "testing"

func TestBuildProgramFileName(t *testing.T) {
	got := BuildProgramFileName("SORA to HOSHI no ORCHESTRA", "20260211230000")
	want := "SORA to HOSHI no ORCHESTRA - 20260211.aac"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestBuildProgramFileNameSanitize(t *testing.T) {
	got := BuildProgramFileName(`A/B:C*D?"E<F>G|`, "20260211230000")
	want := "A_B_C_D_E_F_G_ - 20260211.aac"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
