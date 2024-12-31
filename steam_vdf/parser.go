package steam_vdf

import (
	"errors"
	"io"
	"os"
	"strings"
)

type parseStateFn func(parser *parser) parseStateFn

type parser struct {
	lex   *lexer
	last  *KeyValues
	stack []*KeyValues
	kv    []*KeyValues
	err   error
}

func newParser(lex *lexer) *parser {
	p := &parser{
		lex: lex,
		kv:  make([]*KeyValues, 0),
	}
	return p
}

func (p *parser) run() error {
	for st := parseKey; st != nil; {
		st = st(p)
	}
	if p.err != nil {
		return p.err
	}
	return nil
}

func parseKey(p *parser) parseStateFn {
	for {
		item := p.lex.nextItem()
		if item.typ == itemKeyValue {
			p.last = &KeyValues{Key: item.val}
			if len(p.stack) == 0 {
				p.kv = append(p.kv, p.last)
			} else {
				p.stack[len(p.stack)-1].Values = append(p.stack[len(p.stack)-1].Values, p.last)
			}
			return parseValue
		}
		if item.typ == itemRightMeta {
			if len(p.stack) > 0 {
				p.stack = p.stack[:len(p.stack)-1]
			}
			return parseKey
		}
		if item.typ == itemEOF {
			break
		}
		if item.typ == itemError {
			p.err = errors.New(item.val)
		}
	}
	return nil
}

func parseValue(p *parser) parseStateFn {
	item := p.lex.nextItem()
	if item.typ == itemKeyValue {
		if len(p.stack) > 0 {
			p.last.Value = &item.val
			return parseKey
		}
		p.err = errors.New("vdf cannot start with a value")
		return nil
	}
	if item.typ == itemLeftMeta {
		p.stack = append(p.stack, p.last)
		return parseKey
	}
	return nil
}

func Parse(path string) ([]*KeyValues, error) {

	vdfFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer vdfFile.Close()

	sb := &strings.Builder{}

	if _, err := io.Copy(sb, vdfFile); err != nil {
		return nil, err
	}

	p := newParser(newLexer(sb.String()))
	if err := p.run(); err != nil {
		return nil, p.err
	}

	return p.kv, nil
}
