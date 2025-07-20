package main

import (
	"github.com/gork-labs/gork/internal/lintgork"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(lintgork.Analyzer)
}
