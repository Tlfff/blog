package article

import "testing"

func TestCreateArticleRequest_Validate(t *testing.T) {
	req := CreateArticleRequest{
		Title:   "test",
		Content: "content",
		Tags:    []string{"go"},
		Status:  1,
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}
}
func TestCreateArticleRequest_Validate_TitleEmpty(t *testing.T) {
	req := CreateArticleRequest{
		Title:   "",
		Content: "content",
		Status:  1,
	}

	if err := req.Validate(); err == nil {
		t.Fatalf("expected error for empty title")
	}
}
func TestCreateArticleRequest_Validate_ContentEmpty(t *testing.T) {
	req := CreateArticleRequest{
		Title:   "title",
		Content: "",
		Status:  1,
	}

	if err := req.Validate(); err == nil {
		t.Fatalf("expected error for empty content")
	}
}
func TestUpdateArticleRequest_Validate(t *testing.T) {
	req := UpdateArticleRequest{
		ID:      1,
		Title:   "title",
		Content: "content",
		Status:  1,
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request")
	}
}

func TestUpdateArticleRequest_Validate_IDZero(t *testing.T) {
	req := UpdateArticleRequest{
		ID:      0,
		Title:   "title",
		Content: "content",
	}

	if err := req.Validate(); err == nil {
		t.Fatalf("expected error")
	}
}
func TestDeleteArticleRequest_Validate(t *testing.T) {
	req := DeleteArticleRequest{ID: 1}

	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestDeleteArticleRequest_Validate_Zero(t *testing.T) {
	req := DeleteArticleRequest{ID: 0}

	if err := req.Validate(); err == nil {
		t.Fatalf("expected error")
	}
}
func TestPublishArticleRequest_Validate(t *testing.T) {
	req := PublishArticleRequest{ID: 1}

	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestPublishArticleRequest_Validate_Zero(t *testing.T) {
	req := PublishArticleRequest{ID: 0}

	if err := req.Validate(); err == nil {
		t.Fatalf("expected error")
	}
}
func TestGetDetailRequest_Validate(t *testing.T) {
	req := GetDetailRequest{ID: 1}

	if err := req.Validate(); err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestGetDetailRequest_Validate_Zero(t *testing.T) {
	req := GetDetailRequest{ID: 0}

	if err := req.Validate(); err == nil {
		t.Fatalf("expected error")
	}
}
