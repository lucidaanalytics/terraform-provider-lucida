package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const (
	testKey     = "lc_test_key"
	testProject = "p1"
)

// fakeAPI is a minimal Lucida: generic collections under /v1/projects/p1/<route> with cursor
// pagination (two rows per page), problem+json errors and server-side defaults.
type fakeAPI struct {
	mu    sync.Mutex
	rows  map[string][]map[string]any
	seq   int
	calls []string
	fail  map[string]int // "METHOD route" -> status to answer once
}

func newFake(t *testing.T) (*fakeAPI, *Client) {
	t.Helper()
	f := &fakeAPI{rows: map[string][]map[string]any{}, fail: map[string]int{}}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, &Client{Host: srv.URL, APIKey: testKey, ProjectID: testProject, Backoff: 1}
}

func (f *fakeAPI) problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"title": http.StatusText(status), "status": status, "detail": detail})
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer "+testKey {
		f.problem(w, 401, "bad key")
		return
	}
	if r.URL.Path == "/v1/projects/"+testProject {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": testProject, "name": "Shop"})
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, "/v1/projects/"+testProject+"/")
	if !ok {
		f.problem(w, 404, "no such route")
		return
	}
	rest = strings.Replace(strings.Replace(rest, "web/goals", "web-goals", 1), "web/revenue/config", "revenue/config", 1)
	route, id, _ := strings.Cut(rest, "/")
	f.calls = append(f.calls, r.Method+" "+rest)
	if st := f.fail[r.Method+" "+route]; st != 0 {
		delete(f.fail, r.Method+" "+route)
		f.problem(w, st, "injected")
		return
	}
	var body map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	switch {
	case r.Method == "GET" && id == "":
		f.list(w, r, route)
	case r.Method == "POST" && id == "":
		f.seq++
		row := map[string]any{"id": fmt.Sprintf("id-%d", f.seq), "created_at": "2026-01-01T00:00:00Z", "tags": []any{}}
		for k, v := range body {
			row[k] = v
		}
		f.rows[route] = append(f.rows[route], row)
		out := map[string]any{}
		for k, v := range row {
			out[k] = v
		}
		if route == "keys" {
			out["secret"] = fmt.Sprintf("lcsk_secret_%d", f.seq)
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(out)
	default:
		f.item(w, r, route, id, body)
	}
}

func (f *fakeAPI) list(w http.ResponseWriter, r *http.Request, route string) {
	start, _ := strconv.Atoi(r.URL.Query().Get("cursor"))
	all := f.rows[route]
	end := min(start+2, len(all))
	next := ""
	if end < len(all) {
		next = strconv.Itoa(end)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": all[start:end], "next_cursor": next})
}

func (f *fakeAPI) item(w http.ResponseWriter, r *http.Request, route, id string, body map[string]any) {
	for i, row := range f.rows[route] {
		if row["id"] != id {
			continue
		}
		switch r.Method {
		case "PATCH":
			for k, v := range body {
				row[k] = v
			}
		case "PUT":
			f.rows[route][i] = body
			body["id"] = id
		case "DELETE":
			if route == "keys" {
				row["revoked_at"] = "2026-01-02T00:00:00Z"
			} else {
				f.rows[route] = append(f.rows[route][:i], f.rows[route][i+1:]...)
			}
			w.WriteHeader(204)
			return
		}
		_ = json.NewEncoder(w).Encode(row)
		return
	}
	f.problem(w, 404, "not found")
}

// row returns the stored row with the given natural key.
func (f *fakeAPI) row(route, field, val string) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows[route] {
		if r[field] == val {
			return r
		}
	}
	return nil
}

func (f *fakeAPI) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}
