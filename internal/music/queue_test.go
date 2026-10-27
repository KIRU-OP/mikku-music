package music

import "testing"

func TestQueueEnqueueListSkipAndClear(t *testing.T) {
	q := NewQueues()
	if err := q.Enqueue(10, Item{Title: " first "}); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(10, Item{Title: "second"}); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(20, Item{Title: "other chat"}); err != nil {
		t.Fatal(err)
	}

	items := q.List(10)
	if len(items) != 2 || items[0].Title != "first" || items[1].Title != "second" {
		t.Fatalf("unexpected queue: %#v", items)
	}
	items[0].Title = "mutated copy"
	if q.List(10)[0].Title != "first" {
		t.Fatal("List must return a copy")
	}

	skipped, ok := q.Skip(10)
	if !ok || skipped.Title != "first" {
		t.Fatalf("unexpected skipped item: %#v, %v", skipped, ok)
	}
	if got := q.List(10); len(got) != 1 || got[0].Title != "second" {
		t.Fatalf("unexpected queue after skip: %#v", got)
	}
	q.Clear(10)
	if got := q.List(10); len(got) != 0 {
		t.Fatalf("queue should be empty: %#v", got)
	}
	if got := q.List(20); len(got) != 1 || got[0].Title != "other chat" {
		t.Fatalf("chat queues leaked: %#v", got)
	}
}

func TestQueueRejectsEmptyTitle(t *testing.T) {
	q := NewQueues()
	if err := q.Enqueue(1, Item{Title: "  "}); err != ErrEmptyTitle {
		t.Fatalf("got %v, want %v", err, ErrEmptyTitle)
	}
}

func TestQueueRejectsItemsOverLimit(t *testing.T) {
	q := NewQueues()
	for i := 0; i < MaxQueueItems; i++ {
		if err := q.Enqueue(1, Item{Title: "song"}); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}
	if err := q.Enqueue(1, Item{Title: "one too many"}); err != ErrQueueFull {
		t.Fatalf("got %v, want %v", err, ErrQueueFull)
	}
	if got := len(q.List(1)); got != MaxQueueItems {
		t.Fatalf("queue length = %d, want %d", got, MaxQueueItems)
	}
}
