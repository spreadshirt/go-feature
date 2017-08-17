package feature

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBooleanFlag(t *testing.T) {
	f := NewBooleanFlag("test")

	if f.IsEnabled() {
		t.Fatal("should be initially disabled")
	}

	f.Set(true)
	if !f.IsEnabled() {
		t.Fatal("should be enabled after Set(true)")
	}

	for i := 0; i < 10; i++ {
		if !f.IsEnabled() {
			t.Fatal("should be possible to read multiple times")
		}
	}

	f.Set(false)
	if f.IsEnabled() {
		t.Fatal("should be disabled after Set(true)")
	}
}

func TestRatioFlag(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	f := NewRatioFlag("test", 1.0)
	for i := 0; i < 100; i++ {
		if f.IsEnabled() {
			t.Fatal("should be initially disabled")
		}
	}

	f.Set(true)
	for i := 0; i < 100; i++ {
		if !f.IsEnabled() {
			t.Fatal("should always be enabled")
		}
	}

	f = NewRatioFlag("test", 0.5)
	f.Set(true)
	c := 0
	for i := 0; i < 100; i++ {
		if f.IsEnabled() {
			c += 1
		}
	}
	if c < 40 || c > 60 {
		t.Fatalf("should be enabled for about 50%% of invocations, but was for %d/100", c)
	}

	f = NewRatioFlag("test", 0.333)
	f.Set(true)
	c = 0
	for i := 0; i < 100; i++ {
		if f.IsEnabled() {
			c += 1
		}
	}
	if c < 20 || c > 50 {
		t.Fatalf("should be enabled for about 33%% of invocations, but was for %d/100", c)
	}
}

func TestSet(t *testing.T) {
	s := NewSet()

	f := s.Get("test")
	if f != nil {
		t.Fatal("should be initially empty")
	}

	s.Add(NewBooleanFlag("test-Add()"))
	f = s.Get("test-Add()")
	if f == nil {
		t.Fatal("should contain 'test-Add()' flag")
	}

	s.NewFlag("test-NewFlag()")
	f = s.Get("test-NewFlag()")
	if f == nil {
		t.Fatal("should contain 'test-NewFlag()' flag")
	}

	err := s.Add(NewBooleanFlag("test-Add()"))
	if err == nil {
		t.Fatal("should fail when Add()ing a flag with an existing name")
	}

	_, err = s.NewFlag("test-NewFlag()")
	if err == nil {
		t.Fatal("should fail when NewFlag() is called with an existing name")
	}
}

func TestHTTPBooleanFlag(t *testing.T) {
	s := NewSet()
	f := NewBooleanFlag("test")
	err := s.Add(f)
	if err != nil {
		t.Fatal("should be able to add the flag")
	}

	tc := []struct {
		req     *http.Request
		code    int
		body    string
		enabled bool
	}{
		{httptest.NewRequest("GET", "/test", nil), 200, "test: false", false},
		{httptest.NewRequest("POST", "/test", nil), 400, "", false},
		{httptest.NewRequest("POST", "/test?enabled", nil), 400, "", false},
		{httptest.NewRequest("POST", "/test?enabled=", nil), 400, "", false},
		{httptest.NewRequest("POST", "/test?enabled=tru", nil), 400, "", false},
		{httptest.NewRequest("POST", "/test?enabled=fal", nil), 400, "", false},
		{httptest.NewRequest("POST", "/test?enabled=whatever", nil), 400, "", false},
		{httptest.NewRequest("POST", "/test?enabled=true", nil), 200, "", true},
		{httptest.NewRequest("GET", "/test", nil), 200, "test: true", true},
		{httptest.NewRequest("POST", "/test?enabled=", nil), 400, "", true},
		{httptest.NewRequest("POST", "/test?enabled=fal", nil), 400, "", true},
		{httptest.NewRequest("POST", "/test?enabled=false", nil), 200, "", false},
		{httptest.NewRequest("GET", "/test", nil), 200, "test: false", false},
	}

	for _, c := range tc {
		t.Run(fmt.Sprintf("%s %s", c.req.Method, c.req.URL.Path), func(t *testing.T) {
			rec := httptest.NewRecorder()

			s.ServeHTTP(rec, c.req)
			if rec.Code != c.code {
				t.Fatalf("should respond with %d, but was %d", c.code, rec.Code)
			}

			if !strings.Contains(rec.Body.String(), c.body) {
				t.Fatalf("should contain %q, but was %q", c.body, rec.Body.String())
			}

			if f.IsEnabled() != c.enabled {
				t.Fatalf("should be %v but was %v", c.enabled, f.IsEnabled())
			}
		})
	}
}

func postFormRequest(url string, vals url.Values) *http.Request {
	req, err := http.NewRequest("POST", url, strings.NewReader(vals.Encode()))
	if err != nil {
		panic(fmt.Sprintf("http.NewRequest: %s", err))
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func BenchmarkBooleanFlag(b *testing.B) {
	f := NewBooleanFlag("test")

	go func() {
		for {
			n := rand.Intn(100)
			time.Sleep(time.Duration(n) * time.Nanosecond)
			f.Set(n < 50)
		}
	}()

	b.RunParallel(func(pb *testing.PB) {
		c := 0
		for pb.Next() {
			if f.IsEnabled() {
				c += 1
			}
		}
		b.Logf("IsEnabled() %d times", c)
	})
}

func BenchmarkRatioFlag(b *testing.B) {
	f := NewRatioFlag("test", 0.5)

	go func() {
		for {
			n := rand.Intn(100)
			time.Sleep(time.Duration(n) * time.Nanosecond)
			f.Set(n < 50)
		}
	}()

	b.RunParallel(func(pb *testing.PB) {
		c := 0
		for pb.Next() {
			if f.IsEnabled() {
				c += 1
			}
		}
		b.Logf("IsEnabled() %d times", c)
	})
}
