package tui

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// HighlightCode applies syntax highlighting to a code string using chroma.
// The language parameter selects the lexer; if empty or unknown, chroma
// tries to auto-detect from the code content.
func HighlightCode(code, language string) string {
	lexer := pickLexer(language, code)
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	var b strings.Builder
	if err := formatter.Format(&b, style, iterator); err != nil {
		return code
	}
	return b.String()
}

func pickLexer(language, code string) chroma.Lexer {
	if language != "" {
		if l := lexers.Get(language); l != nil {
			return l
		}
	}
	if l := lexers.Analyse(code); l != nil {
		return l
	}
	return lexers.Fallback
}
