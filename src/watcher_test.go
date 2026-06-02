package main

import (
	"testing"
	"time"
)

func TestBroadcaster_SubscriberReceivesNotification(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Notify()

	select {
	case <-ch:
		// ok
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive notification")
	}
}

func TestBroadcaster_MultipleSubscribersAllReceive(t *testing.T) {
	b := NewBroadcaster()

	ch1 := b.Subscribe()
	ch2 := b.Subscribe()
	ch3 := b.Subscribe()
	defer b.Unsubscribe(ch1)
	defer b.Unsubscribe(ch2)
	defer b.Unsubscribe(ch3)

	b.Notify()

	for i, ch := range []chan struct{}{ch1, ch2, ch3} {
		select {
		case <-ch:
			// ok
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d did not receive notification", i)
		}
	}
}

func TestBroadcaster_UnsubscribedChannelGetsNothing(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Subscribe()
	b.Unsubscribe(ch)

	b.Notify()

	select {
	case <-ch:
		t.Fatal("unsubscribed channel should not receive notification")
	case <-time.After(50 * time.Millisecond):
		// ok — nothing received
	}
}

func TestBroadcaster_NotifyDoesNotBlockOnFullChannel(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// First notify fills the buffered channel.
	b.Notify()
	// Second notify should not block even though the first hasn't been consumed.
	done := make(chan struct{})
	go func() {
		b.Notify()
		close(done)
	}()

	select {
	case <-done:
		// ok — Notify returned without blocking
	case <-time.After(time.Second):
		t.Fatal("Notify blocked on a full channel")
	}
}

func TestBroadcaster_SubscribeAfterNotifyGetsNothing(t *testing.T) {
	b := NewBroadcaster()

	b.Notify() // no subscribers yet

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	select {
	case <-ch:
		t.Fatal("late subscriber should not see a past notification")
	case <-time.After(50 * time.Millisecond):
		// ok
	}
}

func TestWatchRepo_ReturnsCloseableWatcher(t *testing.T) {
	requireGit(t)
	repoDir := initTestRepo(t)
	b := NewBroadcaster()

	w, err := WatchRepo(repoDir, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Closing should not panic or error.
	if err := w.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}
