package connectqr

import "testing"

func TestEncode(t *testing.T) {
	got := Encode("http://198.51.100.10:8080", "secret")
	want := `{"server":"http://198.51.100.10:8080","token":"secret"}`
	if got != want {
		t.Fatalf("Encode() = %q, want %q", got, want)
	}
}
