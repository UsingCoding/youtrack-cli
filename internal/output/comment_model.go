package output

import (
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type CommentJSON struct {
	EntityID string     `json:"entityId"`
	Author   *UserJSON  `json:"author"`
	Text     *string    `json:"text"`
	Created  time.Time  `json:"created"`
	Updated  *time.Time `json:"updated"`
	Deleted  bool       `json:"deleted"`
}

type CommentRemovalJSON struct {
	EntityID string `json:"entityId"`
	Removed  bool   `json:"removed"`
}

func commentJSON(comment domain.Comment) CommentJSON {
	out := CommentJSON{EntityID: comment.ID, Text: comment.Text, Created: comment.Created, Updated: comment.Updated, Deleted: comment.Deleted}
	if comment.Author != nil {
		out.Author = &UserJSON{EntityID: comment.Author.ID, Login: comment.Author.Login, Name: comment.Author.FullName}
	}
	return out
}

func commentsJSON(comments []domain.Comment) []CommentJSON {
	out := make([]CommentJSON, 0, len(comments))
	for _, comment := range comments {
		out = append(out, commentJSON(comment))
	}
	return out
}
