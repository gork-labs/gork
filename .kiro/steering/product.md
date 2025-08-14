# Product Overview

**Gork** is an opinionated Convention Over Configuration OpenAPI framework for Go that provides type-safe HTTP handlers, automatic OpenAPI 3.1.0 generation, and union types. Built for developer productivity and business development efficiency.

## Core Value Proposition

- **Convention Over Configuration**: Eliminates boilerplate through standardized request/response structures
- **Type Safety**: Compile-time validation with strict handler signatures
- **Automatic Documentation**: OpenAPI specs generated from Go source code with zero maintenance
- **Multi-Framework Support**: Works with Gin, Echo, Chi, Gorilla Mux, Fiber, and stdlib
- **Union Types**: Type-safe variants with JSON marshaling for modeling API polymorphism

## Key Features

- Structured request types with standard sections: `Query`, `Body`, `Path`, `Headers`, `Cookies`
- Automatic OpenAPI 3.1.0 spec generation from Go comments and struct tags
- Built-in interactive documentation server
- Custom linter (`lintgork`) for convention compliance
- CLI tool (`gork`) for OpenAPI generation and validation
- 100% test coverage requirement across all modules

## Target Users

- Go developers building REST APIs
- Teams prioritizing developer experience and rapid business development
- Organizations requiring comprehensive API documentation
- Projects needing type-safe HTTP handling with minimal boilerplate