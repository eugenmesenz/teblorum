package repo

import (
	"errors"
	"testing"

	"github.com/teblorum/teblorum/internal/model"
)

func TestPostRepo_Create(t *testing.T) {
	db := NewTestDB(t)
	repo := NewPostRepo(db)
	author := SeedUser(t, db, nil)

	t.Run("create article", func(t *testing.T) {
		post, err := repo.Create(model.CreatePostRequest{
			AuthorID:        author.ID,
			Type:            "article",
			Title:           "My Article",
			Body:            "Content here",
			CommentsEnabled: true,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if post.Title != "My Article" {
			t.Errorf("Title = %q, want %q", post.Title, "My Article")
		}
		if post.Type != "article" {
			t.Errorf("Type = %q, want %q", post.Type, "article")
		}
		if !post.CommentsEnabled {
			t.Error("CommentsEnabled should be true")
		}
		if post.ID == 0 {
			t.Error("ID should be > 0")
		}
	})

	t.Run("create thread", func(t *testing.T) {
		post, err := repo.Create(model.CreatePostRequest{
			AuthorID:        author.ID,
			Type:            "thread",
			Title:           "My Thread",
			Body:            "First post",
			CommentsEnabled: true,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if post.Type != "thread" {
			t.Errorf("Type = %q, want %q", post.Type, "thread")
		}
	})
}

func TestPostRepo_GetByID(t *testing.T) {
	db := NewTestDB(t)
	repo := NewPostRepo(db)
	author := SeedUser(t, db, nil)
	post := SeedPost(t, db, map[string]interface{}{"author_id": author.ID})

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetByID(post.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.ID != post.ID {
			t.Errorf("ID = %d, want %d", got.ID, post.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByID(99999)
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("expected %v, got %v", model.ErrNotFound, err)
		}
	})
}

func TestPostRepo_GetFeed(t *testing.T) {
	db := NewTestDB(t)
	repo := NewPostRepo(db)
	author := SeedUser(t, db, nil)

	// Создаём 5 постов
	for i := 0; i < 5; i++ {
		SeedPost(t, db, map[string]interface{}{
			"author_id": author.ID,
			"title":     "Post",
		})
	}

	t.Run("paginated", func(t *testing.T) {
		posts, err := repo.GetFeed(model.PaginationParams{Page: 1, Limit: 3})
		if err != nil {
			t.Fatalf("GetFeed() error = %v", err)
		}
		if len(posts) != 3 {
			t.Errorf("got %d posts, want 3", len(posts))
		}
	})

	t.Run("second page", func(t *testing.T) {
		posts, err := repo.GetFeed(model.PaginationParams{Page: 2, Limit: 3})
		if err != nil {
			t.Fatalf("GetFeed() error = %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("got %d posts, want 2", len(posts))
		}
	})
}

func TestPostRepo_SoftDelete(t *testing.T) {
	db := NewTestDB(t)
	repo := NewPostRepo(db)
	author := SeedUser(t, db, nil)
	post := SeedPost(t, db, map[string]interface{}{"author_id": author.ID})

	err := repo.SoftDelete(post.ID)
	if err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}

	// Пост должен быть недоступен через GetByID (due to soft delete filtering)
	// Actually, GetByID returns deleted_at, so it'll still find it but with deleted_at set
	softDeleted, _ := repo.GetByID(post.ID)
	if !softDeleted.DeletedAt.Valid {
		t.Error("DeletedAt should be set after soft delete")
	}

	// Feed не должен включать удалённые посты
	posts, _ := repo.GetFeed(model.PaginationParams{Page: 1, Limit: 10})
	for _, p := range posts {
		if p.ID == post.ID {
			t.Error("soft-deleted post should not appear in feed")
		}
	}
}

func TestPostRepo_GetByType(t *testing.T) {
	db := NewTestDB(t)
	repo := NewPostRepo(db)
	author := SeedUser(t, db, nil)

	article := SeedPost(t, db, map[string]interface{}{
		"author_id": author.ID,
		"type":      "article",
		"title":     "Article 1",
	})
	SeedPost(t, db, map[string]interface{}{
		"author_id": author.ID,
		"type":      "thread",
		"title":     "Thread 1",
	})

	t.Run("articles only", func(t *testing.T) {
		posts, err := repo.GetByType("article", model.PaginationParams{Page: 1, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(posts) != 1 || posts[0].ID != article.ID {
			t.Errorf("expected 1 article, got %d", len(posts))
		}
	})

	t.Run("threads only", func(t *testing.T) {
		posts, err := repo.GetByType("thread", model.PaginationParams{Page: 1, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(posts) != 1 {
			t.Errorf("expected 1 thread, got %d", len(posts))
		}
	})
}

func TestPostRepo_GetByAuthorID(t *testing.T) {
	db := NewTestDB(t)
	repo := NewPostRepo(db)
	author1 := SeedUser(t, db, nil)
	author2 := SeedUser(t, db, map[string]interface{}{
		"username": "author2",
		"email":    "a2@example.com",
	})

	for i := 0; i < 3; i++ {
		SeedPost(t, db, map[string]interface{}{"author_id": author1.ID})
	}
	SeedPost(t, db, map[string]interface{}{"author_id": author2.ID})

	posts, err := repo.GetByAuthorID(author1.ID, model.PaginationParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 3 {
		t.Errorf("expected 3 posts for author1, got %d", len(posts))
	}
}

func TestPostRepo_Update(t *testing.T) {
	db := NewTestDB(t)
	repo := NewPostRepo(db)
	author := SeedUser(t, db, nil)
	post := SeedPost(t, db, map[string]interface{}{
		"author_id": author.ID,
		"title":     "Original Title",
	})

	enabled := false
	err := repo.Update(post.ID, model.UpdatePostRequest{
		Title:           "Updated Title",
		Body:            "Updated body",
		CommentsEnabled: &enabled,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, _ := repo.GetByID(post.ID)
	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", updated.Title, "Updated Title")
	}
	if updated.Body != "Updated body" {
		t.Errorf("Body = %q, want %q", updated.Body, "Updated body")
	}
	if updated.CommentsEnabled {
		t.Error("CommentsEnabled should be false")
	}
}
