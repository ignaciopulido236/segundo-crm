package crm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testAPI(t *testing.T, handler http.HandlerFunc) *API {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &API{server.URL, "sentinel-secret", &http.Client{Timeout: 45 * time.Second, CheckRedirect: redirect}}
}
func TestContract(t *testing.T) {
	for _, action := range []string{"list", "get", "create", "update", "delete"} {
		t.Run(action, func(t *testing.T) {
			a := testAPI(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
					t.Error("not JSON POST")
				}
				var body map[string]interface{}
				if json.NewDecoder(r.Body).Decode(&body) != nil {
					t.Error("decode")
				}
				if body["api_key"] != "sentinel-secret" || body["action"] != action {
					t.Error("missing contract fields")
				}
				if r.URL.RawQuery != "" {
					t.Error("secret in URL")
				}
				status := 200
				if action == "create" {
					status = 201
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "status": status, "data": map[string]string{"id": "id"}})
			})
			_, e := a.call(action, map[string]interface{}{"id": "id", "data": map[string]string{"telefono": "+00012"}})
			if e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestErrors(t *testing.T) {
	cases := []struct {
		name, body, kind string
		http             int
	}{
		{"unauthorized", `{"ok":false,"status":401,"error":{"code":"UNAUTHORIZED","message":"sentinel-secret"}}`, "acceso rechazado", 200},
		{"duplicate", `{"ok":false,"status":409}`, "ya existe", 200},
		{"html", `<!doctype html><html>Google login</html>`, "HTML", 200},
		{"invalid", `not json`, "JSON inválida", 200},
		{"http", `failure`, "HTTP 502", 502},
		{"missing status", `{"ok":true,"data":{}}`, "API", 200},
		{"unexpected", `{"ok":true,"status":201,"data":{}}`, "inesperado", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := testAPI(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.http)
				_, _ = io.WriteString(w, tc.body)
			})
			_, e := a.call("get", map[string]interface{}{})
			if e == nil || !strings.Contains(e.Error(), tc.kind) || strings.Contains(e.Error(), "sentinel-secret") {
				t.Fatalf("unsafe or wrong error: %v", e)
			}
		})
	}
}
func TestWriteUncertainNoRetry(t *testing.T) {
	calls := 0
	a := testAPI(t, func(w http.ResponseWriter, r *http.Request) { calls++; _, _ = io.WriteString(w, `broken`) })
	_, e := a.call("create", map[string]interface{}{})
	if e == nil || !strings.Contains(e.Error(), "incierto") || calls != 1 {
		t.Fatalf("%v / %d", e, calls)
	}
}
func TestRedirectPolicy(t *testing.T) {
	for _, tc := range []struct {
		target, method string
		ok             bool
	}{
		{"https://script.googleusercontent.com/macros/echo?user_content_key=abc", "GET", true},
		{"https://evil.com/macros/echo", "GET", false},
		{"https://script.googleusercontent.com.evil.com/macros/echo", "GET", false},
		{"http://script.googleusercontent.com/macros/echo", "GET", false},
		{"https://script.googleusercontent.com/macros/echo", "POST", false},
		{"https://script.googleusercontent.com:443/macros/echo", "GET", false},
		{"https://script.googleusercontent.com/other", "GET", false},
	} {
		u, _ := url.Parse(tc.target)
		e := redirect(&http.Request{URL: u, Method: tc.method}, nil)
		if (e == nil) != tc.ok {
			t.Errorf("%s %s: %v", tc.method, tc.target, e)
		}
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestRedirect302303DropsBody(t *testing.T) {
	for _, code := range []int{302, 303} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			calls := 0
			a := newAPI("https://script.google.com/macros/s/test/exec", "sentinel-secret")
			a.http.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return &http.Response{StatusCode: code, Header: http.Header{"Location": []string{"https://script.googleusercontent.com/macros/echo?user_content_key=abc"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
				}
				if r.Method != "GET" || r.Body != nil {
					t.Error("credential body forwarded")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"ok":true,"status":200,"data":{"id":"x"}}`)), Request: r}, nil
			})
			if _, e := a.call("get", map[string]interface{}{"id": "x"}); e != nil {
				t.Fatal(e)
			}
			if calls != 2 {
				t.Fatal(calls)
			}
		})
	}
}
func TestValidation(t *testing.T) {
	for _, d := range []map[string]string{{"nombre": ""}, {"email": "bad"}, {"estado": "other"}, {"id": "readonly"}, {"notas": strings.Repeat("x", 5001)}, {"empresa": strings.Repeat("á", 501)}} {
		if validate(d, false) == nil {
			t.Errorf("accepted %v", d)
		}
	}
	if e := validate(map[string]string{"nombre": "Ana", "telefono": "+00012", "email": "", "notas": strings.Repeat("á", 5000)}, true); e != nil {
		t.Fatal(e)
	}
}
func TestPartialUpdate(t *testing.T) {
	a := testAPI(t, func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Data map[string]string `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		if len(b.Data) != 2 || b.Data["email"] != "" || b.Data["estado"] != "cliente" {
			t.Error(b)
		}
		_, _ = io.WriteString(w, `{"ok":true,"status":200,"data":{"id":"x"}}`)
	})
	s := &Service{api: a}
	if _, e := s.Write("update", "x", map[string]string{"email": "", "estado": "cliente"}); e != nil {
		t.Fatal(e)
	}
}
func TestListAndDelete(t *testing.T) {
	a := testAPI(t, func(w http.ResponseWriter, r *http.Request) {
		var b map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b["action"] == "list" {
			if b["q"] != "Ana" || b["estado"] != "nuevo" || b["offset"] != float64(50) || b["limit"] != float64(50) {
				t.Error(b)
			}
			_, _ = io.WriteString(w, `{"ok":true,"status":200,"data":[{"id":"x"}],"meta":{"total":51,"limit":50,"offset":50}}`)
		} else {
			_, _ = io.WriteString(w, `{"ok":true,"status":200,"data":{"id":"x","eliminado":true}}`)
		}
	})
	s := &Service{api: a}
	p, e := s.List("Ana", "nuevo", 50)
	if e != nil || p.Meta.Total != 51 || len(p.Data) != 1 {
		t.Fatalf("%v %v", p, e)
	}
	if e = s.Delete("x"); e != nil {
		t.Fatal(e)
	}
}
func TestPublicConfigurationNoSecret(t *testing.T) {
	s := &Service{api: newAPI("https://script.google.com/macros/s/x/exec", "sentinel-secret")}
	b, _ := json.Marshal(s.Configuration())
	if strings.Contains(string(b), "sentinel-secret") || strings.Contains(string(b), "cipher") || strings.Contains(string(b), "key") {
		t.Fatal(string(b))
	}
}
func TestURL(t *testing.T) {
	if ValidateURL("https://script.google.com/macros/s/abc_123/exec") != nil {
		t.Fatal("valid rejected")
	}
	for _, u := range []string{"http://script.google.com/macros/s/a/exec", "https://evil.com/macros/s/a/exec", "https://script.google.com/macros/s/a/exec?api_key=oops", "https://user@script.google.com/macros/s/a/exec"} {
		if ValidateURL(u) == nil {
			t.Fatal(u)
		}
	}
}
func TestAtomicSave(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nested", "connection.json")
	for _, s := range []string{"one", "two"} {
		if e := atomicSave(p, []byte(s)); e != nil {
			t.Fatal(e)
		}
		b, _ := os.ReadFile(p)
		if string(b) != s {
			t.Fatal(string(b))
		}
	}
	files, _ := os.ReadDir(filepath.Dir(p))
	if len(files) != 1 {
		t.Fatal(files)
	}
}
