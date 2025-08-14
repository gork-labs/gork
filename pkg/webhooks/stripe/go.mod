module github.com/gork-labs/gork/pkg/webhooks/stripe

go 1.24

toolchain go1.24.4

require (
	github.com/gork-labs/gork/pkg/api v0.0.0
	github.com/stripe/stripe-go/v76 v76.25.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gork-labs/gork/pkg/api => ../../api
