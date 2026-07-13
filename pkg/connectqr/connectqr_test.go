package connectqr

import "testing"

func TestEncode(t *testing.T) {
	got := Encode("http://10.8.1.1:8080", "secret")
	want := `{"server":"http://10.8.1.1:8080","token":"secret"}`
	if got != want {
		t.Fatalf("Encode() = %q, want %q", got, want)
	}
}
