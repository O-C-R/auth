package session

import (
	"testing"
	"time"

	"github.com/O-C-R/auth/id"
	"github.com/garyburd/redigo/redis"
)

func TestSession(t *testing.T) {
	sessionStore, err := NewSessionStore(SessionStoreOptions{
		Addr:            ":6379",
		SessionDuration: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	conn := sessionStore.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("FLUSHDB"); err != nil {
		t.Fatal(err)
	}

	userID, err := id.New()
	if err != nil {
		t.Fatal(err)
	}

	sessionID1, err := id.New()
	if err != nil {
		t.Fatal(err)
	}

	if err := sessionStore.SetSession(sessionID1, userID, "1"); err != nil {
		t.Fatal(err)
	}

	sessionID2, err := id.New()
	if err != nil {
		t.Fatal(err)
	}

	if err := sessionStore.SetSession(sessionID2, userID, "2"); err != nil {
		t.Fatal(err)
	}

	var returnedUserID string
	if err := sessionStore.Session(sessionID1, &returnedUserID); err != nil {
		t.Error(err)
	}

	if returnedUserID != "1" {
		t.Errorf("incorrect user ID, %s, expected %s", returnedUserID, "1")
	}

	// TODO: get group key some other way
	groupKey := "g" + userID.String()

	res, err := redis.Strings(conn.Do("ZRANGE", groupKey, 0, -1))
	if err != nil {
		t.Error(err)
	}
	if len(res) != 2 {
		t.Errorf("Expected 2 sessions in group, got %d: %v", len(res), res)
	}

	if err := sessionStore.DeleteSession(sessionID1); err != nil {
		t.Fatal(err)
	}

	res, err = redis.Strings(conn.Do("ZRANGE", groupKey, 0, -1))
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Errorf("Expected 1 sessions in group, got %d: %v", len(res), res)
	}

	if err := sessionStore.Session(sessionID2, &returnedUserID); err != nil {
		t.Error(err)
	}

	if returnedUserID != "2" {
		t.Errorf("incorrect user ID, %s, expected %s", returnedUserID, "2")
	}

	if err := sessionStore.InvalidateSessions(userID); err != nil {
		t.Error(err)
	}
}

func TestCappedSessions(t *testing.T) {
	sessionStore, err := NewSessionStore(SessionStoreOptions{
		Addr:            ":6379",
		SessionDuration: time.Second,
		MaxSessions:     5,
	})
	if err != nil {
		t.Fatal(err)
	}

	conn := sessionStore.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("FLUSHDB"); err != nil {
		t.Fatal(err)
	}

	userID, err := id.New()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		sessionID, err := id.New()
		if err != nil {
			t.Fatal(err)
		}

		if err := sessionStore.SetSession(sessionID, userID, userID); err != nil {
			t.Fatal(err)
		}
	}

	// TODO: get the group key some other way
	res, err := redis.Strings(conn.Do("ZRANGE", "g"+userID.String(), 0, -1))
	if err != nil {
		t.Error(err)
	}
	if len(res) != 5 {
		t.Errorf("Expected 5 sessions in group, got %d: %v", len(res), res)
	}

	if err := sessionStore.InvalidateSessions(userID); err != nil {
		t.Error(err)
	}
}
