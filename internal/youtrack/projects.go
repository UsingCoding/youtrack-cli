package youtrack

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack/dto"
)

func (c *Client) GetProject(ctx context.Context, ref domain.ProjectRef) (domain.Project, error) {
	var p dto.Project
	q := url.Values{"fields": []string{projectFields}}
	err := c.doJSON(ctx, http.MethodGet, "/api/admin/projects/"+string(ref), q, nil, &p)
	if err != nil {
		return domain.Project{}, err
	}
	return domain.Project{ID: p.ID, Name: p.Name, ShortName: p.ShortName}, nil
}

func (c *Client) SearchProjects(ctx context.Context, query string) ([]domain.Project, error) {
	items, err := paginate(ctx, func(ctx context.Context, skip, top int) ([]dto.Project, error) {
		var page []dto.Project
		q := url.Values{"fields": []string{projectFields}, "$skip": []string{itoa(skip)}, "$top": []string{itoa(top)}, "query": []string{query}}
		err := c.doJSON(ctx, http.MethodGet, "/api/admin/projects", q, nil, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Project, 0, len(items))
	for _, p := range items {
		out = append(out, domain.Project{ID: p.ID, Name: p.Name, ShortName: p.ShortName})
	}
	return out, nil
}

func (c *Client) ListProjectFields(ctx context.Context, projectID string) ([]domain.FieldDefinition, error) {
	items, err := paginate(ctx, func(ctx context.Context, skip, top int) ([]dto.ProjectCustomField, error) {
		var page []dto.ProjectCustomField
		q := url.Values{"fields": []string{projectCustomFieldFields}, "$skip": []string{itoa(skip)}, "$top": []string{itoa(top)}}
		err := c.doJSON(ctx, http.MethodGet, "/api/admin/projects/"+projectID+"/customFields", q, nil, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.FieldDefinition, 0, len(items))
	for _, item := range items {
		kind, cardinality, err := parseFieldType(item.Field.FieldType.ID, item.Field.FieldType.ValueType, item.Field.FieldType.IsMultiValue)
		if err != nil {
			kind = domain.FieldUnknown
			cardinality = domain.CardinalitySingle
		}
		def := domain.FieldDefinition{ID: item.ID, Name: item.Field.Name, Kind: kind, Cardinality: cardinality, CanBeEmpty: item.CanBeEmpty}
		if item.Bundle != nil {
			def.BundleID = item.Bundle.ID
		}
		out = append(out, def)
	}
	return out, nil
}

func (c *Client) ListFieldOptions(ctx context.Context, projectID string, field domain.FieldDefinition) ([]domain.FieldOption, error) {
	items, err := paginate(ctx, func(ctx context.Context, skip, top int) ([]dto.BundleValue, error) {
		var page []dto.BundleValue
		q := url.Values{"fields": []string{bundleValueFields}, "$skip": []string{itoa(skip)}, "$top": []string{itoa(top)}}
		path := "/api/admin/projects/" + projectID + "/customFields/" + field.ID + "/bundle/values"
		err := c.doJSON(ctx, http.MethodGet, path, q, nil, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.FieldOption, 0, len(items))
	for _, item := range items {
		if item.Archived {
			continue
		}
		out = append(out, domain.FieldOption{ID: item.ID, Name: item.Name})
	}
	return out, nil
}

func (c *Client) ListFieldUsers(ctx context.Context, projectID string, field domain.FieldDefinition) ([]domain.User, error) {
	if field.BundleID == "" {
		return nil, app.Runtimef("user field %q has no user bundle", field.Name)
	}
	items, err := paginate(ctx, func(ctx context.Context, skip, top int) ([]dto.User, error) {
		var page []dto.User
		q := url.Values{"fields": []string{userFields}, "$skip": []string{itoa(skip)}, "$top": []string{itoa(top)}}
		path := "/api/admin/customFieldSettings/bundles/user/" + field.BundleID + "/aggregatedUsers"
		err := c.doJSON(ctx, http.MethodGet, path, q, nil, &page)
		return page, err
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(items))
	for _, u := range items {
		out = append(out, domain.User{ID: u.ID, Login: u.Login, FullName: u.FullName})
	}
	return out, nil
}

func parseFieldType(id, valueType string, isMultiValue bool) (domain.FieldKind, domain.Cardinality, error) {
	v := id
	if v == "" {
		v = valueType
	}
	v = strings.TrimSpace(v)
	card := domain.CardinalitySingle
	if isMultiValue {
		card = domain.CardinalityMulti
	}
	if strings.HasSuffix(v, "[*]") {
		card = domain.CardinalityMulti
		v = strings.TrimSuffix(v, "[*]")
	}
	v = strings.TrimSuffix(v, "[1]")
	switch v {
	case "string":
		return domain.FieldString, card, nil
	case "integer":
		return domain.FieldInteger, card, nil
	case "float":
		return domain.FieldFloat, card, nil
	case "date":
		return domain.FieldDate, card, nil
	case "date and time", "dateTime":
		return domain.FieldDateTime, card, nil
	case "period":
		return domain.FieldPeriod, card, nil
	case "text":
		return domain.FieldText, card, nil
	case "enum":
		return domain.FieldEnum, card, nil
	case "state":
		return domain.FieldState, card, nil
	case "user":
		return domain.FieldUser, card, nil
	case "version":
		return domain.FieldVersion, card, nil
	case "build":
		return domain.FieldBuild, card, nil
	case "ownedField", "owned":
		return domain.FieldOwned, card, nil
	case "group":
		return domain.FieldGroup, card, nil
	default:
		return domain.FieldUnknown, card, app.Validationf("unknown YouTrack field type %q", v)
	}
}
