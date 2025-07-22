package cache

type CacheManager struct {
	cache CacheService
	queue Queue
}

func NewCacheManager(cache CacheService, queue Queue) *CacheManager {
	return &CacheManager{
		cache: cache,
		queue: queue,
	}
}
