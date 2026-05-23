package repo

import (
	"testing"

	"github.com/teblorum/teblorum/internal/model"
)

func TestCommentRepo_Create(t *testing.T) {
	db := NewTestDB(t)
	repo := NewCommentRepo(db)
	author := SeedUser(t, db, nil)
	post := SeedPost(t, db, map[string]interface{}{"author_id": author.ID})

	t.Run("root comment", func(t *testing.T) {
		comment, err := repo.Create(model.CreateCommentRequest{
			PostID:   post.ID,
			AuthorID: author.ID,
			Body:     "Great article!",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if comment.Body != "Great article!" {
			t.Errorf("Body = %q, want %q", comment.Body, "Great article!")
		}
		if comment.ParentID.Valid {
			t.Error("root comment should not have parent_id")
		}
	})

	t.Run("reply to comment", func(t *testing.T) {
		parent, _ := repo.Create(model.CreateCommentRequest{
			PostID:   post.ID,
			AuthorID: author.ID,
			Body:     "Parent",
		})

		reply, err := repo.Create(model.CreateCommentRequest{
			PostID:   post.ID,
			AuthorID: author.ID,
			ParentID: &parent.ID,
			Body:     "Reply",
		})
		if err != nil {
			t.Fatalf("Create reply error = %v", err)
		}
		if !reply.ParentID.Valid {
			t.Fatal("reply should have parent_id")
		}
		if reply.ParentID.Int64 != parent.ID {
			t.Errorf("ParentID = %d, want %d", reply.ParentID.Int64, parent.ID)
		}
	})
}

func TestCommentRepo_GetByPostID(t *testing.T) {
	db := NewTestDB(t)
	repo := NewCommentRepo(db)
	author := SeedUser(t, db, nil)
	post := SeedPost(t, db, map[string]interface{}{"author_id": author.ID})

	// Создаём дерево: root1 → reply1, root2
	root1 := SeedComment(t, db, map[string]interface{}{
		"post_id": post.ID, "author_id": author.ID, "body": "Root 1",
	})
	SeedComment(t, db, map[string]interface{}{
		"post_id": post.ID, "author_id": author.ID, "body": "Root 2",
	})
	SeedComment(t, db, map[string]interface{}{
		"post_id":   post.ID,
		"author_id": author.ID,
		"parent_id": root1.ID,
		"body":      "Reply to root1",
	})

	tree, err := repo.GetByPostID(post.ID)
	if err != nil {
		t.Fatalf("GetByPostID() error = %v", err)
	}

	if len(tree) != 2 {
		t.Fatalf("expected 2 root comments, got %d", len(tree))
	}

	// root1 должен иметь 1 ребёнка
	if len(tree[0].Children) == 0 && len(tree[1].Children) == 0 {
		t.Fatal("expected at least one root to have children")
	}
}

func TestCommentRepo_SoftDelete(t *testing.T) {
	db := NewTestDB(t)
	repo := NewCommentRepo(db)
	author := SeedUser(t, db, nil)
	post := SeedPost(t, db, map[string]interface{}{"author_id": author.ID})
	comment := SeedComment(t, db, map[string]interface{}{
		"post_id": post.ID, "author_id": author.ID,
	})

	err := repo.SoftDelete(comment.ID)
	if err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}

	// Дерево не должно включать удалённый комментарий
	tree, _ := repo.GetByPostID(post.ID)
	if len(tree) != 0 {
		t.Error("soft-deleted comment should not appear in tree")
	}
}
