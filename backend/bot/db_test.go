package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Every pool connection must get the pragmas, not just the first one.
func TestOpenDB_PragmasOnEveryConnection(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "pragmas.db"))
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if got := db.Stats().MaxOpenConnections; got != maxOpenConns {
		t.Errorf("MaxOpenConnections = %d, want %d", got, maxOpenConns)
	}

	ctx := context.Background()
	want := map[string]string{
		"journal_mode": "wal",
		"busy_timeout": "5000",
		"synchronous":  "1", // NORMAL
		"foreign_keys": "1",
		"cache_size":   "-16000",
		"temp_store":   "2", // MEMORY
	}
	// Hold maxOpenConns connections at once so each one is a distinct conn.
	for i := range maxOpenConns {
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatalf("conn %d: %v", i, err)
		}
		defer func() { _ = conn.Close() }()
		for pragma, w := range want {
			var got string
			if err := conn.QueryRowContext(ctx, "PRAGMA "+pragma).Scan(&got); err != nil {
				t.Fatalf("conn %d PRAGMA %s: %v", i, pragma, err)
			}
			if got != w {
				t.Errorf("conn %d PRAGMA %s = %s, want %s", i, pragma, got, w)
			}
		}
	}
}

func TestOpenDB_RejectsQueryInPath(t *testing.T) {
	for _, p := range []string{"bot.db?mode=ro", "bot.db#x"} {
		if _, err := openDB(filepath.Join(t.TempDir(), p)); err == nil {
			t.Errorf("openDB(%q) succeeded, want error", p)
		}
	}
}

func TestOpenDB_FailsOnUnusablePath(t *testing.T) {
	if _, err := openDB(filepath.Join(t.TempDir(), "missing-dir", "bot.db")); err == nil {
		t.Error("openDB in a missing directory succeeded, want error")
	}
}

// Writers (the bot) and readers (dashboards) run concurrently without
// SQLITE_BUSY: WAL lets readers proceed, BEGIN IMMEDIATE plus busy_timeout
// queue the writers.
func TestStore_ConcurrentReadersAndWriters(t *testing.T) {
	s := newTestStore(t)
	day := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)

	const writers, readers, perWorker = 4, 4, 50
	var wg sync.WaitGroup
	errs := make(chan error, (writers+readers)*perWorker)
	for w := range writers {
		wg.Go(func() {
			for i := range perWorker {
				ts := fmt.Sprintf("1740830400.%03d%03d", w, i)
				if err := s.AddBeer(fmt.Sprintf("UW%d", w), "UR", ts, day, 1); err != nil {
					errs <- fmt.Errorf("AddBeer: %w", err)
				}
				if err := s.IncEmoji(fmt.Sprintf("UW%d", w), ":beer:"); err != nil {
					errs <- fmt.Errorf("IncEmoji: %w", err)
				}
			}
		})
	}
	for range readers {
		wg.Go(func() {
			for range perWorker {
				if _, err := s.GetTopUsers(day, day, 10); err != nil {
					errs <- fmt.Errorf("GetTopUsers: %w", err)
				}
				if _, err := s.CountReceivedInDateRange("UR", day, day); err != nil {
					errs <- fmt.Errorf("CountReceived: %w", err)
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	n, err := s.CountReceivedInDateRange("UR", day, day)
	if err != nil {
		t.Fatal(err)
	}
	if n != writers*perWorker {
		t.Errorf("received = %d, want %d", n, writers*perWorker)
	}
}

func TestStore_OptimizeAndClose(t *testing.T) {
	db, err := openDB(filepath.Join(t.TempDir(), "close.db"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Optimize(); err != nil {
		t.Errorf("Optimize: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if err := db.Ping(); err == nil {
		t.Error("db still usable after Close")
	}
	// Optimize on a closed store fails; Close still reports the close result.
	if err := s.Optimize(); err == nil {
		t.Error("Optimize on closed db succeeded")
	}
}

func TestRunOptimizer_StopsOnCancel(t *testing.T) {
	s := newTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.RunOptimizer(ctx, time.Millisecond)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunOptimizer did not return after cancel")
	}
}
