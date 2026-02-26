// Package rules provides a lightweight, zero-dependency declarative rule engine.
// Rules are JSON-serializable, deterministic, and extensible. Perfect for policies,
// feature flags, and AI-agent decision making.
package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Operator defines supported comparison operators.
type Operator string

const (
	OperatorEQ       Operator = "eq"
	OperatorNE       Operator = "ne"
	OperatorGT       Operator = "gt"
	OperatorGTE      Operator = "gte"
	OperatorLT       Operator = "lt"
	OperatorLTE      Operator = "lte"
	OperatorContains Operator = "contains"
	OperatorIn       Operator = "in"
)

// Condition is a single field-operator-value check.
type Condition struct {
	Field string   `json:"field"`
	Op    Operator `json:"op"`
	Value any      `json:"value"`
}

// Logic combines multiple conditions.
type Logic string

const (
	LogicAND Logic = "and"
	LogicOR  Logic = "or"
)

// Rule is a declarative, JSON-friendly rule.
type Rule struct {
	Conditions []Condition `json:"conditions"`
	Logic      Logic       `json:"logic,omitempty"` // defaults to AND
}

// Result is the machine-readable evaluation outcome.
type Result struct {
	Matched     bool   `json:"matched"`
	Explanation string `json:"explanation,omitempty"`
}

// Engine holds registered operators (minimal state, reusable).
type Engine struct {
	ops map[Operator]func(any, any) (bool, error)
}

// New creates a new Engine with built-in operators.
func New() *Engine {
	e := &Engine{ops: make(map[Operator]func(any, any) (bool, error))}
	e.registerDefaults()
	return e
}

func (e *Engine) registerDefaults() {
	e.ops[OperatorEQ] = func(a, b any) (bool, error) { return equal(a, b), nil }
	e.ops[OperatorNE] = func(a, b any) (bool, error) { return !equal(a, b), nil }
	e.ops[OperatorGT] = greater
	e.ops[OperatorGTE] = func(a, b any) (bool, error) { return greaterOrEqual(a, b) }
	e.ops[OperatorLT] = func(a, b any) (bool, error) { return less(a, b) }
	e.ops[OperatorLTE] = func(a, b any) (bool, error) { return lessOrEqual(a, b) }
	e.ops[OperatorContains] = contains
	e.ops[OperatorIn] = in
}

func (e *Engine) Register(op Operator, fn func(any, any) (bool, error)) {
	e.ops[op] = fn
}

// Default is the shared default engine.
var Default = New()

// Evaluate uses the default engine.
func Evaluate(rule Rule, data map[string]any) (Result, error) {
	return Default.Evaluate(rule, data)
}

// EvaluateWithContext respects context (for future async operators).
func EvaluateWithContext(ctx context.Context, rule Rule, data map[string]any) (Result, error) {
	return Default.EvaluateWithContext(ctx, rule, data)
}

func (e *Engine) Evaluate(rule Rule, data map[string]any) (Result, error) {
	return e.EvaluateWithContext(context.Background(), rule, data)
}

func (e *Engine) EvaluateWithContext(ctx context.Context, rule Rule, data map[string]any) (Result, error) {
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if len(rule.Conditions) == 0 {
		return Result{Matched: true}, nil
	}
	logic := rule.Logic
	if logic == "" {
		logic = LogicAND
	}
	if logic == LogicAND {
		for _, c := range rule.Conditions {
			matched, expl, err := e.evalCondition(ctx, c, data)
			if err != nil {
				return Result{}, err
			}
			if !matched {
				return Result{Matched: false, Explanation: expl}, nil
			}
		}
		return Result{Matched: true, Explanation: "all conditions met"}, nil
	}
	// OR
	for _, c := range rule.Conditions {
		matched, expl, err := e.evalCondition(ctx, c, data)
		if err != nil {
			return Result{}, err
		}
		if matched {
			return Result{Matched: true, Explanation: expl}, nil
		}
	}
	return Result{Matched: false, Explanation: "no conditions met"}, nil
}

func (e *Engine) evalCondition(ctx context.Context, c Condition, data map[string]any) (bool, string, error) {
	if ctx.Err() != nil {
		return false, "", ctx.Err()
	}
	v, ok := getValue(data, c.Field)
	if !ok {
		return false, "", fmt.Errorf("field %q not found: %w", c.Field, errors.New("field not found"))
	}
	fn, ok := e.ops[c.Op]
	if !ok {
		return false, "", fmt.Errorf("unknown operator %q", c.Op)
	}
	matched, err := fn(v, c.Value)
	if err != nil {
		return false, "", err
	}
	expl := fmt.Sprintf("%s %s %v → %t", c.Field, c.Op, c.Value, matched)
	return matched, expl, nil
}

// Helper: getValue supports dot notation for nested maps.
func getValue(data map[string]any, path string) (any, bool) {
	if data == nil {
		return nil, false
	}
	parts := strings.Split(path, ".")
	cur := any(data)
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m[p]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// FromStruct converts a struct to map[string]any (uses JSON round-trip for simplicity and correctness).
func FromStruct(s any) (map[string]any, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Helper comparison functions (pure, deterministic).
func equal(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	if fa, oka := toFloat(a); oka {
		if fb, okb := toFloat(b); okb {
			return fa == fb
		}
	}
	return false
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(x).Int()), true
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(x).Uint()), true
	case string:
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func greater(a, b any) (bool, error) {
	fa, oka := toFloat(a)
	fb, okb := toFloat(b)
	if oka && okb {
		return fa > fb, nil
	}
	return false, fmt.Errorf("type mismatch for >")
}

func greaterOrEqual(a, b any) (bool, error) {
	fa, oka := toFloat(a)
	fb, okb := toFloat(b)
	if oka && okb {
		return fa >= fb, nil
	}
	return false, fmt.Errorf("type mismatch for >=")
}

func less(a, b any) (bool, error) {
	fa, oka := toFloat(a)
	fb, okb := toFloat(b)
	if oka && okb {
		return fa < fb, nil
	}
	return false, fmt.Errorf("type mismatch for <")
}

func lessOrEqual(a, b any) (bool, error) {
	fa, oka := toFloat(a)
	fb, okb := toFloat(b)
	if oka && okb {
		return fa <= fb, nil
	}
	return false, fmt.Errorf("type mismatch for <=")
}

func contains(a, b any) (bool, error) {
	if s, ok := a.(string); ok {
		if search, ok := b.(string); ok {
			return strings.Contains(s, search), nil
		}
	}
	return false, fmt.Errorf("type mismatch for contains")
}

func in(a, b any) (bool, error) {
	slice, ok := b.([]any)
	if !ok {
		return false, fmt.Errorf("in requires slice value")
	}
	for _, item := range slice {
		if equal(a, item) {
			return true, nil
		}
	}
	return false, nil
}
