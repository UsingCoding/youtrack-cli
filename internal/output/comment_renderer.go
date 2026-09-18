package output

import (
	"fmt"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func (r *Renderer) Comments(comments []domain.Comment) error {
	switch r.Format {
	case FormatJSON:
		return r.json(commentsJSON(comments))
	case FormatPlain:
		for _, comment := range comments {
			if _, err := fmt.Fprintln(r.Out, comment.ID); err != nil {
				return err
			}
		}
		return nil
	default:
		for i, comment := range comments {
			if i > 0 {
				if _, err := fmt.Fprintln(r.Out); err != nil {
					return err
				}
			}
			if err := r.humanComment(comment); err != nil {
				return err
			}
		}
		return nil
	}
}

func (r *Renderer) Comment(comment domain.Comment) error {
	switch r.Format {
	case FormatJSON:
		return r.json(commentJSON(comment))
	case FormatPlain:
		_, err := fmt.Fprintln(r.Out, comment.ID)
		return err
	default:
		return r.humanComment(comment)
	}
}

func (r *Renderer) CommentRemoval(id string) error {
	switch r.Format {
	case FormatJSON:
		return r.json(CommentRemovalJSON{EntityID: id, Removed: true})
	case FormatPlain:
		return nil
	default:
		_, err := fmt.Fprintf(r.Out, "Comment %s removed (reversible soft removal).\n", id)
		return err
	}
}

func (r *Renderer) humanComment(comment domain.Comment) error {
	text := "-"
	if comment.Text != nil {
		text = *comment.Text
	}
	updated := "-"
	if comment.Updated != nil {
		updated = comment.Updated.Format("2006-01-02 15:04")
	}
	if _, err := fmt.Fprintf(r.Out, "ID: %s\nAuthor: %s\nCreated: %s\nUpdated: %s\nDeleted: %t\nText:\n", comment.ID, commentAuthor(comment.Author), comment.Created.Format("2006-01-02 15:04"), updated, comment.Deleted); err != nil {
		return err
	}
	if _, err := fmt.Fprint(r.Out, text); err != nil {
		return err
	}
	if !strings.HasSuffix(text, "\n") {
		_, err := fmt.Fprintln(r.Out)
		return err
	}
	return nil
}

func commentAuthor(author *domain.User) string {
	if author == nil {
		return "-"
	}
	if author.FullName != "" && author.Login != "" {
		return author.FullName + " (" + author.Login + ")"
	}
	if author.FullName != "" {
		return author.FullName
	}
	if author.Login != "" {
		return author.Login
	}
	return author.ID
}
