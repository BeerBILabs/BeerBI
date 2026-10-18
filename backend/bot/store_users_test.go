package main

import (
	"fmt"
	"testing"
	"time"
)

func TestGetCachedUsers_BatchAndChunks(t *testing.T) {
	s := newTestStore(t)

	users := make(map[string]UserInfo)
	ids := make([]string, 0, 1200)
	for i := range 1200 {
		id := fmt.Sprintf("UBATCH%04d", i)
		ids = append(ids, id)
		if i%2 == 0 {
			users[id] = UserInfo{RealName: "Name " + id, ProfileImage: "img-" + id}
		}
	}
	if err := s.SetCachedUsers(users); err != nil {
		t.Fatalf("SetCachedUsers: %v", err)
	}

	got, err := s.GetCachedUsers(ids)
	if err != nil {
		t.Fatalf("GetCachedUsers: %v", err)
	}
	if len(got) != 600 {
		t.Fatalf("got %d rows, want 600", len(got))
	}
	u := got["UBATCH0002"]
	if u.RealName != "Name UBATCH0002" || u.ProfileImage != "img-UBATCH0002" {
		t.Errorf("row = %+v", u)
	}
	if time.Since(u.UpdatedAt) > time.Minute {
		t.Errorf("updated_at not parsed: %v", u.UpdatedAt)
	}
	if _, ok := got["UBATCH0001"]; ok {
		t.Error("unexpected row for uncached ID")
	}
}

func TestSetCachedUsers_UpsertsAndEmpty(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetCachedUsers(nil); err != nil {
		t.Fatalf("empty: %v", err)
	}
	if err := s.SetCachedUser("UUP", "Before", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.SetCachedUsers(map[string]UserInfo{"UUP": {RealName: "After", ProfileImage: "img"}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetCachedUsers([]string{"UUP"})
	if err != nil {
		t.Fatal(err)
	}
	if got["UUP"].RealName != "After" || got["UUP"].ProfileImage != "img" {
		t.Errorf("upsert failed: %+v", got["UUP"])
	}
	if empty, err := s.GetCachedUsers(nil); err != nil || len(empty) != 0 {
		t.Errorf("GetCachedUsers(nil) = %v, %v", empty, err)
	}
}

func TestParseCacheTime(t *testing.T) {
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, in := range []string{"2026-01-02T03:04:05Z", "2026-01-02 03:04:05+00:00", "2026-01-02 03:04:05"} {
		if got := parseCacheTime(in); !got.Equal(want) {
			t.Errorf("parseCacheTime(%q) = %v", in, got)
		}
	}
	if got := parseCacheTime("garbage"); !got.IsZero() {
		t.Errorf("garbage parsed as %v", got)
	}
}
