package app

const DefaultPageLimit = 50

func (r PageRequest) Validate() error {
	if r.Offset < 0 {
		return Validationf("offset must be non-negative")
	}
	if r.All && r.Limit != nil {
		return Validationf("--all and --limit are mutually exclusive")
	}
	if r.Limit != nil && *r.Limit < 1 {
		return Validationf("limit must be positive")
	}
	return nil
}

func (r PageRequest) pageLimit() int {
	if r.Limit != nil {
		return *r.Limit
	}
	return DefaultPageLimit
}
