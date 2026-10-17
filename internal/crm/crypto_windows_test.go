//go:build windows

package crm

import (
	"bytes"
	"testing"
)

func TestDPAPIRoundTrip(t *testing.T) {
	plain := []byte("test-only-credential")
	a, e := protect(plain)
	if e != nil {
		t.Fatal(e)
	}
	b, e := protect(plain)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Equal(a, plain) || bytes.Equal(a, b) {
		t.Fatal("cipher must be randomized")
	}
	out, e := unprotect(a)
	if e != nil || !bytes.Equal(out, plain) {
		t.Fatalf("roundtrip: %v", e)
	}
	a[0] ^= 255
	if _, e = unprotect(a); e == nil {
		t.Fatal("accepted corrupted data")
	}
}
