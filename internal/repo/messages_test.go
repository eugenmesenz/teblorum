package repo

import (
	"testing"
)

func TestMessageRepo_Send(t *testing.T) {
	db := NewTestDB(t)
	repo := NewMessageRepo(db)
	alice := SeedUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@test.com"})
	bob := SeedUser(t, db, map[string]interface{}{"username": "bob", "email": "bob@test.com"})

	t.Run("first message", func(t *testing.T) {
		msg, err := repo.Send(alice.ID, bob.ID, "Hello Bob!")
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		if msg.Body != "Hello Bob!" {
			t.Errorf("Body = %q, want %q", msg.Body, "Hello Bob!")
		}
		if !msg.ThreadID.Valid {
			t.Fatal("thread_id should be set")
		}
		if msg.ThreadID.Int64 != msg.ID {
			t.Errorf("thread_id should equal message id for first message, got %d", msg.ThreadID.Int64)
		}
	})

	t.Run("reply in same thread", func(t *testing.T) {
		msg1, _ := repo.Send(alice.ID, bob.ID, "Hello Bob!")
		msg2, err := repo.Send(bob.ID, alice.ID, "Hi Alice!")
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		// thread_id должен совпадать
		if msg2.ThreadID.Int64 != msg1.ThreadID.Int64 {
			t.Errorf("thread_id mismatch: %d vs %d", msg2.ThreadID.Int64, msg1.ThreadID.Int64)
		}
	})
}

func TestMessageRepo_GetUnreadCount(t *testing.T) {
	db := NewTestDB(t)
	repo := NewMessageRepo(db)
	alice := SeedUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@test.com"})
	bob := SeedUser(t, db, map[string]interface{}{"username": "bob", "email": "bob@test.com"})

	repo.Send(alice.ID, bob.ID, "Msg 1")
	repo.Send(alice.ID, bob.ID, "Msg 2")

	count, err := repo.GetUnreadCount(bob.ID)
	if err != nil {
		t.Fatalf("GetUnreadCount() error = %v", err)
	}
	if count != 2 {
		t.Errorf("unread count = %d, want 2", count)
	}

	// Alice (отправитель) не должен иметь непрочитанных
	countAlice, _ := repo.GetUnreadCount(alice.ID)
	if countAlice != 0 {
		t.Errorf("alice unread count = %d, want 0", countAlice)
	}
}

func TestMessageRepo_MarkRead(t *testing.T) {
	db := NewTestDB(t)
	repo := NewMessageRepo(db)
	alice := SeedUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@test.com"})
	bob := SeedUser(t, db, map[string]interface{}{"username": "bob", "email": "bob@test.com"})

	repo.Send(alice.ID, bob.ID, "Test")
	repo.Send(alice.ID, bob.ID, "Test 2")

	err := repo.MarkRead(bob.ID, alice.ID)
	if err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}

	count, _ := repo.GetUnreadCount(bob.ID)
	if count != 0 {
		t.Errorf("unread count after MarkRead = %d, want 0", count)
	}
}

func TestMessageRepo_GetConversations(t *testing.T) {
	db := NewTestDB(t)
	repo := NewMessageRepo(db)
	alice := SeedUser(t, db, map[string]interface{}{"username": "alice", "email": "alice@test.com"})
	bob := SeedUser(t, db, map[string]interface{}{"username": "bob", "email": "bob@test.com"})
	charlie := SeedUser(t, db, map[string]interface{}{"username": "charlie", "email": "charlie@test.com"})

	repo.Send(bob.ID, alice.ID, "Bob -> Alice")
	repo.Send(charlie.ID, alice.ID, "Charlie -> Alice")

	convs, err := repo.GetConversations(alice.ID)
	if err != nil {
		t.Fatalf("GetConversations() error = %v", err)
	}
	if len(convs) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(convs))
	}
}
