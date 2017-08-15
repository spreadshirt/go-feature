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

// Set collects a named set of features.
type Set struct {
	mu       sync.Mutex
	features map[string]*Feature
}

// NewSet returns a new feature set.
func NewSet() *Set {
	return &Set{
		features: make(map[string]*Feature),
	}
}

// Add adds a feature to the feature set.
func (fs *Set) Add(f *Feature) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, ok := fs.features[f.Name()]; ok {
		return fmt.Errorf("duplicate feature %q", f.Name())
	}

	fs.features[f.Name()] = f
	return nil
}

// NewFeature creates a new feature with the given name and adds it to
// the feature set.
//
// Returns an error (and does not add the feature) if a feature with
// that name is already contained in the feature set.
func (fs *Set) NewFeature(name string) (*Feature, error) {
	f := &Feature{
		name: name,

		enabled: false,
	}
	err := fs.Add(f)
	return f, err
}

// Get returns the feature with the given name from the feature set.
//
// Returns nil if no such feature exists.
func (fs *Set) Get(name string) *Feature {
	fs.mu.Lock()
	f := fs.features[name]
	fs.mu.Unlock()
	return f
}

// ServeHTTP implements an admin interface for the feature flag.
//
// The root handler lists all features, and POST
// /feature-name?enabled=true enables the given feature.
func (fs *Set) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	lastSlash := strings.LastIndex(req.URL.Path, "/")

	if lastSlash == -1 || lastSlash == len(req.URL.Path)-1 {
		fs.handleIndex(w, req)
		return
	}

	name := req.URL.Path[lastSlash+1:]
	fs.handleFeature(w, req, name)
}

type featuresByName []*Feature

func (f featuresByName) Len() int           { return len(f) }
func (f featuresByName) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f featuresByName) Less(i, j int) bool { return f[i].Name() < f[j].Name() }

func (fs *Set) handleIndex(w http.ResponseWriter, req *http.Request) {
	fs.mu.Lock()
	features := make([]*Feature, 0, len(fs.features))
	for _, f := range fs.features {
		features = append(features, f)
	}
	fs.mu.Unlock()

	sort.Sort(featuresByName(features))

	fmt.Fprintf(w, "Features:\n\n")
	for _, f := range features {
		fmt.Fprintf(w, "%s: %v\n", f.Name(), f.IsEnabled())
	}
}

func (fs *Set) handleFeature(w http.ResponseWriter, req *http.Request, name string) {
	feature := fs.Get(name)

	if feature == nil {
		http.Error(w, "no such feature", http.StatusNotFound)
		return
	}

	switch req.Method {
	case "GET":
		fmt.Fprintf(w, "%s: %v", name, feature.IsEnabled())
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

		feature.Set(enabled)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

// Feature is an enabled or disabled feature flag.
type Feature struct {
	name string

	mu      sync.Mutex
	enabled bool
}

// NewFeature returns a new feature.
func NewFeature(name string) *Feature {
	f := &Feature{
		name: name,

		enabled: false,
	}
	return f
}

// Name returns the name of the feature.
func (f *Feature) Name() string {
	return f.name
}

// IsEnabled returns true if the feature is enabled.
func (f *Feature) IsEnabled() bool {
	f.mu.Lock()
	isEnabled := f.enabled
	f.mu.Unlock()
	return isEnabled
}

// Set enables or disables the feature.
func (f *Feature) Set(enabled bool) {
	f.mu.Lock()
	f.enabled = enabled
	f.mu.Unlock()
}
