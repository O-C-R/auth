package session

import (
	"testing"
	"time"

	"github.com/O-C-R/auth/id"
)

func TestSession(t *testing.T) {
	sessionStore, err := NewSessionStore(SessionStoreOptions{
		Addr:            ":6379",
		SessionDuration: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	sessionID, err := id.New()
	if err != nil {
		t.Fatal(err)
	}

	userID, err := id.New()
	if err != nil {
		t.Fatal(err)
	}

	if err := sessionStore.SetSession(sessionID, userID); err != nil {
		t.Fatal(err)
	}

	var returnedUserID id.ID
	if err := sessionStore.Session(sessionID, &returnedUserID); err != nil {
		t.Error(err)
	}

	if returnedUserID != userID {
		t.Errorf("incorrect user ID, %s, expected %s", returnedUserID, userID)
	}

	if err := sessionStore.DeleteSession(sessionID); err != nil {
		t.Fatal(err)
	}
}
