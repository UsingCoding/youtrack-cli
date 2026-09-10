package youtrackcli

import "embed"

// SkillsFS contains the agent skill bundled with the CLI.
//
//go:embed skills/**
var SkillsFS embed.FS
