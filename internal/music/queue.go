package music

import (
	"errors"
	"strings"
	"sync"
)

var (
	ErrEmptyTitle = errors.New("queue item title cannot be empty")
	ErrQueueFull  = errors.New("queue is full (maximum 100 requests per chat)")
)

const MaxQueueItems = 100

type Item struct {
	Title       string
	RequestedBy string
	URL         string
	ID          string
	DurationSec int
	Thumbnail   string
}

type Queues struct {
	mu    sync.RWMutex
	chats map[int64][]Item
}

func NewQueues() *Queues { return &Queues{chats: make(map[int64][]Item)} }

func (q *Queues) Enqueue(chatID int64, item Item) error {
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" { return ErrEmptyTitle }
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.chats[chatID]) >= MaxQueueItems { return ErrQueueFull }
	q.chats[chatID] = append(q.chats[chatID], item)
	return nil
}

func (q *Queues) List(chatID int64) []Item {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return append([]Item(nil), q.chats[chatID]...)
}

func (q *Queues) Skip(chatID int64) (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	items := q.chats[chatID]
	if len(items) == 0 { return Item{}, false }
	item := items[0]
	if len(items) == 1 { delete(q.chats, chatID) } else { q.chats[chatID] = append([]Item(nil), items[1:]...) }
	return item, true
}

func (q *Queues) Clear(chatID int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.chats, chatID)
}
