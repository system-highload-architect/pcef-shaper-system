package lru

import (
	"container/list"
	"context"
	"runtime" // Импортируем для ручного форсирования Garbage Collector
	"sync"
	"time"
)

type EvictCallback func(key string)

type cacheEntry struct {
	key           string
	value         any
	lastHeartbeat time.Time
}

type ReactiveLruCache struct {
	mu          sync.RWMutex
	capacity    int
	idleTimeout time.Duration
	items       map[string]*list.Element
	evictList   *list.List
	onEvict     EvictCallback
}

func NewReactiveLruCache(capacity int, idleTimeout time.Duration, onEvict EvictCallback) *ReactiveLruCache {
	return &ReactiveLruCache{
		capacity:    capacity,
		idleTimeout: idleTimeout,
		items:       make(map[string]*list.Element),
		evictList:   list.New(),
		onEvict:     onEvict,
	}
}

func (c *ReactiveLruCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, exists := c.items[key]
	if !exists {
		return nil, false
	}

	entry, _ := element.Value.(*cacheEntry)

	if time.Since(entry.lastHeartbeat) > c.idleTimeout {
		c.removeElement(element)
		if c.onEvict != nil {
			go c.onEvict(entry.key)
		}
		return nil, false
	}

	entry.lastHeartbeat = time.Now()
	c.evictList.MoveToFront(element)
	return entry.value, true
}

func (c *ReactiveLruCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[key]; exists {
		c.evictList.MoveToFront(element)
		entry, _ := element.Value.(*cacheEntry)
		entry.value = value
		entry.lastHeartbeat = time.Now()
		return
	}

	entry := &cacheEntry{key: key, value: value, lastHeartbeat: time.Now()}
	element := c.evictList.PushFront(entry)
	c.items[key] = element

	// Каскадное сжатие при переполнении
	c.evictTailCascade()
}

// StartHourlyJanitor страхует систему в Read-Heavy режиме, вычищая хвост раз в час
// StartHourlyJanitor safeguards Read-Heavy states, purging the tail once per hour
func (c *ReactiveLruCache) StartHourlyJanitor(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				// Проверяем самый крайний элемент. Если он жив — весь список выше него гарантированно жив
				tailElement := c.evictList.Back()
				if tailElement != nil {
					tailEntry, _ := tailElement.Value.(*cacheEntry)
					if time.Since(tailEntry.lastHeartbeat) > c.idleTimeout {
						// Запускаем каскадное сжатие
						evictedCount := c.evictTailCascade()
						c.mu.Unlock()

						// Если мы выкинули массив протухших сессий, помогаем рантайму освободить RAM
						if evictedCount > 0 {
							// Принудительно запускаем сборщик мусора в неблокирующем режиме
							runtime.GC()
						}
						continue
					}
				}
				c.mu.Unlock()
			}
		}
	}()
}

// evictTailCascade — внутреннее ядро каскадного сжатия (Должно вызываться под Lock!)
func (c *ReactiveLruCache) evictTailCascade() int {
	evictedCount := 0
	for c.evictList.Len() > c.capacity {
		tailElement := c.evictList.Back()
		if tailElement == nil {
			break
		}

		tailEntry := tailElement.Value.(*cacheEntry)

		if time.Since(tailEntry.lastHeartbeat) > c.idleTimeout {
			c.removeElement(tailElement)
			evictedCount++
			if c.onEvict != nil {
				go c.onEvict(tailEntry.key)
			}
			continue
		}
		break
	}
	return evictedCount
}

func (c *ReactiveLruCache) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*cacheEntry)
	delete(c.items, kv.key)
}
