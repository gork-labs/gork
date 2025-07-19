package generator

import "strings"

// OpenAPITagInfo holds parsed information from `openapi` struct tag.
type OpenAPITagInfo struct {
	Name          string // name override for parameter / field
	In            string // query|header|path|body
	Discriminator string // discriminator field name (for union)
}

// parseOpenAPITag parses a tag value like `my-field,in=query` or `name=X-API-Key,in=header`.
// Items are comma-separated. The first item, if it does not contain '=', is treated as `name`.
// Recognised key=value pairs: name, in, discriminator.
func parseOpenAPITag(tag string) OpenAPITagInfo {
	var info OpenAPITagInfo
	if tag == "" {
		return info
	}
	parts := strings.Split(tag, ",")
	for idx, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if kv := strings.SplitN(p, "=", 2); len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			switch key {
			case "name":
				info.Name = val
			case "in":
				info.In = val
			case "discriminator":
				info.Discriminator = val
			}
		} else {
			// no '=' present
			if idx == 0 && info.Name == "" {
				info.Name = p
			}
		}
	}
	return info
}
