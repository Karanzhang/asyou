package validation

import "testing"

func TestValidEmail(t *testing.T) {
	cases := map[string]bool{
		"alice@example.com": true,
		"bob.smith+tag@sub.domain.co": true,
		"invalid@": false,
		"@no-local.com": false,
		"noatsymbol.com": false,
	}
	for s, want := range cases {
		if ValidEmail(s) != want {
			t.Fatalf("ValidEmail(%q) = %v, want %v", s, !want, want)
		}
	}
}

func TestValidPort(t *testing.T) {
	if !ValidPort(80) {
		t.Fatal("80 should be valid")
	}
	if ValidPort(0) {
		t.Fatal("0 should be invalid")
	}
	if ValidPort(70000) {
		t.Fatal("70000 should be invalid")
	}
}

func TestValidProxyType(t *testing.T) {
	valid := []string{"tcp", "http", "https", "udp", "stcp", "xtcp", "tcpmux"}
	for _, v := range valid {
		if !ValidProxyType(v) {
			t.Fatalf("%s should be valid proxy type", v)
		}
	}
	if ValidProxyType("ftp") {
		t.Fatal("ftp should be invalid")
	}
}
