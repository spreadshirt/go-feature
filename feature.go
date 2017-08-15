// Package feature provides a simple mechanism for creating and managing
// feature flags.
//
// This allows code to be dynamically enabled and disabled in
// production, without having to restart or redeploy services.
//
// All functions are concurrency-safe, i.e. can be used from multiple
// goroutines concurrently.
package feature

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Set collects a named set of feature flags.
type Set struct {
	mu    sync.Mutex
	flags map[string]Flag
}

// NewSet returns a new set of feature flags.
func NewSet() *Set {
	return &Set{
		flags: make(map[string]Flag),
	}
}

// Add adds a feature flag to the set.
func (fs *Set) Add(f Flag) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, ok := fs.flags[f.Name()]; ok {
		return fmt.Errorf("duplicate feature %q", f.Name())
	}

	fs.flags[f.Name()] = f
	return nil
}

// NewFlag creates a new feature flag with the given name and adds it to
// the feature set.
//
// Returns an error (and does not add the flag) if a flag with
// that name is already contained in the set.
func (fs *Set) NewFlag(name string) (*BooleanFlag, error) {
	f := NewBooleanFlag(name)
	err := fs.Add(f)
	return f, err
}

// Get returns the flag with the given name from the feature set.
//
// Returns nil if no such flag exists.
func (fs *Set) Get(name string) Flag {
	fs.mu.Lock()
	f := fs.flags[name]
	fs.mu.Unlock()
	return f
}

// ServeHTTP implements an admin interface for the feature set.
//
// The root handler lists all features, and POST
// /feature-name?enabled=true enables the given feature flag.
func (fs *Set) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	lastSlash := strings.LastIndex(req.URL.Path, "/")

	if lastSlash == -1 || lastSlash == len(req.URL.Path)-1 {
		fs.handleIndex(w, req)
		return
	}

	name := req.URL.Path[lastSlash+1:]
	fs.handleFlag(w, req, name)
}

type flagsByName []Flag

func (f flagsByName) Len() int           { return len(f) }
func (f flagsByName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f flagsByName) Less(i, j int) bool { return f[i].Name() < f[j].Name() }

func (fs *Set) handleIndex(w http.ResponseWriter, req *http.Request) {
	fs.mu.Lock()
	flags := make([]Flag, 0, len(fs.flags))
	for _, f := range fs.flags {
		flags = append(flags, f)
	}
	fs.mu.Unlock()

	sort.Sort(flagsByName(flags))

	fmt.Fprintf(w, "Flags:\n\n")
	for _, f := range flags {
		fmt.Fprintf(w, "%s: %v\n", f.Name(), f.IsEnabled())
	}
}

func (fs *Set) handleFlag(w http.ResponseWriter, req *http.Request, name string) {
	flag := fs.Get(name)

	if flag == nil {
		http.Error(w, "no such feature", http.StatusNotFound)
		return
	}

	switch req.Method {
	case "GET":
		fmt.Fprintf(w, "%s: %v", name, flag.IsEnabled())
	case "POST":
		enabledRaw := req.URL.Query().Get("enabled")
		if strings.TrimSpace(enabledRaw) == "" {
			http.Error(w, "missing 'enabled' parameter", http.StatusBadRequest)
			return
		}

		enabled, err := strconv.ParseBool(enabledRaw)
		if err != nil {
			log.Printf("Error: parsing bool param %q: %s\n", enabledRaw, err)
			http.Error(w, "invalid parameter", http.StatusBadRequest)
			return
		}

		flag.Set(enabled)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

// Flag is an enabled or disabled feature flag.
type Flag interface {
	// Name returns the name of the flag.
	Name() string

	// IsEnabled returns true if the flag is enabled.
	IsEnabled() bool

	// Set enables or disables the flag.
	Set(bool)
}

// BooleanFlag is a feature flag that can be switched on or off.
type BooleanFlag struct {
	name string

	mu      sync.Mutex
	enabled bool
}

// NewBooleanFlag returns a new boolean feature flag.
func NewBooleanFlag(name string) *BooleanFlag {
	f := &BooleanFlag{
		name: name,

		enabled: false,
	}
	return f
}

// Name returns the name of the flag.
func (f *BooleanFlag) Name() string {
	return f.name
}

// IsEnabled returns true if the flag is enabled.
func (f *BooleanFlag) IsEnabled() bool {
	f.mu.Lock()
	isEnabled := f.enabled
	f.mu.Unlock()
	return isEnabled
}

// Set enables or disables the flag.
func (f *BooleanFlag) Set(enabled bool) {
	f.mu.Lock()
	f.enabled = enabled
	f.mu.Unlock()
}
