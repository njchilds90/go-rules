# go-rules

[![Go Reference](https://pkg.go.dev/badge/github.com/njchilds90/go-rules.svg)](https://pkg.go.dev/github.com/njchilds90/go-rules)
[![Go Report Card](https://goreportcard.com/badge/github.com/njchilds90/go-rules)](https://goreportcard.com/report/github.com/njchilds90/go-rules)

Lightweight declarative rule engine for policy evaluation and decisions in Go.

## Features
- Zero external dependencies
- JSON-serializable rules
- Dot-notation nested fields
- Structured results and typed errors
- Context support
- Extensible operators
- Works with maps or structs

## Installation
```bash
go get github.com/njchilds90/go-rules
Usage
Gopackage main

import (
	"fmt"
	"github.com/njchilds90/go-rules"
)

func main() {
	rule := rules.Rule{
		Conditions: []rules.Condition{
			{Field: "user.role", Op: rules.OperatorEQ, Value: "admin"},
			{Field: "score", Op: rules.OperatorGT, Value: 100},
		},
		Logic: rules.LogicAND,
	}

	data := map[string]any{
		"user":  map[string]any{"role": "admin"},
		"score": 150,
	}

	res, err := rules.Evaluate(rule, data)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Matched:", res.Matched, "Explanation:", res.Explanation)
}
See full Godoc for every exported function and examples.
Extensibility
Goengine := rules.New()
engine.Register("custom-op", func(a, b any) (bool, error) { ... })
Perfect for humans writing policies and AI agents generating them.

