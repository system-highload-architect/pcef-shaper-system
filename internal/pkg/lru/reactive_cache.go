package lru

import (
	"context"
	"sync"
	"time"
)

type EvictCallback func(key string)

type Node struct {
	key           string
	value         any
	lastHeartbeat time.Time
	prev          *Node
	next          *Node
}

type ReactiveLruCache struct {
	mu          sync.RWMutex
	capacity    int
	idleTimeout time.Duration
	onEvict     EvictCallback
	wakeupChan  chan struct{}

	items map[string]*Node // b2b-индекс: UUID ➔ Нода
	head  *Node            // Физическая вершина ленты памяти
	tail  *Node            // Физический хвост ленты памяти

	activeCount int   // Числовой виртуальный указатель живых сессий
	activeTail  *Node // Указатель Демона на самый нижний живой узел
}

func NewReactiveLruCache(capacity int, idleTimeout time.Duration, onEvict EvictCallback) *ReactiveLruCache {
	c := &ReactiveLruCache{
		capacity:    capacity,
		idleTimeout: idleTimeout,
		onEvict:     onEvict,
		items:       make(map[string]*Node, capacity),
		wakeupChan:  make(chan struct{}, 1),
	}

	// ИНИЦИАЛИЗАЦИЯ: Аллоцируем жесткую ленту из capacity нод. Память выделяется 1 раз.
	var prevNode *Node
	for i := 0; i < capacity; i++ {
		node := &Node{}
		if c.head == nil {
			c.head = node
		}
		if prevNode != nil {
			prevNode.next = node
			node.prev = prevNode
		}
		prevNode = node
	}
	c.tail = prevNode // Замыкаем физический хвост

	return c
}

func (c *ReactiveLruCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.items[key]
	if !exists {
		return nil, false
	}

	node.lastHeartbeat = time.Now()
	c.moveToHead(node)

	return node.value, true
}

func (c *ReactiveLruCache) Set(key string, value any) {
	c.mu.Lock()

	isEmpty := c.activeCount == 0

	// Сценарий 1: Сессия уже существует в мапе — обновляем данные и пушим в Head
	if node, exists := c.items[key]; exists {
		node.value = value
		node.lastHeartbeat = time.Now()
		c.moveToHead(node)
		c.mu.Unlock()
		return
	}

	// ФАЗА 1: Мы в рамках лимита (Zero-Allocation ротация виртуального хвоста)
	if c.activeCount < c.capacity {
		targetNode := c.tail // Берем свободную ноду строго с физического хвоста

		if targetNode.key != "" {
			delete(c.items, targetNode.key)
		}

		targetNode.key = key
		targetNode.value = value
		targetNode.lastHeartbeat = time.Now()

		c.items[key] = targetNode
		c.activeCount++

		// Вырезаем ноду из физического хвоста и переносим в голову
		if targetNode.prev != nil {
			c.tail = targetNode.prev
			c.tail.next = nil
		}

		targetNode.prev = nil
		targetNode.next = c.head
		if c.head != nil {
			c.head.prev = targetNode
		}
		c.head = targetNode

		// Если это самая первая сессия в пустом кэше, инициализируем activeTail демона
		if isEmpty {
			c.activeTail = targetNode
		}

		c.mu.Unlock()
		triggerWakeup(c.wakeupChan, isEmpty)
		return
	}

	// ФАЗА 2: Аварийное расширение под нагрузкой (activeCount >= capacity)
	// Честно создаем одну новую ноду и аккуратно ставим её в самую вершину c.head
	node := &Node{
		key:           key,
		value:         value,
		lastHeartbeat: time.Now(),
	}

	node.next = c.head
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node

	c.items[key] = node
	c.activeCount++
	c.mu.Unlock()

	triggerWakeup(c.wakeupChan, isEmpty)
}

// StartAdaptiveJanitor — Твой канонический адаптивный демон.
// Больше никаких принудительных выжиганий в циклах. Только ленивый сдвиг стрелки activeTail ВВЕРХ!
func (c *ReactiveLruCache) StartAdaptiveJanitor(ctx context.Context) {
	go func() {
		for {
			c.mu.Lock()

			// Если живых сессий нет — обнуляем указатели и ложимся спать на канале
			if c.activeCount == 0 || c.activeTail == nil {
				c.activeTail = nil
				c.activeCount = 0
				c.mu.Unlock()

				select {
				case <-ctx.Done():
					return
				case <-c.wakeupChan:
					continue
				}
			}

			targetNode := c.activeTail
			timeSinceHeartbeat := time.Since(targetNode.lastHeartbeat)

			// --- ИСПРАВЛЕНО (Уничтожение утечки памяти после DDoS): ---
			// Если время сессии activeTail вышло — проверяем границы физической емкости!
			// FIXED: Dynamically purge physical node allocations only if list length strictly exceeds capacity limits
			if timeSinceHeartbeat >= c.idleTimeout {
				oldKey := targetNode.key

				// Стираем строковый ключ из мапы-индекса в любом случае
				delete(c.items, oldKey)
				c.activeCount--

				// Запоминаем соседа сверху ПЕРЕД какими-либо манипуляциями с узлом
				nextActiveTail := targetNode.prev

				if c.GetLen() > c.capacity {
					// СЦЕНАРИЙ А: Мы вышли за границы capacity (Остатки DDoS шторма).
					// Физически вырезаем этот узел из ленты памяти, возвращая ОЗУ операционной системе!
					c.removeNodePhysical(targetNode)
				}

				// СЦЕНАРИЙ Б: Мы находимся внутри жестких границ емкости (Штатный режим).
				// Узел не трогаем, данные не зануляем!
				// Просто бережно переносим живой указатель Демона ВВЕРХ по ленте памяти.
				c.activeTail = nextActiveTail

				if c.onEvict != nil {
					go c.onEvict(oldKey)
				}
				c.mu.Unlock()
				continue
			}

			// Если нижний живой узел еще жив — высчитываем дельту до его смерти и засыпаем
			sleepDuration := c.idleTimeout - timeSinceHeartbeat
			c.mu.Unlock()

			select {
			case <-ctx.Done():
				return
			case <-time.After(sleepDuration):
				// Проснулись точно под истечение срока activeTail
			}
		}
	}()
}

func (c *ReactiveLruCache) moveToHead(node *Node) {
	if node == c.head {
		return
	}
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if node == c.tail {
		c.tail = node.prev
	}
	node.prev = nil
	node.next = c.head
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node
}

func triggerWakeup(ch chan struct{}, isEmpty bool) {
	if isEmpty {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (c *ReactiveLruCache) GetLen() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activeCount
}

// removeNodePhysical — Вырезает физический узел из кастомной ленты памяти,
// аккуратно перенаправляя указатели его соседей и декрементируя listLen.
// Fixed naming mismatch to satisfy strict compiler linking standards
func (c *ReactiveLruCache) removeNodePhysical(node *Node) {
	if node == nil {
		return
	}
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		c.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.tail = node.prev
	}
}
