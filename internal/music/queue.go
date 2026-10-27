package music

import (
	"errors"
	"sync"
)

var ErrEmptyTitle = errors.New("queue item title cannot be empty")

type Item struct {
	Title       string
	RequestedBy string
}

type Queues struct {
	mu     sync.RWMutex
	chats  map[int64][]Item
}

func NewQueues() *Queues { return &Queues{chats: make(map[int64][]Item)} }

func (q *Queues) Enqueue(chatID int64, item Item) error {
	if item.Title == "" {
		return ErrEmptyTitle
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.chats[chatID] = append(q.chats[chatID], item)
	return nil
}

func (q *Queues) List(chatID int64) []Item {
	q.mu.RLock()
	defer q.mu.RUnlock()
	items := q.chats[chatID]
	return append([]Item(nil), items...)
}

func (q *Queues) Skip(chatID int64) (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	items := q.chats[chatID]
	if len(items) == 0 {
		return Item{}, false
	}
	item := items[0]
	if len(items) == 1 {
		delete(q.chats, chatID)
	} else {
		q.chats[chatID] = append([]Item(nil), items[1:]...)
	}
	return item, true
}

func (q *Queues) Clear(chatID int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.chats, chatID)
}
