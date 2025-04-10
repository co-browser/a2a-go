// In-memory cache utility (Singleton, thread-safe)

package utils

import (
	"sync"
	"time"
)

type InMemoryCache struct {
	cacheData map[string]interface{}
	ttl       map[string]float64
	dataLock  sync.Mutex
}

var instance *InMemoryCache
var once sync.Once

func GetCacheInstance() *InMemoryCache {
	once.Do(func() {
		instance = &InMemoryCache{
			cacheData: make(map[string]interface{}),
			ttl:       make(map[string]float64),
		}
	})
	return instance
}

func (c *InMemoryCache) Set(key string, value interface{}, ttlSeconds *int) {
	c.dataLock.Lock()
	defer c.dataLock.Unlock()

	c.cacheData[key] = value
	if ttlSeconds != nil {
		c.ttl[key] = float64(time.Now().Unix()) + float64(*ttlSeconds)
	} else {
		delete(c.ttl, key)
	}
}

func (c *InMemoryCache) Get(key string, defaultValue interface{}) interface{} {
	c.dataLock.Lock()
	defer c.dataLock.Unlock()

	if expiration, exists := c.ttl[key]; exists && float64(time.Now().Unix()) > expiration {
		delete(c.cacheData, key)
		delete(c.ttl, key)
		return defaultValue
	}
	if val, ok := c.cacheData[key]; ok {
		return val
	}
	return defaultValue
}

func (c *InMemoryCache) Delete(key string) bool {
	c.dataLock.Lock()
	defer c.dataLock.Unlock()

	if _, exists := c.cacheData[key]; exists {
		delete(c.cacheData, key)
		delete(c.ttl, key)
		return true
	}
	return false
}

func (c *InMemoryCache) Clear() bool {
	c.dataLock.Lock()
	defer c.dataLock.Unlock()

	c.cacheData = make(map[string]interface{})
	c.ttl = make(map[string]float64)
	return true
}
