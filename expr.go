package conflint

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type Context struct {
	Current *yaml.Node
	Root    *yaml.Node
}

type CondOpOr struct {
	Exprs []BoolExpr
}

func (e *CondOpOr) Eval(ctx Context) (bool, error) {
	any := false

	for _, expr := range e.Exprs {
		r, err := expr.Eval(ctx)
		if err != nil {
			return false, fmt.Errorf("evaluating `%s` within or expr: %w", expr, err)
		}

		any = any || r
	}

	return any, nil
}

func (e *CondOpOr) String() string {
	all := []string{}

	for _, expr := range e.Exprs {
		all = append(all, expr.String())
	}

	return strings.Join(all, " || ")
}

type CondOpAnd struct {
	Exprs []BoolExpr
}

func (e *CondOpAnd) Eval(ctx Context) (bool, error) {
	all := true

	for _, expr := range e.Exprs {
		r, err := expr.Eval(ctx)
		if err != nil {
			return false, fmt.Errorf("evaluating `%s` within and expr: %w", expr, err)
		}

		all = all && r
	}

	return all, nil
}

func (e *CondOpAnd) String() string {
	all := []string{}

	for _, expr := range e.Exprs {
		all = append(all, expr.String())
	}

	return strings.Join(all, " && ")
}

type ExprEq struct {
	Path  *Path
	Value string
	Code  string
}

func (e *ExprEq) String() string {
	return e.Code
}

func (e *ExprEq) Eval(ctx Context) (bool, error) {
	got, err := e.Path.Get(ctx.Current)
	if err != nil {
		return false, fmt.Errorf("evaluating `%s`: %w", e.Code, err)
	}

	return got.Value == e.Value, nil
}

type BoolExpr interface {
	Eval(Context) (bool, error)
	String() string
}

var _ BoolExpr = &ExprEq{}

func parseBoolExp(condStr string) (BoolExpr, error) {
	var any []BoolExpr

	anyComps := strings.Split(condStr, "||")
	for i, anyComp := range anyComps {

		var all []BoolExpr

		andComps := strings.Split(anyComp, "&&")
		for _, andComp := range andComps {
			comp := strings.Split(andComp, "==")
			if len(comp) != 2 {
				return nil, fmt.Errorf("unsupported expression found: wanted an equality expression like `A == B`, but got %v", condStr)
			}

			left := strings.TrimSpace(comp[0])
			right := strings.TrimSpace(comp[1])

			var rightValue string

			switch right[0] {
			case '"':
				rightValue = strings.TrimRight(right[1:], "\"")
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				rightValue = right
			case '\'':
				rightValue = strings.TrimRight(right[1:], "'")
			default:
				switch right {
				case "true", "false":
					rightValue = right
				default:
					return nil, fmt.Errorf("parsing right side of equality: unexpected expression: %v", right)
				}
			}

			nestedPath, err := parseJsonpath(left)
			if err != nil {
				return nil, fmt.Errorf("parsing %s: %w", left, err)
			}

			boolExp := &ExprEq{
				Path:  nestedPath,
				Value: rightValue,
				Code:  condStr,
			}

			all = append(all, boolExp)
		}

		if len(all) == 0 {
			return nil, fmt.Errorf("parsing and components from `%s` at %d", anyComp, i)
		} else if len(all) == 1 {
			any = append(any, all[0])
		} else {
			any = append(any, &CondOpAnd{
				Exprs: all,
			})
		}
	}

	if len(any) == 0 {
		return nil, fmt.Errorf("parsing or components from `%s`", condStr)
	} else if len(any) == 1 {
		return any[0], nil
	}

	return &CondOpOr{Exprs: any}, nil
}
