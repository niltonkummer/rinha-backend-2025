package cache

import (
	"sync"
)

type cacheImpl struct {
	cache []any
}

type CacheService struct {
	cache []any
	sync.Mutex
}

func NewCacheService(size int) *CacheService {
	return &CacheService{
		cache: make([]any, 0, size),
	}
}

func (cs *CacheService) Add(value any) {
	cs.Lock()
	defer cs.Unlock()
	cs.cache = append(cs.cache, value)
}

func (cs *CacheService) GetList() ([]any, error) {
	cs.Lock()
	defer cs.Unlock()
	copyCache := make([]any, len(cs.cache))
	copy(copyCache, cs.cache)
	return copyCache, nil
}
