package feature

import (
	"math/rand"
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
