module github.com/gork-labs/gork/cmd/lintgork

go 1.24

require (
	github.com/gork-labs/gork/internal/lintgork v0.0.0
	golang.org/x/tools v0.36.0
)

require (
	golang.org/x/mod v0.27.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
)

replace github.com/gork-labs/gork/internal/lintgork => ../../internal/lintgork
