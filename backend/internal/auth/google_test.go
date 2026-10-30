package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVerify(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	enc := base64.RawURLEncoding
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"keys":[{"kid":"k1","n":%q,"e":%q}]}`,
			enc.EncodeToString(key.N.Bytes()), enc.EncodeToString(big.NewInt(int64(key.E)).Bytes()))
	}))
	defer srv.Close()

	sign := func(claims map[string]any) string {
		h, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "k1"})
		c, _ := json.Marshal(claims)
		in := enc.EncodeToString(h) + "." + enc.EncodeToString(c)
		sum := sha256.Sum256([]byte(in))
		sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
		return in + "." + enc.EncodeToString(sig)
	}
	v := NewVerifier("my-client")
	v.CertsURL = srv.URL
	valid := map[string]any{"sub": "123", "email": "a@b.c", "iss": "https://accounts.google.com", "aud": "my-client", "exp": time.Now().Add(time.Hour).Unix()}

	c, err := v.Verify(context.Background(), sign(valid))
	if err != nil || c.Sub != "123" {
		t.Fatalf("valid token: %v %+v", err, c)
	}
	for name, mutate := range map[string]func(m map[string]any){
		"audience": func(m map[string]any) { m["aud"] = "other" },
		"expired":  func(m map[string]any) { m["exp"] = time.Now().Add(-time.Hour).Unix() },
		"issuer":   func(m map[string]any) { m["iss"] = "evil.com" },
	} {
		m := map[string]any{}
		for k, val := range valid {
			m[k] = val
		}
		mutate(m)
		if _, err := v.Verify(context.Background(), sign(m)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	tok := sign(valid)
	if _, err := v.Verify(context.Background(), tok[:len(tok)-4]+"AAAA"); err == nil {
		t.Error("tampered signature accepted")
	}
}
