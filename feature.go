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

type FeatureSet struct {
	mu       sync.Mutex
	features map[string]*Feature
}

func NewFeatureSet() *FeatureSet {
	return &FeatureSet{
		features: make(map[string]*Feature),
	}
}

func (fs *FeatureSet) Add(f *Feature) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, ok := fs.features[f.Name()]; ok {
		return fmt.Errorf("duplicate feature %q", f.Name())
	}

	fs.features[f.Name()] = f
	return nil
}

func (fs *FeatureSet) NewFeature(name string) (*Feature, error) {
	f := &Feature{
		name: name,

		enabled: false,
	}
	err := fs.Add(f)
	return f, err
}

func (fs *FeatureSet) Get(name string) *Feature {
	fs.mu.Lock()
	f := fs.features[name]
	fs.mu.Unlock()
	return f
}

func (fs *FeatureSet) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	lastSlash := strings.LastIndex(req.URL.Path, "/")

	// index
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

func (fs *FeatureSet) handleIndex(w http.ResponseWriter, req *http.Request) {
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

func (fs *FeatureSet) handleFeature(w http.ResponseWriter, req *http.Request, name string) {
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

type Feature struct {
	name string

	mu      sync.Mutex
	enabled bool
}

func NewFeature(name string) *Feature {
	f := &Feature{
		name: name,

		enabled: false,
	}
	return f
}

func (f *Feature) Name() string {
	return f.name
}

func (f *Feature) IsEnabled() bool {
	f.mu.Lock()
	isEnabled := f.enabled
	f.mu.Unlock()
	return isEnabled
}

func (f *Feature) Set(enabled bool) {
	f.mu.Lock()
	f.enabled = enabled
	f.mu.Unlock()
}
