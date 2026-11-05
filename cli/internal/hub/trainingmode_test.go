package hub

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPinnedModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			w.WriteHeader(401)
			return
		}
		// Shape of llama-swap's GET /api/profiles (zaphod, 2026-11-05).
		w.Write([]byte(`{"active":null,"profiles":[{"id":"training","description":"x",
			"pins":{"flint":"","coder":"","chat-teacher":""}},{"id":"image-mode","pins":{"coder-2":"coder"}}]}`))
	}))
	defer srv.Close()
	l := LS{"test", srv.URL, "k"}
	got, err := l.PinnedModels("training")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"chat-teacher", "coder", "flint"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pins %v, want %v", got, want)
	}
	if _, err := l.PinnedModels("missing"); err == nil {
		t.Fatal("missing profile: want an error")
	}
	if a, _ := l.ActiveProfile(); a != "none" {
		t.Fatalf("active %q, want none", a)
	}
}

func TestMemFree(t *testing.T) {
	cases := []struct {
		cards []int64
		avail int64
		want  bool
	}{
		{[]int64{3000, 8192}, 102400, true},
		{[]int64{3000, 8193}, 102400, false}, // card 2 over 8 GB
		{[]int64{3000, 3000}, 102399, false}, // marvin under 100 GB
		{[]int64{-1, 3000}, 110000, false},   // card unreadable
		{[]int64{3000, 3000}, -1, false},     // marvin unreachable
	}
	for _, c := range cases {
		if got := memFree(c.cards, c.avail); got != c.want {
			t.Errorf("memFree(%v, %d) = %v, want %v", c.cards, c.avail, got, c.want)
		}
	}
}
