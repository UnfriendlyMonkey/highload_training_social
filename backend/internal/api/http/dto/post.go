package dto

import "github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"

type CreatePostRequest struct {
	Text *string `json:"text"`
}

type UpdatePostRequest struct {
	ID   string  `json:"id"`
	Text *string `json:"text"`
}

// CreatePostResponse is encoded as a JSON string, as specified by the API.
type CreatePostResponse string

type PostResponse struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	AuthorID string `json:"author_user_id"`
}

func (r *PostResponse) FromEntity(post postgres.Post) *PostResponse {
	r.ID = post.ID.String()
	r.Text = post.Text
	r.AuthorID = post.AuthorID.String()
	return r
}
