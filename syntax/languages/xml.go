package languages

import (
	"unicode"

	"github.com/aretext/aretext/syntax/parser"
)

const (
	xmlTokenRoleAttrKey         = parser.TokenRoleCustom1
	xmlTokenRoleCharacterEntity = parser.TokenRoleCustom2
	xmlTokenRoleCData           = parser.TokenRoleCustom3
	xmlTokenRoleTag             = parser.TokenRoleCustom4
	xmlTokenRolePrologue        = parser.TokenRoleCustom5
)

type xmlParseState uint8

const (
	xmlParseStateNormal = xmlParseState(iota)
	xmlParseStateInTag
)

func (s xmlParseState) Equals(other parser.State) bool {
	otherState, ok := other.(xmlParseState)
	return ok && s == otherState
}

// XmlParseFunc returns a parse func for XML.
// See https://www.w3.org/TR/2006/REC-xml11-20060816/
func XmlParseFunc() parser.Func {
	parsePrologue := matchState(
		xmlParseStateNormal,
		consumeString("<?").
			Then(consumeToString("?>")).
			Map(recognizeToken(xmlTokenRolePrologue)))

	parseCData := matchState(
		xmlParseStateNormal,
		consumeString("<![CDATA[").
			Then(consumeToString("]]>")).
			Map(recognizeToken(xmlTokenRoleCData)))

	parseComment := matchState(
		xmlParseStateNormal,
		consumeString("<!--").
			Then(consumeToString("-->")).
			Map(recognizeToken(parser.TokenRoleComment)))

	parseTagStart := matchState(
		xmlParseStateNormal,
		consumeLongestMatchingOption([]string{"<", "</"}).
			ThenMaybe(consumeRunesLike(func(r rune) bool { return r != '>' && r != '/' && !unicode.IsSpace(r) })).
			Map(recognizeToken(xmlTokenRoleTag)).
			Map(setState(xmlParseStateInTag)))

	parseCharacterEntity := matchState(
		xmlParseStateNormal,
		consumeString("&").
			Then(consumeRunesLike(func(r rune) bool { return r != '<' && r != '>' && r != ';' && !unicode.IsSpace(r) })).
			Then(consumeString(";")).
			Map(recognizeToken(xmlTokenRoleCharacterEntity)))

	parseAttrKey := consumeRunesLike(func(r rune) bool { return r != '>' && r != '=' && !unicode.IsSpace(r) }).
		Then(consumeString("=")).
		Map(recognizeToken(xmlTokenRoleAttrKey))

	consumeAttrValSingleQuote := consumeString("'").
		Then(consumeToEofOrRuneLike(func(r rune) bool { return r == '\'' || r == '\n' || r == '>' }))

	consumeAttrValDoubleQuote := consumeString("\"").
		Then(consumeToEofOrRuneLike(func(r rune) bool { return r == '"' || r == '\n' || r == '>' }))

	parseAttrVal := consumeAttrValSingleQuote.
		Or(consumeAttrValDoubleQuote).
		Map(recognizeToken(parser.TokenRoleString))

	parseTagContent := matchState(
		xmlParseStateInTag,
		parseAttrVal.Or(parseAttrKey))

	parseTagEnd := matchState(
		xmlParseStateInTag,
		consumeLongestMatchingOption([]string{">", "/>"}).
			Map(recognizeToken(xmlTokenRoleTag)).
			Map(setState(xmlParseStateNormal)))

	parseTag := parseTagStart.
		Or(parseTagContent).
		Or(parseTagEnd)

	return initialState(
		xmlParseStateNormal,
		parseComment.
			Or(parsePrologue).
			Or(parseCData).
			Or(parseCharacterEntity).
			Or(parseTag))
}
