package conflint

import (
	"fmt"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type Path struct {
	Getter []Getter
}

type Getter struct {
	Get  func(node *yaml.Node) (*yaml.Node, error)
	Expr string
}

func yamlMapGet(key string) Getter {
	return Getter{
		Expr: "." + key,
		Get: func(node *yaml.Node) (*yaml.Node, error) {
			var found *yaml.Node

			switch node.Kind {
			case yaml.MappingNode:
				for j := 0; j < len(node.Content); j += 2 {
					k := node.Content[j]
					v := node.Content[j+1]

					if k.Value == key {
						found = v
						break
					}
				}
			case yaml.SequenceNode:
				idx, err := strconv.Atoi(key)
				if err != nil {
					return nil, fmt.Errorf("converting %q to int: %w", key, err)
				}

				found = node.Content[idx]
			default:
				return nil, fmt.Errorf("expected mapping or sequence node: got %+v(%v)", node, node.Kind)
			}

			if found == nil {
				return nil, fmt.Errorf("%s does not have child named %s", node.Value, key)
			}

			return found, nil
		},
	}
}

func yamlArrayElemIndex(idx int) Getter {
	return Getter{
		Expr: fmt.Sprintf("[%d]", idx),
		Get: func(node *yaml.Node) (*yaml.Node, error) {
			if node.Kind != yaml.SequenceNode {
				return nil, fmt.Errorf("expected sequence node: got %v(%v)", node.Value, node.Kind)
			}

			if idx > len(node.Content)-1 {
				return nil, fmt.Errorf("index out of range: index = %v, len = %v, value = %+v", idx, len(node.Content), node)
			}

			return node.Content[idx], nil
		},
	}
}

func yamlArrayElemWhere(expr BoolExpr) Getter {
	return Getter{
		Expr: expr.String(),
		Get: func(node *yaml.Node) (*yaml.Node, error) {
			if node.Kind != yaml.SequenceNode {
				return nil, fmt.Errorf("expected sequence node: got %v(%v)", node.Value, node.Kind)
			}

			for i, n := range node.Content {
				ok, err := expr.Eval(Context{Current: n})
				if err != nil {
					return nil, fmt.Errorf("reading index %d in array: %w", i, err)
				}

				if ok {
					return n, nil
				}
			}

			return nil, fmt.Errorf("no array element matching `%s` found in %+v", expr.String(), node)
		},
	}
}

func (p *Path) Get(node *yaml.Node) (*yaml.Node, error) {
	for i, g := range p.Getter {
		next, err := g.Get(node)
		if err != nil {
			return nil, fmt.Errorf("evaluating jsonpath `%s` at %d: %w", g.Expr, i, err)
		}

		node = next
	}

	return node, nil
}

func parseJsonpath(expr string) (*Path, error) {
	type State int
	const (
		Init State = iota
		Next
		ReadPropertySelector
		ReadArrayElementWithWhereClause
		ReadArrayElementWithInlineWhereClause
		ReadArrayElementWithIndex
		End
	)

	state := Init

	var path Path

	for i := 0; i < len(expr); {
		cur := expr[i]

		switch state {
		case Init:
			if cur != '$' && cur != '@' {
				return nil, fmt.Errorf("expression must start with $ or @, but got: %c in %s", cur, expr)
			}

			if expr[i+1] != '.' {
				return nil, fmt.Errorf("the root element must be an object value to be queried by .KEY, but got %s in %s", expr[i:i+2], expr)
			}

			state = ReadPropertySelector
			i += 1
		case Next:
			if expr[i] == '.' {
				state = ReadPropertySelector
			} else if strings.HasPrefix(expr[i:], "[*]?(@") {
				state = ReadArrayElementWithWhereClause
			} else if strings.HasPrefix(expr[i:], "[?(@") {
				state = ReadArrayElementWithInlineWhereClause
			} else if strings.HasPrefix(expr[i:], "[") {
				state = ReadArrayElementWithIndex
			} else {
				return nil, fmt.Errorf("reading next token from %s: unexpected string in buffer. expected any of %v", expr[i:], strings.Join([]string{".", "["}, ", "))
			}
		case ReadPropertySelector:
			start := i + 1
			j := start

			for {
				if j == len(expr)-1 {
					state = End
					break
				} else if expr[j+1] == '.' || expr[j+1] == '[' {
					state = Next
					break
				} else if expr[j+1] == '-' || expr[j+1] == '+' {
					return nil, fmt.Errorf("reading property selector from %s: unexpected character found at beginning", expr[j:])
				}

				j++
			}

			getter := yamlMapGet(expr[start : j+1])

			path.Getter = append(path.Getter, getter)

			i = j + 1
		case ReadArrayElementWithIndex:
			j := i
			idxStart := i + 1

			for {
				if j == len(expr)-1 {
					return nil, fmt.Errorf("reading array element with index: unexpected EOS found at index %d", j)
				} else if expr[j+1] == ']' {
					break
				}
				j++
			}

			idxStr := expr[idxStart : j+1]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, fmt.Errorf("converting %s to int: %w", idxStr, err)
			}

			getter := yamlArrayElemIndex(idx)

			path.Getter = append(path.Getter, getter)

			state = Next

			i = j + 2
		case ReadArrayElementWithWhereClause:
			condStart := i + 5
			j := condStart

			for {
				if j == len(expr)-1 {
					return nil, fmt.Errorf("reading array element with where clause: unexpected EOS found at index %d", j)
				} else if expr[j+1] == ')' {
					break
				}
				j++
			}

			condStr := expr[condStart : j+1]

			boolExp, err := parseBoolExp(condStr)
			if err != nil {
				return nil, fmt.Errorf("parsing %s: %w", condStr, err)
			}

			getter := yamlArrayElemWhere(boolExp)

			path.Getter = append(path.Getter, getter)

			state = Next

			i = j + 2
		case ReadArrayElementWithInlineWhereClause:
			condStart := i + 3
			j := condStart

			for {
				if j == len(expr)-1 {
					return nil, fmt.Errorf("reading array element with inline where clause: unexpected EOS found at index %d", j)
				} else if expr[j+1] == ')' {
					break
				}
				j++
			}

			condStr := expr[condStart : j+1]

			boolExp, err := parseBoolExp(condStr)
			if err != nil {
				return nil, fmt.Errorf("parsing %s: %w", condStr, err)
			}

			getter := yamlArrayElemWhere(boolExp)

			path.Getter = append(path.Getter, getter)

			state = Next

			i = j + 3
		case End:

		default:
			panic(fmt.Errorf("unexpected state: %v", state))
		}
	}

	return &path, nil
}
