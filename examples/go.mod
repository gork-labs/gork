module github.com/gork-labs/gork/examples

go 1.24

require (
	github.com/gork-labs/gork/pkg/api v0.0.0
	github.com/gork-labs/gork/pkg/unions v0.0.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/kr/text v0.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/gork-labs/gork/cmd/gork => ../cmd/gork
	github.com/gork-labs/gork/pkg/api => ../pkg/api
	github.com/gork-labs/gork/pkg/unions => ../pkg/unions
)
