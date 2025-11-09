package envseed

import (
	"context"
)

type passResolver struct {
	ctx      context.Context
	cache    *secretCache
	closed   bool
	rendered map[string][]string
}

func newPassResolver(ctx context.Context, client PassClient) *passResolver {
	return &passResolver{
		ctx:      ctx,
		cache:    newSecretCache(client),
		rendered: make(map[string][]string),
	}
}

func (r *passResolver) Resolve(path string) (string, error) {
	if r.closed {
		return "", NewExitError("EVE-199-2")
	}
	entry, err := r.cache.get(r.ctx, path)
	if err != nil {
		return "", err
	}
	return entry.value, nil
}

func (r *passResolver) RecordRendered(path, value string) {
	if r.closed {
		return
	}
	r.rendered[path] = append(r.rendered[path], value)
}

func (r *passResolver) RenderedValues() map[string][]string {
	out := make(map[string][]string, len(r.rendered))
	for path, values := range r.rendered {
		copyValues := make([]string, len(values))
		copy(copyValues, values)
		out[path] = copyValues
	}
	return out
}

func (r *passResolver) Close() {
	if r.closed {
		return
	}
	r.closed = true
	r.cache.clear()
	for k := range r.rendered {
		delete(r.rendered, k)
	}
}

func (r *passResolver) Snapshot() map[string]string {
	if r.cache == nil || r.cache.cache == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(r.cache.cache))
	for k, entry := range r.cache.cache {
		out[k] = entry.value
	}
	return out
}
