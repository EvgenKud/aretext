package parser

import "errors"

// Regexp represents a regular expression.
type Regexp interface {
	CompileNfa() *Nfa
}

// regexpEmpty represents a language containing only the empty string.
type regexpEmpty struct{}

func (r regexpEmpty) CompileNfa() *Nfa {
	return EmptyStringNfa()
}

// regexpConcat represents the concatenation operation.
type regexpConcat struct {
	left  Regexp
	right Regexp
}

func (r regexpConcat) CompileNfa() *Nfa {
	return r.left.CompileNfa().Concat(r.right.CompileNfa())
}

// regexpUnion represents the union operation.
type regexpUnion struct {
	left  Regexp
	right Regexp
}

func (r regexpUnion) CompileNfa() *Nfa {
	return r.left.CompileNfa().Union(r.right.CompileNfa())
}

// regexpStar represents the Kleene star operation.
type regexpStar struct {
	child Regexp
}

func (r regexpStar) CompileNfa() *Nfa {
	return r.child.CompileNfa().Star()
}

// regexpParenExpr represents an expression in parentheses.
type regexpParenExpr struct {
	child Regexp
}

func (r regexpParenExpr) CompileNfa() *Nfa {
	return r.child.CompileNfa()
}

// regexpChar represents a character match in the regular expression.
type regexpChar struct {
	char byte
}

func (r regexpChar) CompileNfa() *Nfa {
	return NfaForChars([]byte{r.char})
}

// regexpCharClass represents a character class.
type regexpCharClass struct {
	negated bool
	chars   []byte
}

func (r regexpCharClass) CompileNfa() *Nfa {
	if r.negated {
		return NfaForNegatedChars(r.chars)
	}
	return NfaForChars(r.chars)
}

// regexpStartOfText represents the start-of-text pattern (^)
type regexpStartOfText struct{}

func (r regexpStartOfText) CompileNfa() *Nfa {
	return NfaForStartOfText()
}

// regexpEndOfText represents the end-of-text pattern ($)
type regexpEndOfText struct{}

func (r regexpEndOfText) CompileNfa() *Nfa {
	return NfaForEndOfText()
}

// ParseRegexp parses a regular expression string.
func ParseRegexp(s string) (Regexp, error) {
	regexp, _, err := parseRegexp(s, 0, false)
	return regexp, err
}

