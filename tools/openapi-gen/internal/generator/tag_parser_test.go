package generator

import "testing"

func TestParseOpenAPITag(t *testing.T) {
	cases := []struct {
		tag      string
		wantName string
		wantIn   string
	}{
		{"my-field,in=query", "my-field", "query"},
		{"name=X-API-Key,in=header", "X-API-Key", "header"},
		{"in=path", "", "path"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.tag, func(t *testing.T) {
			info := parseOpenAPITag(c.tag)
			if info.Name != c.wantName {
				t.Errorf("name: got %q, want %q", info.Name, c.wantName)
			}
			if info.In != c.wantIn {
				t.Errorf("in: got %q, want %q", info.In, c.wantIn)
			}
		})
	}
}
