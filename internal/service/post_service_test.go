package service

import (
	"testing"

	"github.com/teblorum/teblorum/internal/model"
)

func TestCreateArticle(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		u := seedServiceUser(t, db, nil)

		post, err := svc.CreateArticle(u.ID, "Test Article Title", "This is the body of the test article with enough characters.", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if post == nil {
			t.Fatal("expected post, got nil")
		}
		if post.Type != "article" {
			t.Errorf("expected type 'article', got '%s'", post.Type)
		}
		if post.Title != "Test Article Title" {
			t.Errorf("expected title 'Test Article Title', got '%s'", post.Title)
		}
		if !post.CommentsEnabled {
			t.Error("expected comments enabled, got disabled")
		}
	})

	t.Run("comments disabled", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		u := seedServiceUser(t, db, nil)

		post, err := svc.CreateArticle(u.ID, "Test Article", "This is the body with enough characters for validation.", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if post.CommentsEnabled {
			t.Error("expected comments disabled, got enabled")
		}
	})

	t.Run("short title", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		u := seedServiceUser(t, db, nil)

		_, err := svc.CreateArticle(u.ID, "AB", "This is a long enough body for validation.", true)
		if err == nil {
			t.Fatal("expected error for short title")
		}
	})

	t.Run("empty title", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		u := seedServiceUser(t, db, nil)

		_, err := svc.CreateArticle(u.ID, "", "This is a long enough body for validation.", true)
		if err == nil {
			t.Fatal("expected error for empty title")
		}
	})

	t.Run("short body", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		u := seedServiceUser(t, db, nil)

		_, err := svc.CreateArticle(u.ID, "Valid Title Here", "Short", true)
		if err == nil {
			t.Fatal("expected error for short body")
		}
	})
}

func TestCreateThread(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		u := seedServiceUser(t, db, nil)

		post, err := svc.CreateThread(u.ID, "Test Thread Title", "This is the body of the test thread with enough characters to pass validation.")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if post.Type != "thread" {
			t.Errorf("expected type 'thread', got '%s'", post.Type)
		}
	})

	t.Run("short title", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		u := seedServiceUser(t, db, nil)

		_, err := svc.CreateThread(u.ID, "XY", "This is a long enough body for validation.")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Привет Мир", "privet-mir"},
		{"Hello World", "hello-world"},
		{"Go 1.22 Released!", "go-122-released"},
		{"  Spaces  Around  ", "spaces-around"},
		{"Special @#$ Characters", "special-characters"},
		{"", "post"},
		{"-leading-and-trailing-", "leading-and-trailing"},
		{"Очень длинный заголовок который должен быть обрезан до восьмидесяти символов и даже больше", ""},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := GenerateSlug(tt.title)
			if tt.want != "" && got != tt.want {
				t.Errorf("GenerateSlug(%q) = %q, want %q", tt.title, got, tt.want)
			}
			if got == "" {
				t.Errorf("GenerateSlug(%q) returned empty slug", tt.title)
			}
			if len(got) > 80 {
				t.Errorf("slug too long: %d characters", len(got))
			}
		})
	}
}

func TestFeed(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	u := seedServiceUser(t, db, nil)

	// Создаём несколько постов
	for i := 0; i < 5; i++ {
		_, err := svc.CreateArticle(u.ID, "Test Article", "This is the body with enough characters for validation.", true)
		if err != nil {
			t.Fatalf("create post: %v", err)
		}
	}

	posts, err := svc.GetFeed(model.PaginationParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 5 {
		t.Errorf("expected 5 posts, got %d", len(posts))
	}

	meta, err := svc.GetFeedMeta(model.PaginationParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("get feed meta: %v", err)
	}
	if meta.Total != 5 {
		t.Errorf("expected total 5, got %d", meta.Total)
	}
}

func TestByType(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	u := seedServiceUser(t, db, nil)

	svc.CreateArticle(u.ID, "Article 1", "Body with enough characters for validation.", true)
	svc.CreateThread(u.ID, "Thread 1", "Body with enough characters for validation.")

	articles, err := svc.GetByType("article", model.PaginationParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 1 {
		t.Errorf("expected 1 article, got %d", len(articles))
	}

	threads, err := svc.GetByType("thread", model.PaginationParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threads) != 1 {
		t.Errorf("expected 1 thread, got %d", len(threads))
	}

	_, err = svc.GetByType("invalid", model.PaginationParams{Page: 1, Limit: 10})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestSoftDeletePost(t *testing.T) {
	t.Run("author can delete own post", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		u := seedServiceUser(t, db, nil)

		post, _ := svc.CreateArticle(u.ID, "Test Article", "Body with enough characters for validation.", true)
		err := svc.SoftDelete(post.ID, u.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Пост должен быть недоступен через GetFeed
		posts, _ := svc.GetFeed(model.PaginationParams{Page: 1, Limit: 10})
		if len(posts) != 0 {
			t.Error("expected no posts in feed after soft delete")
		}
	})

	t.Run("moderator can delete user's post", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		user := seedServiceUser(t, db, map[string]interface{}{"username": "author", "email": "a@b.com"})
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "moderator", "email": "m@b.com", "role": model.RoleModerator})

		post, _ := svc.CreateArticle(user.ID, "Test Article", "Body with enough characters for validation.", true)
		err := svc.SoftDelete(post.ID, mod.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		posts, _ := svc.GetFeed(model.PaginationParams{Page: 1, Limit: 10})
		if len(posts) != 0 {
			t.Error("expected no posts in feed after mod delete")
		}
	})

	t.Run("user cannot delete moderator's post", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		mod := seedServiceUser(t, db, map[string]interface{}{"username": "mod", "email": "m@b.com", "role": model.RoleModerator})
		user := seedServiceUser(t, db, map[string]interface{}{"username": "author", "email": "a@b.com"})

		post, _ := svc.CreateArticle(mod.ID, "Test Article", "Body with enough characters for validation.", true)
		err := svc.SoftDelete(post.ID, user.ID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestUpdatePost(t *testing.T) {
	t.Run("author can update", func(t *testing.T) {
		db := newTestDB(t)
		svc := NewPostService(db)
		u := seedServiceUser(t, db, nil)

		post, _ := svc.CreateArticle(u.ID, "Original Title", "Original body with enough characters for validation.", true)

		commentsEnabled := false
		err := svc.Update(post.ID, u.ID, model.UpdatePostRequest{
			Title:           "Updated Title",
			Body:            "Updated body with enough characters for validation purposes.",
			CommentsEnabled: &commentsEnabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		updated, _ := svc.GetByID(post.ID)
		if updated.Title != "Updated Title" {
			t.Errorf("expected 'Updated Title', got '%s'", updated.Title)
		}
	})
}