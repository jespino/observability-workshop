package main

// A small in-process cache in front of the list read path.
//
// No network, no extra container: just a map with time-to-live entries.

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

const cacheTTL = 30 * time.Second

type cacheEntry struct {
	value   []Todo
	expires time.Time
}

var (
	cacheMu sync.Mutex
	store   = map[string]cacheEntry{}
)

func cacheKey(list string) string {
	return "todos:summary:" + list + ":" + strconv.FormatInt(time.Now().UnixMilli(), 10)
}

func readSummary(list string) (hit bool, key string, value []Todo) {
	key = cacheKey(list)
	cacheMu.Lock()
	entry, ok := store[key]
	cacheMu.Unlock()
	if ok && entry.expires.After(time.Now()) {
		return true, key, entry.value
	}
	return false, key, nil
}

func writeSummary(list string, value []Todo) string {
	key := cacheKey(list)
	cacheMu.Lock()
	store[key] = cacheEntry{value: value, expires: time.Now().Add(cacheTTL)}
	cacheMu.Unlock()
	return key
}

func invalidate(list string) {
	prefix := "todos:summary:" + list + ":"
	cacheMu.Lock()
	for key := range store {
		if strings.HasPrefix(key, prefix) {
			delete(store, key)
		}
	}
	cacheMu.Unlock()
}
