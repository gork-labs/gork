package generator

import (
	"fmt"
	"sort"
)

// generateTags collects all unique tags from operations and creates tag definitions
func (g *Generator) generateTags() {
	tagMap := make(map[string]bool)

	// Collect all unique tags from operations
	for _, pathItem := range g.spec.Paths {
		operations := []*Operation{
			pathItem.Get,
			pathItem.Post,
			pathItem.Put,
			pathItem.Delete,
			pathItem.Patch,
		}

		for _, op := range operations {
			if op != nil {
				for _, tag := range op.Tags {
					tagMap[tag] = true
				}
			}
		}
	}

	// Create tag definitions
	var tags []Tag
	for tag := range tagMap {
		tags = append(tags, Tag{
			Name:        tag,
			Description: fmt.Sprintf("Operations related to %s", tag),
		})
	}

	// Sort tags by name for consistent output
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	g.spec.Tags = tags
}
