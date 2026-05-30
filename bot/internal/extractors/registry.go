package extractors

// Registry routes URLs to the appropriate Extractor by calling CanHandle
// in registration order; the first match wins.
type Registry struct {
	extractors []Extractor
}

func NewRegistry(extractors ...Extractor) *Registry {
	return &Registry{extractors: extractors}
}

// For returns the first Extractor that can handle the given URL.
func (r *Registry) For(url string) (Extractor, bool) {
	for _, e := range r.extractors {
		if e.CanHandle(url) {
			return e, true
		}
	}
	return nil, false
}
