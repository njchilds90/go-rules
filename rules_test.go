package rules

import (
	"context"
	"testing"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name     string
		rule     Rule
		data     map[string]any
		want     bool
		wantErr  bool
	}{
		{
			name: "simple eq true",
			rule: Rule{Conditions: []Condition{{Field: "status", Op: OperatorEQ, Value: "active"}}},
			data: map[string]any{"status": "active"},
			want: true,
		},
		{
			name: "and false",
			rule: Rule{
				Conditions: []Condition{
					{Field: "age", Op: OperatorGT, Value: 18},
					{Field: "premium", Op: OperatorEQ, Value: true},
				},
			},
			data: map[string]any{"age": 17, "premium": true},
			want: false,
		},
		{
			name: "or true",
			rule: Rule{
				Conditions: []Condition{
					{Field: "role", Op: OperatorEQ, Value: "admin"},
					{Field: "score", Op: OperatorGT, Value: 100},
				},
				Logic: LogicOR,
			},
			data: map[string]any{"role": "user", "score": 150},
			want: true,
		},
		{
			name: "contains",
			rule: Rule{Conditions: []Condition{{Field: "tags", Op: OperatorContains, Value: "go"}}},
			data: map[string]any{"tags": "golang rules"},
			want: true,
		},
		{
			name: "in",
			rule: Rule{Conditions: []Condition{{Field: "status", Op: OperatorIn, Value: []any{"active", "pending"}}}},
			data: map[string]any{"status": "pending"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Evaluate(tt.rule, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if res.Matched != tt.want {
				t.Errorf("Matched = %v, want %v", res.Matched, tt.want)
			}
		})
	}
}

func TestFromStruct(t *testing.T) {
	type User struct {
		Age     int    `json:"age"`
		Premium bool   `json:"premium"`
	}
	u := User{Age: 25, Premium: true}
	m, err := FromStruct(u)
	if err != nil {
		t.Fatal(err)
	}
	if m["age"] != float64(25) || m["premium"] != true {
		t.Error("FromStruct failed")
	}
}