func parseRegexp(s string, pos int, inParen bool) (Regexp, int, error) {
	if pos >= len(s) {
		return nil, 0, errors.New("Unexpected end of regular expression")
	}

	regexp := Regexp(regexpEmpty{})
	for pos < len(s) {
		switch s[pos] {
		case '(':
			nextRegexp, newPos, err := parseRegexp(s, pos+1, true)
			if err != nil {
				return nil, 0, err
			}

			if newPos >= len(s) || s[newPos] != ')' {
				return nil, 0, errors.New("Expected closing paren")
			}

			if _, ok := regexp.(regexpEmpty); ok {
				regexp = regexpParenExpr{child: nextRegexp}
			} else {
				regexp = regexpConcat{
					left: regexp,
					right: regexpParenExpr{
						child: nextRegexp,
					},
				}
			}

			pos = newPos + 1

		case ')':
			if !inParen {
				return nil, 0, errors.New("Unexpected closing paren")
			}
			return regexp, pos, nil

		case '|':
			nextRegexp, newPos, err := parseRegexp(s, pos+1, inParen)
			if err != nil {
				return nil, 0, err
			}
			if _, ok := regexp.(regexpEmpty); ok {
				return nil, 0, errors.New("Expected characters before union")
			}
			regexp = regexpUnion{left: regexp, right: nextRegexp}
			pos = newPos

		case '*':
			if _, ok := regexp.(regexpEmpty); ok {
				return nil, 0, errors.New("Expected characters before star")
			} else if concat, ok := regexp.(regexpConcat); ok {
				regexp = regexpConcat{
					left:  concat.left,
					right: regexpStar{child: concat.right},
				}
			} else {
				regexp = regexpStar{child: regexp}
			}
			pos++

		case '+':
			if _, ok := regexp.(regexpEmpty); ok {
				return nil, 0, errors.New("Expected characters before plus")
			} else if concat, ok := regexp.(regexpConcat); ok {
				regexp = regexpConcat{
					left: concat.left,
					right: regexpConcat{
						left:  concat.right,
						right: regexpStar{child: concat.right},
					},
				}
			} else {
				regexp = regexpConcat{
					left:  regexp,
					right: regexpStar{child: regexp},
				}
			}
			pos++

		case '?':
			if _, ok := regexp.(regexpEmpty); ok {
				return nil, 0, errors.New("Expected characters before question mark")
			} else if concat, ok := regexp.(regexpConcat); ok {
				regexp = regexpConcat{
					left: concat.left,
					right: regexpUnion{
						left:  regexpEmpty{},
						right: concat.right,
					},
				}
			} else {
				regexp = regexpUnion{
					left:  regexpEmpty{},
					right: regexp,
				}
			}
			pos++

		case '\\':
			nextRegexp, newPos, err := parseEscapeSequence(s, pos)
			if err != nil {
				return nil, 0, err
			}

			if _, ok := regexp.(regexpEmpty); ok {
				regexp = nextRegexp
			} else {
				regexp = regexpConcat{left: regexp, right: nextRegexp}
			}

			pos = newPos

		case '[':
			nextRegexp, newPos, err := parseCharacterClass(s, pos)
			if err != nil {
				return nil, 0, err
			}

			if _, ok := regexp.(regexpEmpty); ok {
				regexp = nextRegexp
			} else {
				regexp = regexpConcat{left: regexp, right: nextRegexp}
			}
			pos = newPos

		case '.':
			// Negation of no characters is equivalent to accepting every character.
			nextRegexp := regexpCharClass{negated: true}
			if _, ok := regexp.(regexpEmpty); ok {
				regexp = nextRegexp
			} else {
				regexp = regexpConcat{left: regexp, right: nextRegexp}
			}
			pos++

		case '^':
			nextRegexp := regexpStartOfText{}
			if _, ok := regexp.(regexpEmpty); ok {
				regexp = nextRegexp
			} else {
				regexp = regexpConcat{left: regexp, right: nextRegexp}
			}
			pos++

		case '$':
			nextRegexp := regexpEndOfText{}
			if _, ok := regexp.(regexpEmpty); ok {
				regexp = nextRegexp
			} else {
				regexp = regexpConcat{left: regexp, right: nextRegexp}
			}
			pos++

		default:
			nextRegexp := regexpChar{char: s[pos]}
			if _, ok := regexp.(regexpEmpty); ok {
				regexp = nextRegexp
			} else {
				regexp = regexpConcat{left: regexp, right: nextRegexp}
			}
			pos++
		}
	}
	return regexp, pos, nil
}

func parseEscapeSequence(s string, pos int) (Regexp, int, error) {
	if pos+1 >= len(s) {
		return nil, 0, errors.New("Invalid escape sequence")
	}

	c := s[pos+1]
	return regexpChar{char: c}, pos + 2, nil
}

func parseCharacterClass(s string, pos int) (Regexp, int, error) {
	regexp := regexpCharClass{}

	// Consume '['
	pos++

	// If '^', consume the carat and mark the char class as negated.
	if pos < len(s) && s[pos] == '^' {
		regexp.negated = true
		pos++
	}

	// Consume all characters up to and including the closing ']'
	for pos < len(s) {
		if s[pos] == ']' {
			pos++
			return regexp, pos, nil
		} else if s[pos] == '\\' {
			if pos+1 >= len(s) {
				return nil, 0, errors.New("Invalid escape sequence in character class")
			} else if c := s[pos+1]; c == '[' || c == ']' || c == '^' || c == '\\' {
				regexp.chars = append(regexp.chars, c)
				pos += 2
			} else {
				return nil, 0, errors.New("Unrecognized escape sequence in character class")
			}
		} else {
			regexp.chars = append(regexp.chars, s[pos])
			pos++
		}
	}

	return nil, 0, errors.New("Expected closing bracket")
}
