package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestIsBrowserRequest exercises the gate that controls whether an
// auth-failure 302→login fires. The post-fix rule: gate on Accept:
// text/html only. The pre-fix code also fell back to a User-Agent
// substring match ("Mozilla"), which misclassified browser-issued
// fetch()/XHR JSON calls as page navigations.
func TestIsBrowserRequest(t *testing.T) {
	cases := []struct {
		name      string
		accept    string
		userAgent string
		want      bool
	}{
		{
			name:      "top-level browser nav",
			accept:    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			want:      true,
		},
		{
			name:      "html-only accept, no UA",
			accept:    "text/html",
			userAgent: "",
			want:      true,
		},
		{
			name:      "SPA fetch (json accept, Mozilla UA)",
			accept:    "application/json",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64)",
			want:      false, // critical: was true pre-fix → spurious 302
		},
		{
			name:      "SPA fetch with */* accept",
			accept:    "*/*",
			userAgent: "Mozilla/5.0",
			want:      false,
		},
		{
			name:      "curl default",
			accept:    "*/*",
			userAgent: "curl/8.0.1",
			want:      false,
		},
		{
			name:      "curl impersonating browser UA",
			accept:    "*/*",
			userAgent: "Mozilla/5.0 (compatible; curl)",
			want:      false, // pre-fix: true; post-fix: UA fallback removed
		},
		{
			name:      "no headers at all",
			accept:    "",
			userAgent: "",
			want:      false,
		},
		{
			name:      "Accept includes both html and json (browser-preferred html)",
			accept:    "application/json, text/html;q=0.9",
			userAgent: "Mozilla/5.0",
			want:      true,
		},
		{
			name:      "multi-valued Accept header (HTTP spec allows split values)",
			accept:    "", // we set two Accept values manually below
			userAgent: "",
			want:      true, // exercised by the dedicated multi-value subtest below
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/x", nil)
			if tc.name == "multi-valued Accept header (HTTP spec allows split values)" {
				r.Header.Add("Accept", "application/json")
				r.Header.Add("Accept", "text/html")
			} else {
				if tc.accept != "" {
					r.Header.Set("Accept", tc.accept)
				}
				if tc.userAgent != "" {
					r.Header.Set("User-Agent", tc.userAgent)
				}
			}
			got := IsBrowserRequest(r)
			if got != tc.want {
				t.Errorf("IsBrowserRequest(Accept=%q UA=%q) = %v, want %v",
					r.Header.Get("Accept"), r.Header.Get("User-Agent"), got, tc.want)
			}
		})
	}
}

// TestHandleAuthFailureRedirectsBrowserGet asserts a top-level browser
// navigation to a protected page (GET + Accept: text/html) with
// redirect_on_failure configured gets a 302 to the login URL.
func TestHandleAuthFailureRedirectsBrowserGet(t *testing.T) {
	am, err := NewAuthManager(map[string]*AuthConfig{
		"primary": {Type: "default", RedirectOnFailure: "/login"},
	}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/protected", nil)
	r.Header.Set("Accept", "text/html,*/*;q=0.8")
	r.Header.Set("User-Agent", "Mozilla/5.0")
	w := httptest.NewRecorder()

	am.handleAuthFailure(w, r, am.configs["primary"], errors.New("no session"))

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
}

// TestHandleAuthFailureJSONForSPAFetch is the regression case for the
// observed bug: an SPA fetch() to a protected JSON endpoint should
// receive a JSON 401, never a 302. Before the IsBrowserRequest fix,
// the UA-based detection misclassified this as a browser nav.
func TestHandleAuthFailureJSONForSPAFetch(t *testing.T) {
	am, err := NewAuthManager(map[string]*AuthConfig{
		"primary": {Type: "default", RedirectOnFailure: "/login"},
	}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			r := httptest.NewRequest(method, "/api/data", nil)
			r.Header.Set("Accept", "application/json")
			r.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")
			w := httptest.NewRecorder()

			am.handleAuthFailure(w, r, am.configs["primary"], errors.New("expired"))

			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", w.Code)
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("response body not JSON: %v (%q)", err, w.Body.String())
			}
			if body["code"] != "unauthorized" {
				t.Errorf("body.code = %v, want \"unauthorized\"", body["code"])
			}
		})
	}
}

// TestHandleAuthFailureJSONWhenNoRedirectConfigured asserts that with
// no RedirectOnFailure, every failure is JSON 401 regardless of
// request shape. This guarantees back-compat for API-only deployments.
func TestHandleAuthFailureJSONWhenNoRedirectConfigured(t *testing.T) {
	am, err := NewAuthManager(map[string]*AuthConfig{
		"primary": {Type: "default"}, // no RedirectOnFailure
	}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/protected", nil)
	r.Header.Set("Accept", "text/html")
	r.Header.Set("User-Agent", "Mozilla/5.0")
	w := httptest.NewRecorder()

	am.handleAuthFailure(w, r, am.configs["primary"], errors.New("nope"))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (no redirect configured)", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("Location header set unexpectedly: %q", loc)
	}
}

// TestHandleAuthFailureJSONForBrowserPOST asserts that even a "real
// browser" POST (Accept: text/html, but POST method) gets JSON 401,
// not a 302. POSTing to /protected and getting a 302→/login would
// strand the user's form data; JSON 401 lets the SPA handle it.
func TestHandleAuthFailureJSONForBrowserPOST(t *testing.T) {
	am, err := NewAuthManager(map[string]*AuthConfig{
		"primary": {Type: "default", RedirectOnFailure: "/login"},
	}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "/things", nil)
	r.Header.Set("Accept", "text/html")
	r.Header.Set("User-Agent", "Mozilla/5.0")
	w := httptest.NewRecorder()

	am.handleAuthFailure(w, r, am.configs["primary"], errors.New("no session"))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for browser POST", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("Location set on POST failure: %q", loc)
	}
}

// TestHandleAuthFailureJSONForCurl asserts curl-style clients get JSON
// 401 in every case — the most common API consumer path.
func TestHandleAuthFailureJSONForCurl(t *testing.T) {
	am, err := NewAuthManager(map[string]*AuthConfig{
		"primary": {Type: "default", RedirectOnFailure: "/login"},
	}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/api/items", nil)
	r.Header.Set("Accept", "*/*")
	r.Header.Set("User-Agent", "curl/8.0.1")
	w := httptest.NewRecorder()

	am.handleAuthFailure(w, r, am.configs["primary"], errors.New("no token"))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (curl)", w.Code)
	}
}

// TestHandleAuthFailureNilConfigSafe asserts the function doesn't
// panic with a nil config — defensive against startup-race or
// optional-auth scenarios.
func TestHandleAuthFailureNilConfigSafe(t *testing.T) {
	am, err := NewAuthManager(map[string]*AuthConfig{
		"primary": {Type: "default"},
	}, "test-secret")
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "/x", nil)
	w := httptest.NewRecorder()

	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("panicked on nil config: %v", rec)
		}
	}()

	am.handleAuthFailure(w, r, nil, errors.New("boom"))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
