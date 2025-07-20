module github.com/gork-labs/gork/cmd/lintgork

go 1.24

require (
	github.com/gork-labs/gork/internal/lintgork v0.0.0
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d
)

require (
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
)

replace github.com/gork-labs/gork/internal/lintgork => ../../internal/lintgork
