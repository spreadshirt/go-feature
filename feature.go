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
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Set collects a named set of feature flags.
type Set struct {
	mu    sync.RWMutex
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
	fs.mu.RLock()
	f := fs.flags[name]
	fs.mu.RUnlock()
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
	fs.mu.RLock()
	flags := make([]Flag, 0, len(fs.flags))
	for _, f := range fs.flags {
		flags = append(flags, f)
	}
	fs.mu.RUnlock()

	sort.Sort(flagsByName(flags))

	if strings.Contains(req.Header.Get("Accept"), "html") {
		fmt.Fprintf(w, `<!doctype html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>Flags</title>
	</head>

	<body>
		<ul>
`)
		for _, f := range flags {
			fmt.Fprintf(w, "<li>")

			switch f := f.(type) {
			case RenderableFlag:
				f.RenderHTML(w)
			default:
				fmt.Fprintf(w, "<pre>%s: %v</pre>\n", f.Name(), f.IsEnabled())
			}

			fmt.Fprintf(w, "\n</li>\n\n")
		}

		fmt.Fprintf(w, `
		</ul>
	</body>
</html>`)
	} else {
		fmt.Fprintf(w, "Flags:\n\n")
		for _, f := range flags {
			switch f := f.(type) {
			case fmt.Stringer:
				fmt.Fprintf(w, "%s\n", f.String())
			default:
				fmt.Fprintf(w, "%s: %v\n", f.Name(), f.IsEnabled())
			}
		}
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
		switch flag := flag.(type) {
		case fmt.Stringer:
			fmt.Fprintf(w, "%s", flag.String())
		default:
			fmt.Fprintf(w, "%s: %v", name, flag.IsEnabled())
		}
	case "POST":
		err := req.ParseForm()
		if err != nil {
			http.Error(w, "invalid parameters", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.Form.Get("enabled")) == "" {
			if req.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
				req.Form.Set("enabled", "false")
				req.PostForm.Set("enabled", "false")
			}
		}

		switch flag := flag.(type) {
		case SettableFlag:
			err := flag.SetFrom(req.Form)
			if err != nil {
				log.Printf("Error: parsing parameters: %s\n", err)
				http.Error(w, "invalid parameters", http.StatusBadRequest)
			}
		default:
			enabledRaw := req.Form.Get("enabled")
			enabled, err := strconv.ParseBool(enabledRaw)
			if err != nil {
				log.Printf("Error: parsing bool param %q: %s\n", enabledRaw, err)
				http.Error(w, "invalid parameter", http.StatusBadRequest)
				return
			}

			flag.Set(enabled)
		}

		referer := req.Header.Get("Referer")
		if referer != "" {
			w.Header().Set("Location", referer)
			w.WriteHeader(http.StatusTemporaryRedirect)
		}

		switch flag := flag.(type) {
		case fmt.Stringer:
			fmt.Fprintf(w, "%s", flag.String())
		default:
			fmt.Fprintf(w, "%s: %v", name, flag.IsEnabled())
		}
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

	mu      sync.RWMutex
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
	f.mu.RLock()
	isEnabled := f.enabled
	f.mu.RUnlock()
	return isEnabled
}

// Set enables or disables the flag.
func (f *BooleanFlag) Set(enabled bool) {
	f.mu.Lock()
	f.enabled = enabled
	f.mu.Unlock()
}

type SettableFlag interface {
	SetFrom(url.Values) error
}

func (f *BooleanFlag) SetFrom(vals url.Values) error {
	enabledRaw := vals.Get("enabled")
	if strings.TrimSpace(enabledRaw) == "" {
		return fmt.Errorf("missing 'enabled' parameter")
	}

	enabled, err := strconv.ParseBool(enabledRaw)
	if err != nil {
		return fmt.Errorf("invalid parameter")
	}

	f.Set(enabled)
	return nil
}

func (f *BooleanFlag) RenderHTML(w http.ResponseWriter) {
	f.mu.RLock()
	enabled := f.enabled
	f.mu.RUnlock()

	err := booleanTmpl.Execute(w, map[string]interface{}{
		"name":    f.Name(),
		"enabled": enabled,
	})
	if err != nil {
		log.Println(err)
	}
}

var booleanTmpl = template.Must(template.New("").Parse(`
<form id="feature-{{ .name }}" class="feature" method="POST" action="./{{ .name }}" autocomplete="off">
	{{ .name }}
	<input name="enabled" type="checkbox" value="true" {{ if .enabled }}checked{{ end }} />

	<input type="submit" value="Apply!" />
</form>`))

// RatioFlag is a feature flag that is only activated for the given ratio
// of invocations.
//
// This allows enabling a feature for only a subset of invocations, for
// example to test a new feature on a smaller scale.
//
// Be aware that you should seed the random number generator from
// math/rand before using this.  This can be done by calling
// `rand.Seed(...)`.
type RatioFlag struct {
	BooleanFlag

	ratio float64
}

// NewRatioFlag returns a new flag that will only activate for the given
// ratio of invocations.
//
// The given ratio must be in the interval [0.0, 1.0).
//
// Note that the flag must also be enabled (via Set()) separately.
func NewRatioFlag(name string, ratio float64) *RatioFlag {
	return &RatioFlag{
		BooleanFlag: BooleanFlag{
			name: name,

			enabled: false,
		},
		ratio: ratio,
	}
}

// IsEnabled returns true if the flag is enabled.
//
// Even when the flag is enabled, only `ratio` of invocations will
// return true.  That is, the flag has to be enabled and only every nth
// invocation (according to ratio) will return true.
func (f *RatioFlag) IsEnabled() bool {
	f.mu.RLock()
	isEnabled := f.enabled
	ratio := f.ratio
	f.mu.RUnlock()

	return isEnabled && rand.Float64() < ratio
}

// SetRatio sets the ratio with which the flag should be activated.
func (f *RatioFlag) SetRatio(r float64) {
	f.mu.Lock()
	f.ratio = r
	f.mu.Unlock()
}

func (f *RatioFlag) String() string {
	f.mu.RLock()
	enabled := f.enabled
	ratio := f.ratio
	f.mu.RUnlock()
	return fmt.Sprintf("%s: %v (ratio=%.2f)", f.name, enabled, ratio)
}

func (f *RatioFlag) SetFrom(vals url.Values) error {
	enabledRaw := vals.Get("enabled")
	if strings.TrimSpace(enabledRaw) != "" {
		enabled, err := strconv.ParseBool(enabledRaw)
		if err != nil {
			return fmt.Errorf("invalid parameter %q", enabledRaw)
		}

		f.Set(enabled)
	}

	ratioRaw := vals.Get("ratio")
	if strings.TrimSpace(ratioRaw) != "" {
		ratio, err := strconv.ParseFloat(ratioRaw, 64)
		if err != nil {
			return fmt.Errorf("invalid parameter %q", ratioRaw)
		}

		f.SetRatio(ratio)
	}

	return nil
}

type RenderableFlag interface {
	RenderHTML(w http.ResponseWriter)
}

func (f *RatioFlag) RenderHTML(w http.ResponseWriter) {
	f.mu.RLock()
	enabled := f.enabled
	ratio := f.ratio
	f.mu.RUnlock()

	err := ratioTmpl.Execute(w, map[string]interface{}{
		"name":         f.Name(),
		"enabled":      enabled,
		"ratio":        ratio,
		"ratioPercent": fmt.Sprintf("%.0f%%", ratio*100),
	})
	if err != nil {
		log.Println(err)
	}
}

var ratioTmpl = template.Must(template.New("").Parse(`
<form id="feature-{{ .name }}" class="feature" method="POST" action="./{{ .name }}" autocomplete="off" oninput="ratio_value.value = Math.round(parseFloat(ratio.value)*100) + '%'">
	{{ .name }}
	<input name="enabled" type="checkbox" value="true"{{ if .enabled }} checked{{ end }} />
	<input name="ratio" type="range" min="0.0" max="1.0" step="0.01" value="{{ .ratio }}" />
	<output name="ratio_value">{{ .ratioPercent }}</output>

	<input type="submit" value="Apply!" />
</form>`))
