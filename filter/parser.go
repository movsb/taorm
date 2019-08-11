package filter

// ValueType is
type ValueType uint

// ValueType list
const (
	ValueTypeRaw ValueType = iota
	ValueTypeBoolean
	ValueTypeNumber
	ValueTypeString
)

func (t ValueType) String() string {
	switch t {
	case ValueTypeRaw:
		return "raw"
	case ValueTypeBoolean:
		return "bool"
	case ValueTypeNumber:
		return "number"
	case ValueTypeString:
		return "string"
	default:
		return "unknown"
	}
}

// AndExpression is
type AndExpression struct {
	OrExprs []*Expression
}

// AbstractSyntaxTree (AST) is the syntax tree of expression clauses.
type AbstractSyntaxTree struct {
	AndExprs []*AndExpression
}

// Parser parses tokens into parser tree.
type Parser struct {
	tokenizer *Tokenizer
}

// NewParser creates a new parser.
func NewParser(tokenizer *Tokenizer) *Parser {
	return &Parser{
		tokenizer: tokenizer,
	}
}

// Parse creates a new parser and parses source into AST.
func Parse(source string) (*AbstractSyntaxTree, error) {
	tokenizer := NewTokenizer(source)
	parser := NewParser(tokenizer)
	return parser.Parse()
}

// Parse parses tokens into AST.
func (p *Parser) Parse() (ast *AbstractSyntaxTree, err error) {
	ast = &AbstractSyntaxTree{}

	defer func() {
		if er := recover(); er != nil {
			err = er.(error)
		}
	}()

	ast.AndExprs = p.parseAndExpressions()

	return
}

func (p *Parser) parseAndExpressions() []*AndExpression {
	andExprs := make([]*AndExpression, 0)
	for {
		and := p.parseAndExpression()
		if len(and.OrExprs) == 0 {
			break
		}
		andExprs = append(andExprs, and)
	}
	return andExprs
}

func (p *Parser) parseAndExpression() *AndExpression {
	and := &AndExpression{}
	for {
		token, err := p.tokenizer.Next()
		if err != nil {
			panic(err)
		}
		if token.TokenType == TokenTypeEOF {
			break
		} else if token.TokenType == TokenTypeAnd {
			break
		} else if token.TokenType == TokenTypeOr {
			continue
		} else {
			p.tokenizer.Undo(token)
		}
		expression := p.parseExpression()
		if expression == nil {
			break
		}
		and.OrExprs = append(and.OrExprs, expression)
	}
	return and
}

func (p *Parser) parseExpression() *Expression {
	var expr = &Expression{}
	var token1, token2, token3 Token
	var err error

	token1, err = p.tokenizer.Next()
	if err != nil {
		panic(err)
	}
	if token1.TokenType == TokenTypeEOF {
		return nil
	} else if token1.TokenType == TokenTypeInvalid {
		panic(newSyntaxError("invalid token: %s", token1.TokenValue))
	} else if token1.TokenType != TokenTypeIdent {
		p.tokenizer.Undo(token1)
		return nil
	}

	token2, err = p.tokenizer.Next()
	if err != nil {
		panic(err)
	}
	if token2.TokenType == TokenTypeEOF {
		panic(newSyntaxError("operator expected after %s", token1.TokenValue))
	} else if token2.TokenType == TokenTypeInvalid {
		panic(newSyntaxError("invalid token: %s", token2.TokenValue))
	}

	switch token2.TokenType {
	case TokenTypeEqual, TokenTypeNotEqual:
		break
	case TokenTypeInclude, TokenTypeNotInclude, TokenTypeStartsWith, TokenTypeEndsWith, TokenTypeMatch, TokenTypeNotMatch:
		break
	case TokenTypeGreaterThan, TokenTypeLessThan, TokenTypeGreaterThanOrEqual, TokenTypeLessThanOrEqual:
		break
	default:
		panic(newSyntaxError("unsupported operator: %s", token2.TokenValue))
	}

	token3, err = p.tokenizer.Next()
	if err != nil {
		panic(err)
	}
	if token3.TokenType == TokenTypeEOF {
		// token value defaults to empty string
	} else if token3.TokenType == TokenTypeInvalid {
		panic(newSyntaxError("invalid token: %s", token3.TokenValue))
	} else if token3.TokenType != TokenTypeIdent {
		panic(newSyntaxError("ident expected, but found: %s", token1.TokenValue))
	}

	expr.Name = token1.TokenValue
	expr.Operator = token2
	expr.Value = token3.TokenValue
	expr.Type = ValueTypeRaw

	return expr
}
