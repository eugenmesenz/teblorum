package service

import (
	"database/sql"
	"testing"

	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

func TestCreateComment(t *testing.T) {
	t.Run("success root comment", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewCommentService(db)
		u := seedServiceUser(t, db, nil)
		post := seedServicePost(t, db, u.ID)

		comment, err := svc.Create(post.ID, u.ID, nil, "This is a test comment")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if comment == nil {
			t.Fatal("expected comment, got nil")
		}
		if comment.Body != "This is a test comment" {
			t.Errorf("expected body 'This is a test comment', got '%s'", comment.Body)
		}
		if comment.ParentID.Valid {
			t.Error("expected root comment (no parent)")
		}
	})

	t.Run("success reply comment", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewCommentService(db)
		u := seedServiceUser(t, db, nil)
		post := seedServicePost(t, db, u.ID)

		parent, _ := svc.Create(post.ID, u.ID, nil, "Parent comment")
		parentID := parent.ID

		reply, err := svc.Create(post.ID, u.ID, &parentID, "Reply comment")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reply == nil {
			t.Fatal("expected reply, got nil")
		}
		if !reply.ParentID.Valid {
			t.Fatal("expected reply to have parent")
		}
		if reply.ParentID.Int64 != parentID {
			t.Errorf("expected parent_id %d, got %d", parentID, reply.ParentID.Int64)
		}
	})

	t.Run("comments disabled on article", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewCommentService(db)
		u := seedServiceUser(t, db, nil)
		post := seedServicePost(t, db, u.ID, map[string]interface{}{"comments_enabled": false, "type": "article"})

		_, err := svc.Create(post.ID, u.ID, nil, "This comment should be rejected")
		if err == nil {
			t.Fatal("expected error for disabled comments")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewCommentService(db)
		u := seedServiceUser(t, db, nil)
		post := seedServicePost(t, db, u.ID)

		_, err := svc.Create(post.ID, u.ID, nil, "")
		if err == nil {
			t.Fatal("expected error for empty body")
		}
	})

	t.Run("max depth exceeded", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewCommentService(db)
		u := seedServiceUser(t, db, nil)
		post := seedServicePost(t, db, u.ID)

		// Создаём цепочку комментариев: root + maxCommentDepth replies = maxCommentDepth+1 всего
		// Каждый следующий уровень должен быть разрешён, пока parent depth < maxCommentDepth
		var parentID *int64
		for i := 0; i <= maxCommentDepth; i++ {
			if i == 0 {
				comment, err := svc.Create(post.ID, u.ID, nil, "Level 0 comment")
				if err != nil {
					t.Fatalf("level %d: %v", i, err)
				}
				parentID = &comment.ID
			} else {
				comment, err := svc.Create(post.ID, u.ID, parentID, "Level comment")
				if err != nil {
					t.Fatalf("level %d: unexpected error: %v", i, err)
				}
				parentID = &comment.ID
			}
		}
		// Попытка создать ещё один уровень — должна упасть
		_, err := svc.Create(post.ID, u.ID, parentID, "Overflow comment")
		if err == nil {
			t.Fatal("expected error for exceeding max depth, got nil")
		}
	})

	t.Run("parent from different post", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewCommentService(db)
		u := seedServiceUser(t, db, nil)
		post1 := seedServicePost(t, db, u.ID)
		post2 := seedServicePost(t, db, u.ID, map[string]interface{}{"title": "Second Post"})

		parent, _ := svc.Create(post1.ID, u.ID, nil, "Parent on post1")
		parentID := parent.ID

		_, err := svc.Create(post2.ID, u.ID, &parentID, "Reply on post2")
		if err == nil {
			t.Fatal("expected error for different post parent")
		}
	})
}

func TestGetTree(t *testing.T) {
	db := newTestDB(t)
	svc := NewCommentService(db)
	u := seedServiceUser(t, db, nil)
	post := seedServicePost(t, db, u.ID)

	// Создаём несколько комментариев
	c1, _ := svc.Create(post.ID, u.ID, nil, "Root 1")
	c1ID := c1.ID
	c2, _ := svc.Create(post.ID, u.ID, nil, "Root 2")
	c2ID := c2.ID

	svc.Create(post.ID, u.ID, &c1ID, "Reply to root 1")
	svc.Create(post.ID, u.ID, &c2ID, "Reply to root 2")

	tree, err := svc.GetTree(post.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("expected 2 root comments, got %d", len(tree))
	}
	if len(tree[0].Children) != 1 {
		t.Errorf("expected 1 child for root comment, got %d", len(tree[0].Children))
	}
}

func TestSoftDeleteComment(t *testing.T) {
	t.Run("author can delete own comment", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewCommentService(db)
		u := seedServiceUser(t, db, nil)
		post := seedServicePost(t, db, u.ID)

		comment, _ := svc.Create(post.ID, u.ID, nil, "Comment to delete")
		err := svc.SoftDelete(comment.ID, u.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("moderator can delete user comment", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewCommentService(db)
		user := seedServiceUser(t, db, map[string]interface{}{"username": "user", "email": "u@b.com"})
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "m@b.com", "role": model.RoleModerator})
		post := seedServicePost(t, db, user.ID)

		comment, _ := svc.Create(post.ID, user.ID, nil, "Comment to moderate")
		err := svc.SoftDelete(comment.ID, mod.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("user cannot delete moderator comment", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewCommentService(db)
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "m@b.com", "role": model.RoleModerator})
		user := seedServiceUser(t, db, map[string]interface{}{"username": "user", "email": "u@b.com"})
		post := seedServicePost(t, db, mod.ID)

		comment, _ := svc.Create(post.ID, mod.ID, nil, "Mod comment")
		err := svc.SoftDelete(comment.ID, user.ID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// seedServicePost создаёт пост для тестов.
func seedServicePost(t *testing.T, db *sql.DB, authorID int64, overrides ...map[string]interface{}) *model.Post {
	t.Helper()

	title := "Test Post"
	body := "Body with enough characters for validation purposes."
	postType := "article"
	commentsEnabled := 1

	if len(overrides) > 0 {
		if v, ok := overrides[0]["title"]; ok {
			title = v.(string)
		}
		if v, ok := overrides[0]["body"]; ok {
			body = v.(string)
		}
		if v, ok := overrides[0]["type"]; ok {
			postType = v.(string)
		}
		if v, ok := overrides[0]["comments_enabled"]; ok {
			if v.(bool) {
				commentsEnabled = 1
			} else {
				commentsEnabled = 0
			}
		}
	}

	result, err := db.Exec(`
		INSERT INTO posts (author_id, type, title, body, comments_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, authorID, postType, title, body, commentsEnabled)
	if err != nil {
		t.Fatalf("seed post: %v", err)
	}
	id, _ := result.LastInsertId()

	postRepo := repo.NewPostRepo(db)
	post, err := postRepo.GetByID(id)
	if err != nil {
		t.Fatalf("get seeded post: %v", err)
	}
	return post
}
