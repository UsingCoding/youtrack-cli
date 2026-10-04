package output

import (
	"fmt"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func (r *Renderer) WorkItemTypes(items []domain.WorkItemType) error {
	switch r.Format {
	case FormatJSON:
		return r.json(workItemTypesJSON(items))
	case FormatPlain:
		for _, item := range items {
			if _, err := fmt.Fprintln(r.Out, item.ID); err != nil {
				return err
			}
		}
		return nil
	default:
		for _, item := range items {
			if _, err := fmt.Fprintf(r.Out, "ID: %s\nName: %s\n", item.ID, item.Name); err != nil {
				return err
			}
		}
		return nil
	}
}

func (r *Renderer) WorkItems(items []domain.WorkItem) error {
	switch r.Format {
	case FormatJSON:
		return r.json(workItemsJSON(items))
	case FormatPlain:
		for _, item := range items {
			if _, err := fmt.Fprintln(r.Out, item.ID); err != nil {
				return err
			}
		}
		return nil
	default:
		for index, item := range items {
			if index > 0 {
				if _, err := fmt.Fprintln(r.Out); err != nil {
					return err
				}
			}
			if err := r.humanWorkItem(item); err != nil {
				return err
			}
		}
		return nil
	}
}

func (r *Renderer) WorkItem(item domain.WorkItem) error {
	switch r.Format {
	case FormatJSON:
		return r.json(workItemJSON(item))
	case FormatPlain:
		_, err := fmt.Fprintln(r.Out, item.ID)
		return err
	default:
		return r.humanWorkItem(item)
	}
}

func (r *Renderer) WorkItemRemoval(id string) error {
	switch r.Format {
	case FormatJSON:
		return r.json(WorkItemRemovalJSON{EntityID: id, Removed: true})
	case FormatPlain:
		return nil
	default:
		_, err := fmt.Fprintf(r.Out, "Work item %s permanently deleted.\n", id)
		return err
	}
}

func (r *Renderer) humanWorkItem(item domain.WorkItem) error {
	text := "-"
	if item.Text != nil {
		text = *item.Text
	}
	workType := "-"
	if item.Type != nil {
		workType = item.Type.Name
		if workType == "" {
			workType = item.Type.ID
		}
	}
	if _, err := fmt.Fprintf(r.Out, "ID: %s\nDate: %s\nDuration: %dm\nType: %s\nAuthor: %s\n", item.ID, item.Date.UTC().Format("2006-01-02"), item.DurationMinutes, workType, commentAuthor(item.Author)); err != nil {
		return err
	}
	if item.Creator != nil {
		if _, err := fmt.Fprintf(r.Out, "Creator: %s\n", commentAuthor(item.Creator)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(r.Out, "Text:\n", text); err != nil {
		return err
	}
	if !strings.HasSuffix(text, "\n") {
		_, err := fmt.Fprintln(r.Out)
		return err
	}
	return nil
}
