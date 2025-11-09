package envseed

import (
	"context"
	"strings"
)

type secretCache struct {
	client PassClient
	cache  map[string]secretEntry
}

type secretEntry struct {
	value string
}

func newSecretCache(client PassClient) *secretCache {
	return &secretCache{
		client: client,
		cache:  make(map[string]secretEntry),
	}
}

func (c *secretCache) get(ctx context.Context, path string) (secretEntry, error) {
	if entry, ok := c.cache[path]; ok {
		return entry, nil
	}
	raw, err := c.client.Show(ctx, path)
	if err != nil {
		return secretEntry{}, err
	}
	if strings.IndexByte(raw, 0) >= 0 {
		return secretEntry{}, NewExitError("EVE-104-301", path)
	}
	entry := secretEntry{value: raw}
	c.cache[path] = entry
	return entry, nil
}

func (c *secretCache) clear() {
	for k, entry := range c.cache {
		if len(entry.value) > 0 {
			entry.value = ""
			c.cache[k] = entry
		}
		delete(c.cache, k)
	}
	c.cache = nil
}
