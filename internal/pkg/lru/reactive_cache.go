package lru

import (
	"container/list"
	"sync"
	"time"
)

// EvictCallback — финтех-функция для асинхронного возврата квот в OCS при вытеснении сессии
type EvictCallback func(imsi string)

type cacheEntry struct {
	key           string
	value         any
	lastHeartbeat time.Time
}

// ReactiveLruCache реализует ленивое вытеснение по тайм-ауту без фоновых воркеров (Req. 4)
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

// Get извлекает сессию с ленивой проверкой протухания за O(1)
func (c *ReactiveLruCache) Get(key string) (any, bool) {
	c.mu.Lock() // Используем Lock, так как чтение перемещает узел в списке (мутация)
	defer c.mu.Unlock()

	element, exists := c.items[key]
	if !exists {
		return nil, false
	}

	entry := element.Value.(*cacheEntry)

	// ПАТТЕРН ДАВИДА: Ленивая проверка протухания на горячем пути
	if time.Since(entry.lastHeartbeat) > c.idleTimeout {
		// Сессия протухла — реактивно уничтожаем её, освобождая RAM
		c.removeElement(element)
		// Асинхронно триггерим финтех-возврат денег в OCS биллинг
		if c.onEvict != nil {
			go c.onEvict(entry.key)
		}
		return nil, false
	}

	// Сессия жива — продлеваем Heartbeat и двигаем в топ списка (Move to Front)
	entry.lastHeartbeat = time.Now()
	c.evictList.MoveToFront(element)
	return entry.value, true
}

// Set добавляет сессию. Если лимит превышен — реактивно вычищает "хвост" за O(1)
func (c *ReactiveLruCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Если ключ уже есть, просто обновляем значение и таймстамп
	if element, exists := c.items[key]; exists {
		c.evictList.MoveToFront(element)
		entry := element.Value.(*cacheEntry)
		entry.value = value
		entry.lastHeartbeat = time.Now()
		return
	}

	// Создаем новый узел и пихаем в топ хронологической шкалы
	entry := &cacheEntry{key: key, value: value, lastHeartbeat: time.Now()}
	element := c.evictList.PushFront(entry)
	c.items[key] = element

	// Проверяем пограничное условие превышения лимита емкости памяти
	if c.evictList.Len() > c.capacity {
		// Мгновенно смотрим на самый старый неактивный элемент в хвосте списка (O(1))
		tailElement := c.evictList.Back()
		if tailElement != nil {
			tailEntry := tailElement.Value.(*cacheEntry)

			// Вытесняем мертвую или самую старую сессию из RAM
			c.removeElement(tailElement)
			if c.onEvict != nil {
				go c.onEvict(tailEntry.key)
			}
		}
	}
}

func (c *ReactiveLruCache) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*cacheEntry)
	delete(c.items, kv.key)
}
