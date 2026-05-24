package service

import (
	"testing"

	"github.com/teblorum/teblorum/internal/model"
)

func TestSendMessage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewMessageService(db)
		from := seedServiceUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@b.com"})
		to := seedServiceUser(t, db, map[string]interface{}{"username": "bob", "email": "bob@b.com"})

		msg, err := svc.Send(from.ID, to.ID, "Hello Bob!")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg == nil {
			t.Fatal("expected message, got nil")
		}
		if msg.Body != "Hello Bob!" {
			t.Errorf("expected body 'Hello Bob!', got '%s'", msg.Body)
		}
		if msg.FromUserID != from.ID {
			t.Errorf("expected from %d, got %d", from.ID, msg.FromUserID)
		}
		if msg.ToUserID != to.ID {
			t.Errorf("expected to %d, got %d", to.ID, msg.ToUserID)
		}
	})

	t.Run("cannot send to self", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewMessageService(db)
		u := seedServiceUser(t, db, nil)

		_, err := svc.Send(u.ID, u.ID, "Message to self")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewMessageService(db)
		from := seedServiceUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@b.com"})
		to := seedServiceUser(t, db, map[string]interface{}{"username": "bob", "email": "bob@b.com"})

		_, err := svc.Send(from.ID, to.ID, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("non-existent recipient", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewMessageService(db)
		from := seedServiceUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@b.com"})

		_, err := svc.Send(from.ID, 99999, "Hello")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetConversations(t *testing.T) {
	db := newTestDB(t)
	svc := NewMessageService(db)
	alice := seedServiceUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@b.com"})
	bob := seedServiceUser(t, db, map[string]interface{}{"username": "bob", "email": "bob@b.com"})
	carol := seedServiceUser(t, db, map[string]interface{}{"username": "carol", "email": "carol@b.com"})

	// Alice sends messages to Bob and Carol
	svc.Send(alice.ID, bob.ID, "Hi Bob")
	svc.Send(alice.ID, carol.ID, "Hi Carol")

	convs, err := svc.GetConversations(alice.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(convs) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(convs))
	}
}

func TestGetConversation(t *testing.T) {
	db := newTestDB(t)
	svc := NewMessageService(db)
	alice := seedServiceUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@b.com"})
	bob := seedServiceUser(t, db, map[string]interface{}{"username": "bob", "email": "bob@b.com"})

	svc.Send(alice.ID, bob.ID, "Message 1")
	svc.Send(bob.ID, alice.ID, "Message 2")

	msgs, err := svc.GetConversation(alice.ID, bob.ID, model.PaginationParams{Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestGetUnreadCount(t *testing.T) {
	db := newTestDB(t)
	svc := NewMessageService(db)
	alice := seedServiceUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@b.com"})
	bob := seedServiceUser(t, db, map[string]interface{}{"username": "bob", "email": "bob@b.com"})

	svc.Send(alice.ID, bob.ID, "Unread message")

	count, err := svc.GetUnreadCount(bob.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 unread, got %d", count)
	}

	// After reading
	svc.GetConversation(bob.ID, alice.ID, model.PaginationParams{Page: 1, Limit: 50})
	count, _ = svc.GetUnreadCount(bob.ID)
	if count != 0 {
		t.Errorf("expected 0 unread after reading, got %d", count)
	}
}