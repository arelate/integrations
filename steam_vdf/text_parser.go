package steam_vdf

import (
	"errors"
	"io"
	"os"
	"strings"
)

type textParseStateFn func(parser *textParser) textParseStateFn

type textParser struct {
	lex   *textLexer
	last  *KeyValues
	stack []*KeyValues
	kv    []*KeyValues
	err   error
}

func newTextParser(lex *textLexer) *textParser {
	p := &textParser{
		lex: lex,
		kv:  make([]*KeyValues, 0),
	}
	return p
}

func (tp *textParser) run() error {
	for st := parseTextKey; st != nil; {
		st = st(tp)
	}
	if tp.err != nil {
		return tp.err
	}
	return nil
}

func parseTextKey(tp *textParser) textParseStateFn {
	for {
		item := tp.lex.nextItem()
		if item.typ == textItemKeyValue {
			tp.last = &KeyValues{Key: item.val}
			if len(tp.stack) == 0 {
				tp.kv = append(tp.kv, tp.last)
			} else {
				tp.stack[len(tp.stack)-1].Values = append(tp.stack[len(tp.stack)-1].Values, tp.last)
			}
			return parseTextValue
		}
		if item.typ == textItemRightMeta {
			if len(tp.stack) > 0 {
				tp.stack = tp.stack[:len(tp.stack)-1]
			}
			return parseTextKey
		}
		if item.typ == textItemEOF {
			break
		}
		if item.typ == textItemError {
			tp.err = errors.New(item.val)
		}
	}
	return nil
}

func parseTextValue(tp *textParser) textParseStateFn {
	item := tp.lex.nextItem()
	if item.typ == textItemKeyValue {
		if len(tp.stack) > 0 {
			tp.last.Value = &item.val
			return parseTextKey
		}
		tp.err = errors.New("vdf cannot start with a value")
		return nil
	}
	if item.typ == textItemLeftMeta {
		tp.stack = append(tp.stack, tp.last)
		return parseTextKey
	}
	return nil
}

func ParseText(path string) ([]*KeyValues, error) {

	vdfFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer vdfFile.Close()

	sb := &strings.Builder{}

	if _, err := io.Copy(sb, vdfFile); err != nil {
		return nil, err
	}

	p := newTextParser(newTextLexer(sb.String()))
	if err := p.run(); err != nil {
		return nil, p.err
	}

	return p.kv, nil
}
