package app

import "testing"

func TestDashboardURLWithTokenEscapesQueryValue(t *testing.T) {
	got := dashboardURLWithToken("http://127.0.0.1:8765", `a token&with="quotes"`)
	want := `http://127.0.0.1:8765?token=a+token%26with%3D%22quotes%22`

	if got != want {
		t.Fatalf("expected escaped dashboard URL %q, got %q", want, got)
	}
}
