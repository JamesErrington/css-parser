package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type Encoding uint8

const (
	UTF_8 Encoding = iota
)

const (
	// https://unicodebook.readthedocs.io/unicode_encodings.html#utf-8
	UTF_8_MULTIBYTE_START_MARKER = 0b11000000
	UTF_8_MULTIBYTE_BODY_MARKER  = 0b10000000
	// https://drafts.csswg.org/css-syntax/#maximum-allowed-code-point
	// The greatest code point defined by Unicode: U+10FFFF.
	MAX_CODE_POINT int64 = 0x10FFFF
)

const (
	EOF_CHAR                   rune = -1
	NULL_CHAR                  rune = '\u0000'
	REPLACEMENT_CHAR           rune = '\uFFFD'
	LINE_FEED_CHAR             rune = '\u000A'
	CARRIAGE_RETURN_CHAR       rune = '\u000D'
	FORM_FEED_CHAR             rune = '\u000C'
	FORWARD_SLASH_CHAR         rune = '\u002F'
	BACKWARD_SLASH_CHAR        rune = '\u005C'
	ASTERISK_CHAR              rune = '\u002A'
	TAB_CHAR                   rune = '\u0009'
	SPACE_CHAR                 rune = '\u0020'
	QUOTATION_MARK_CHAR        rune = '\u0022'
	APOSTROPHE_CHAR            rune = '\u0027'
	NUMBER_SIGN_CHAR           rune = '\u0023'
	CONTROL_CHAR               rune = '\u0080'
	LOW_LINE_CHAR              rune = '\u005F'
	HYPHEN_MINUS_CHAR          rune = '\u002D'
	OPEN_PAREN_CHAR            rune = '\u0028'
	CLOSE_PAREN_CHAR           rune = '\u0029'
	OPEN_SQUARE_CHAR           rune = '\u005B'
	CLOSE_SQUARE_CHAR          rune = '\u005D'
	OPEN_CURLY_CHAR            rune = '\u007B'
	CLOSE_CURLY_CHAR           rune = '\u007D'
	PLUS_SIGN_CHAR             rune = '\u002B'
	FULL_STOP_CHAR             rune = '\u002E'
	LOWER_E_CHAR               rune = '\u0065'
	UPPER_E_CHAR               rune = '\u0045'
	LOWER_U_CHAR               rune = '\u0075'
	UPPER_U_CHAR               rune = '\u0055'
	PERCENT_SIGN_CHAR          rune = '\u0025'
	COMMA_CHAR                 rune = '\u002C'
	GREATER_THAN_CHAR          rune = '\u003E'
	LESS_THAN_CHAR             rune = '\u003C'
	BACKSPACE_CHAR             rune = '\u0008'
	LINE_TABULATION_CHAR       rune = '\u000B'
	SHIFT_OUT_CHAR             rune = '\u000E'
	INFORMATION_SEPARATOR_CHAR rune = '\u001F'
	DELETE_CHAR                rune = '\u007F'
	COLON_CHAR                 rune = '\u003A'
	SEMICOLON_CHAR             rune = '\u003B'
	EXCLAMATON_MARK_CHAR       rune = '\u0021'
	AT_CHAR                    rune = '\u0040'
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		log.Fatal("No file argument provided")
	}

	file, err := os.Open(args[0])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	ParseStylesheet(file)
}

// https://drafts.csswg.org/css-syntax/#input-preprocessing
func preprocess_input_stream(bytes []byte) []rune {
	// @TODO: determine the encoding from the byte stream instead of only supporting UTF-8
	// https://drafts.csswg.org/css-syntax/#input-byte-stream
	runes := decode_byte_stream(bytes, UTF_8)
	length := len(runes)

	// The input stream consists of the filtered code points pushed into it as the input byte stream is decoded.
	// @NOTE: Here, we fully decode the byte stream first, before we filter the code points.
	input := make([]rune, 0, length)
	for i := 0; i < length; i += 1 {
		char := runes[i]

		switch char {
		// Replace any U+000D CARRIAGE RETURN (CR), U+000C FORM FEED (FF) code points,
		// or pairs of U+000D CARRIAGE RETURN (CR) followed by U+000A LINE FEED (LF) in input
		// by a single U+000A LINE FEED (LF) code point.
		case CARRIAGE_RETURN_CHAR:
			char = LINE_FEED_CHAR
			if i < length-1 && runes[i+1] == LINE_FEED_CHAR {
				i += 1
			}
		case FORM_FEED_CHAR:
			char = LINE_FEED_CHAR
		// Replace any U+0000 NULL or surrogate code points in input with U+FFFD REPLACEMENT CHARACTER (�).
		// @NOTE: We don't support utf16 encoding so there can be no surrogate code points
		case NULL_CHAR:
			char = REPLACEMENT_CHAR
		}

		input = append(input, char)
	}

	return input
}

// @NOTE: We currently only decode into UTF-8
func decode_byte_stream(bytes []byte, encoding Encoding) []rune {
	// @TODO: implement other encodings
	if encoding != UTF_8 {
		log.Fatalf("UTF-8 is currently the only supported encoding")
	}

	result := make([]rune, 0, len(bytes))

	for i := 0; i < len(bytes); {
		bite := bytes[i]
		i += 1
		// Unicode characters are up to 4 bytes wide
		code_point := make([]byte, 0, 4)
		code_point = append(code_point, bite)
		// Use the marker bits to detect if this is a multibyte character
		if is_multibyte_start(bite) {
			for ; i < len(bytes); i += 1 {
				next_byte := bytes[i]
				// Keep appending bytes if they are marked as part of the multibyte character
				if is_multibyte_body(next_byte) {
					code_point = append(code_point, next_byte)
				} else {
					break
				}
			}
		}

		char, _ := utf8.DecodeRune(code_point)
		if char == utf8.RuneError {
			log.Fatalf("Invalid UTF-8 encoding: %v", code_point)
		}

		result = append(result, char)
	}

	return result
}

func is_multibyte_start(bite byte) bool {
	return bite&UTF_8_MULTIBYTE_START_MARKER == UTF_8_MULTIBYTE_START_MARKER
}

func is_multibyte_body(bite byte) bool {
	return bite&UTF_8_MULTIBYTE_BODY_MARKER == UTF_8_MULTIBYTE_BODY_MARKER
}

// https://drafts.csswg.org/css-syntax/#newline
func is_newline(char rune) bool {
	// U+000A LINE FEED

	// Note that U+000D CARRIAGE RETURN and U+000C FORM FEED are not included in this definition,
	// as they are converted to U+000A LINE FEED during preprocessing
	return char == LINE_FEED_CHAR
}

// https://drafts.csswg.org/css-syntax/#whitespace
func is_whitespace(char rune) bool {
	// newline, U+0009 CHARACTER TABULATION, or U+0020 SPACE.
	return is_newline(char) || char == TAB_CHAR || char == SPACE_CHAR
}

// https://drafts.csswg.org/css-syntax/#digit
func is_digit(char rune) bool {
	// A code point between U+0030 DIGIT ZERO (0) and U+0039 DIGIT NINE (9) inclusive.
	return char >= '0' && char <= '9'
}

// https://drafts.csswg.org/css-syntax/#hex-digit
func is_hex_digit(char rune) bool {
	// A digit,
	// or a code point between U+0041 LATIN CAPITAL LETTER A (A) and U+0046 LATIN CAPITAL LETTER F (F) inclusive,
	// or a code point between U+0061 LATIN SMALL LETTER A (a) and U+0066 LATIN SMALL LETTER F (f) inclusive.
	return is_digit(char) || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')
}

// https://drafts.csswg.org/css-syntax/#uppercase-letter
func is_uppercase(char rune) bool {
	// A code point between U+0041 LATIN CAPITAL LETTER A (A) and U+005A LATIN CAPITAL LETTER Z (Z) inclusive.
	return char >= 'A' && char <= 'Z'
}

// https://drafts.csswg.org/css-syntax/#lowercase-letter
func is_lowercase(char rune) bool {
	// A code point between U+0061 LATIN SMALL LETTER A (a) and U+007A LATIN SMALL LETTER Z (z) inclusive.
	return char >= 'a' && char <= 'z'
}

// https://drafts.csswg.org/css-syntax/#letter
func is_letter(char rune) bool {
	// An uppercase letter or a lowercase letter.
	return is_uppercase(char) || is_lowercase(char)
}

// https://drafts.csswg.org/css-syntax/#non-ascii-ident-code-point
func is_non_ascii(char rune) bool {
	// A code point whose value is any of:
	switch char {
	case '\u00B7', '\u200C', '\u200D', '\u203F', '\u2040':
		return true
	}

	// Or in any of these ranges (inclusive):
	switch {
	case char >= '\u00C0' && char <= '\u00D6':
		fallthrough
	case char >= '\u00D8' && char <= '\u00F6':
		fallthrough
	case char >= '\u00F8' && char <= '\u037D':
		fallthrough
	case char >= '\u037F' && char <= '\u1FFF':
		fallthrough
	case char >= '\u2070' && char <= '\u218F':
		fallthrough
	case char >= '\u2C00' && char <= '\u2FEF':
		fallthrough
	case char >= '\u3001' && char <= '\uD7FF':
		fallthrough
	case char >= '\uF900' && char <= '\uFDCF':
		fallthrough
	case char >= '\uFDF0' && char <= '\uFFFD':
		fallthrough
	case char >= 0x10000:
		return true
	}

	return false
}

// https://drafts.csswg.org/css-syntax/#ident-start-code-point
func is_ident_start(char rune) bool {
	// A letter, a non-ASCII code point, or U+005F LOW LINE (_).
	return is_letter(char) || is_non_ascii(char) || char == LOW_LINE_CHAR
}

// https://drafts.csswg.org/css-syntax/#ident-code-point
func is_ident(char rune) bool {
	// An ident-start code point, a digit, or U+002D HYPHEN-MINUS (-).
	return is_ident_start(char) || is_digit(char) || char == HYPHEN_MINUS_CHAR
}

// https://drafts.csswg.org/css-syntax/#non-printable-code-point
func is_non_printable(char rune) bool {
	// A code point between U+0000 NULL and U+0008 BACKSPACE inclusive,
	// or U+000B LINE TABULATION,
	// or a code point between U+000E SHIFT OUT and U+001F INFORMATION SEPARATOR ONE inclusive,
	// or U+007F DELETE.
	switch {
	case char >= NULL_CHAR && char <= BACKSPACE_CHAR:
		fallthrough
	case char == LINE_TABULATION_CHAR:
		fallthrough
	case char >= SHIFT_OUT_CHAR && char <= INFORMATION_SEPARATOR_CHAR:
		fallthrough
	case char == DELETE_CHAR:
		return true
	}

	return false
}

// Check if two code points are a valid escape.
// https://drafts.csswg.org/css-syntax/#starts-with-a-valid-escape
func are_valid_escape(first rune, second rune) bool {
	// @ASSERTION: This algorithm will not consume any additional code point.

	// If the first code point is not U+005C REVERSE SOLIDUS (\), return false.
	if first != BACKWARD_SLASH_CHAR {
		return false
	}

	// Otherwise, if the second code point is a newline, return false.
	// Otherwise, return true.
	return is_newline(second) == false
}

// https://drafts.csswg.org/css-syntax/#starts-with-a-valid-escape
func (t *Tokenizer) starts_with_valid_escape() bool {
	// The two code points in question are the current input code point and the next input code point, in that order.
	return are_valid_escape(t.current_rune(), t.next_rune())
}

// Check if three code points would start an ident sequence.
// https://drafts.csswg.org/css-syntax/#would-start-an-identifier
func are_start_ident(first rune, second rune, third rune) bool {
	// @ASSERTION: This algorithm will not consume any additional code points.

	// Look at the first code point
	switch {
	// U+002D HYPHEN-MINUS
	case first == HYPHEN_MINUS_CHAR:
		// If the second code point is an ident-start code point or a U+002D HYPHEN-MINUS,
		// or the second and third code points are a valid escape, return true.
		// Otherwise, return false.
		return is_ident_start(second) || (second == HYPHEN_MINUS_CHAR) || are_valid_escape(second, third)
	// ident-start code point
	case is_ident_start(first):
		return true
	// U+005C REVERSE SOLIDUS (\)
	case first == BACKWARD_SLASH_CHAR:
		// If the first and second code points are a valid escape, return true. Otherwise, return false.
		return are_valid_escape(first, second)
	// anything else
	default:
		return false
	}
}

// https://drafts.csswg.org/css-syntax/#would-start-an-identifier
func (t *Tokenizer) starts_with_ident() bool {
	// The three code points in question are the current input code point and the next two input code points, in that order.
	return are_start_ident(t.current_rune(), t.next_rune(), t.second_rune())
}

// Check if three code points would start a number.
// https://drafts.csswg.org/css-syntax/#starts-with-a-number
func are_number(first rune, second rune, third rune) bool {
	// @ASSERTION: This algorithm will not consume any additional code points.

	// Look at the first code point
	switch {
	// U+002B PLUS SIGN (+), U+002D HYPHEN-MINUS (-)
	case first == PLUS_SIGN_CHAR, first == HYPHEN_MINUS_CHAR:
		// If the second code point is a digit, return true.
		if is_digit(second) {
			return true
		}
		// Otherwise, if the second code point is a U+002E FULL STOP (.) and the third code point is a digit, return true.
		// Otherwise, return false.
		return (second == FULL_STOP_CHAR) && is_digit(third)
	// U+002E FULL STOP (.)
	case first == FULL_STOP_CHAR:
		// If the second code point is a digit, return true. Otherwise, return false.
		return is_digit(second)
	// digit
	case is_digit(first):
		return true
	// anything else
	default:
		return false
	}
}

// https://drafts.csswg.org/css-syntax/#starts-with-a-number
func (t *Tokenizer) starts_with_number() bool {
	// The three code points in question are the current input code point and the next two input code points, in that order.
	return are_number(t.current_rune(), t.next_rune(), t.second_rune())
}

type Tokenizer struct {
	input  []rune
	length int
	index  int
}

func NewTokenizer(input []rune) *Tokenizer {
	return &Tokenizer{
		input:  input,
		length: len(input),
		index:  -1,
	}
}

type TokenKind uint8

// https://drafts.csswg.org/css-syntax/#tokenization
const (
	IDENT_TOKEN TokenKind = iota
	FUNCTION_TOKEN
	AT_KEYWORD_TOKEN
	HASH_TOKEN
	STRING_TOKEN
	BAD_STRING_TOKEN
	URL_TOKEN
	BAD_URL_TOKEN
	DELIM_TOKEN
	NUMBER_TOKEN
	PERCENTAGE_TOKEN
	DIMENSION_TOKEN
	UNICODE_RANGE_TOKEN
	WHITESPACE_TOKEN
	CDO_TOKEN
	CDC_TOKEN
	COLON_TOKEN
	SEMICOLON_TOKEN
	COMMA_TOKEN
	OPEN_SQUARE_TOKEN
	CLOSE_SQUARE_TOKEN
	OPEN_PAREN_TOKEN
	CLOSE_PAREN_TOKEN
	OPEN_CURLY_TOKEN
	CLOSE_CURLY_TOKEN
	EOF_TOKEN
)

func (k TokenKind) String() string {
	switch k {
	case IDENT_TOKEN:
		return "IDENT_TOKEN"
	case FUNCTION_TOKEN:
		return "FUNCTION_TOKEN"
	case AT_KEYWORD_TOKEN:
		return "AT_KEYWORD_TOKEN"
	case HASH_TOKEN:
		return "HASH_TOKEN"
	case STRING_TOKEN:
		return "STRING_TOKEN"
	case BAD_STRING_TOKEN:
		return "BAD_STRING_TOKEN"
	case URL_TOKEN:
		return "URL_TOKEN"
	case BAD_URL_TOKEN:
		return "BAD_URL_TOKEN"
	case DELIM_TOKEN:
		return "DELIM_TOKEN"
	case NUMBER_TOKEN:
		return "NUMBER_TOKEN"
	case PERCENTAGE_TOKEN:
		return "PERCENTAGE_TOKEN"
	case DIMENSION_TOKEN:
		return "DIMENSION_TOKEN"
	case UNICODE_RANGE_TOKEN:
		return "UNICODE_RANGE_TOKEN"
	case WHITESPACE_TOKEN:
		return "WHITESPACE_TOKEN"
	case CDO_TOKEN:
		return "CDO_TOKEN"
	case CDC_TOKEN:
		return "CDC_TOKEN"
	case COLON_TOKEN:
		return "COLON_TOKEN"
	case SEMICOLON_TOKEN:
		return "SEMICOLON_TOKEN"
	case COMMA_TOKEN:
		return "COMMA_TOKEN"
	case OPEN_SQUARE_TOKEN:
		return "OPEN_SQUARE_TOKEN"
	case CLOSE_SQUARE_TOKEN:
		return "CLOSE_SQUARE_TOKEN"
	case OPEN_PAREN_TOKEN:
		return "OPEN_PAREN_TOKEN"
	case CLOSE_PAREN_TOKEN:
		return "CLOSE_PAREN_TOKEN"
	case OPEN_CURLY_TOKEN:
		return "OPEN_CURLY_TOKEN"
	case CLOSE_CURLY_TOKEN:
		return "CLOSE_CURLY_TOKEN"
	case EOF_TOKEN:
		return "EOF_TOKEN"
	}
	return "<UNKNOWN TOKEN>"
}

func mirror(kind TokenKind) TokenKind {
	switch kind {
	case OPEN_SQUARE_TOKEN:
		return CLOSE_SQUARE_TOKEN
	case OPEN_PAREN_TOKEN:
		return CLOSE_PAREN_TOKEN
	case OPEN_CURLY_TOKEN:
		return CLOSE_CURLY_TOKEN
	default:
		log.Panicf("Invalid call to mirror with TokenKind '%d'", kind)
		return EOF_TOKEN
	}
}

type HashFlag uint8

const (
	// The type flag defaults to "unrestricted" if not otherwise set.
	HASH_UNRESTRICTED HashFlag = iota
	HASH_ID
)

type TypeFlag uint8

const (
	// The type flag defaults to "integer" if not otherwise set.
	TYPE_INTEGER TypeFlag = iota
	TYPE_NUMBER
)

// https://drafts.csswg.org/css-syntax/#tokenization
type Token struct {
	kind TokenKind
	// <ident-token>, <function-token>, <at-keyword-token>, <hash-token>, <string-token>, and <url-token> have a value composed of zero or more code points.
	// <delim-token> has a value composed of a single code point.
	value []rune
	// <number-token>, <percentage-token>, and <dimension-token> have a numeric value.
	numeric float64
	// <number-token>, <percentage-token>, and <dimension-token> have an optional sign character set to either "+" or "-" (or nothing).
	sign []rune
	// <hash-token> have a type flag set to either "id" or "unrestricted".
	hash_flag HashFlag
	// <number-token> and <dimension-token> additionally have a type flag set to either "integer" or "number".
	type_flag TypeFlag
	// <dimension-token> additionally have a unit composed of one or more code points.
	unit []rune
	// <unicode-range-token> has a starting and ending code point.
	// It represents an inclusive range of codepoints (including both the start and end).
	// If the ending code point is before the starting code point, it represents an empty range.
	range_start rune
	range_end   rune
}

func (t Token) String() string {
	// @TODO: Redo with string builder
	kind := t.kind

	str := kind.String()

	if kind == IDENT_TOKEN || kind == FUNCTION_TOKEN || kind == AT_KEYWORD_TOKEN || kind == HASH_TOKEN || kind == STRING_TOKEN || kind == URL_TOKEN || kind == DELIM_TOKEN {
		str = str + fmt.Sprintf(" (%s)", string(t.value))
	}

	if kind == NUMBER_TOKEN || kind == PERCENTAGE_TOKEN {
		str = str + fmt.Sprintf(" (%f)", t.numeric)
	}

	if kind == DIMENSION_TOKEN {
		str = str + fmt.Sprintf(" (%f, %s)", t.numeric, string(t.unit))
	}

	return str
}

func (t *Tokenizer) Tokenize() []Token {
	// To tokenize a stream of code points into a stream of CSS tokens input, repeatedly consume a token from input until an <EOF-token> is reached,
	// pushing each of the returned tokens into a stream.
	var tokens []Token
	for token := t.ConsumeToken(); token.kind != EOF_TOKEN; token = t.ConsumeToken() {
		tokens = append(tokens, token)
	}

	return tokens
}

// https://drafts.csswg.org/css-syntax/#consume-token
func (t *Tokenizer) ConsumeToken() Token {
	// Additionally takes an optional boolean unicode ranges allowed, defaulting to false.
	unicode_ranges_allowed := false
	// Consume comments.
	t.consume_comments()
	// Consume the next input code point.
	char := t.consume_next()
	switch {
	case is_whitespace(char):
		// Consume as much whitespace as possible.
		for is_whitespace(t.next_rune()) {
			t.consume_next()
		}
		// Return a <whitespace-token>.
		return Token{kind: WHITESPACE_TOKEN}
	// U+0022 QUOTATION MARK (")
	case char == QUOTATION_MARK_CHAR:
		// Consume a string token and return it.
		return t.consume_string_token_default()
	// U+0023 NUMBER SIGN (#)
	case char == NUMBER_SIGN_CHAR:
		// If the next input code point is an ident code point or the next two input code points are a valid escape
		if is_ident(t.next_rune()) || are_valid_escape(t.next_rune(), t.second_rune()) {
			// 1. Create a <hash-token>.
			token := Token{kind: HASH_TOKEN}
			// 2. If the next 3 input code points would start an ident sequence, set the <hash-token>’s type flag to "id".
			if are_start_ident(t.next_rune(), t.second_rune(), t.third_rune()) {
				token.hash_flag = HASH_ID
			}
			// 3. Consume an ident sequence, and set the <hash-token>’s value to the returned string.
			token.value = t.consume_ident_sequence()
			// 4. Return the <hash-token>.
			return token
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.current_rune()}}
	// U+0027 APOSTROPHE (')
	case char == APOSTROPHE_CHAR:
		// Consume a string token and return it.
		return t.consume_string_token_default()
	// U+0028 LEFT PARENTHESIS (()
	case char == OPEN_PAREN_CHAR:
		return Token{kind: OPEN_PAREN_TOKEN}
	// U+0029 RIGHT PARENTHESIS ())
	case char == CLOSE_PAREN_CHAR:
		return Token{kind: CLOSE_PAREN_TOKEN}
	// U+002B PLUS SIGN (+)
	case char == PLUS_SIGN_CHAR:
		// If the input stream starts with a number, reconsume the current input code point, consume a numeric token, and return it.
		if t.starts_with_number() {
			t.reconsume_current()
			return t.consume_numeric_token()
		}
		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{char}}
	// U+002C COMMA (,)
	case char == COMMA_CHAR:
		return Token{kind: COMMA_TOKEN}
	// U+002D HYPHEN-MINUS (-)
	case char == HYPHEN_MINUS_CHAR:
		// If the input stream starts with a number, reconsume the current input code point, consume a numeric token, and return it.
		if t.starts_with_number() {
			t.reconsume_current()
			return t.consume_numeric_token()
		}

		// Otherwise, if the next 2 input code points are U+002D HYPHEN-MINUS U+003E GREATER-THAN SIGN (->), consume them and return a <CDC-token>.
		if t.next_rune() == HYPHEN_MINUS_CHAR && t.second_rune() == GREATER_THAN_CHAR {
			t.consume_runes(2)
			return Token{kind: CDC_TOKEN}
		}

		// Otherwise, if the input stream starts with an ident sequence, reconsume the current input code point, consume an ident-like token, and return it.
		if t.starts_with_ident() {
			t.reconsume_current()
			return t.consume_ident_like_token()
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.current_rune()}}
	// U+002E FULL STOP (.)
	case char == FULL_STOP_CHAR:
		// If the input stream starts with a number, reconsume the current input code point, consume a numeric token, and return it.
		if t.starts_with_number() {
			t.reconsume_current()
			return t.consume_numeric_token()
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.current_rune()}}
	// U+003A COLON (:)
	case char == COLON_CHAR:
		return Token{kind: COLON_TOKEN}
	// U+003B SEMICOLON (;)
	case char == SEMICOLON_CHAR:
		return Token{kind: SEMICOLON_TOKEN}
	// U+003C LESS-THAN SIGN (<)
	case char == LESS_THAN_CHAR:
		// If the next 3 input code points are U+0021 EXCLAMATION MARK U+002D HYPHEN-MINUS U+002D HYPHEN-MINUS (!--), consume them and return a <CDO-token>.
		first, second, third := t.next_rune(), t.second_rune(), t.third_rune()
		if first == EXCLAMATON_MARK_CHAR && second == HYPHEN_MINUS_CHAR && third == HYPHEN_MINUS_CHAR {
			t.consume_runes(3)
			return Token{kind: CDO_TOKEN}
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.current_rune()}}
	// U+0040 COMMERCIAL AT (@)
	case char == AT_CHAR:
		// If the next 3 input code points would start an ident sequence
		if are_start_ident(t.next_rune(), t.second_rune(), t.third_rune()) {
			// Consume an ident sequence, create an <at-keyword-token> with its value set to the returned value, and return it.
			return Token{kind: AT_KEYWORD_TOKEN, value: t.consume_ident_sequence()}
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.current_rune()}}
	// U+005B LEFT SQUARE BRACKET ([)
	case char == OPEN_SQUARE_CHAR:
		return Token{kind: OPEN_SQUARE_TOKEN}
	// U+005C REVERSE SOLIDUS (\)
	case char == BACKWARD_SLASH_CHAR:
		// If the input stream starts with a valid escape, reconsume the current input code point, consume an ident-like token, and return it.
		if t.starts_with_valid_escape() {
			t.reconsume_current()
			return t.consume_ident_like_token()
		}

		// Otherwise, this is a parse error. Return a <delim-token> with its value set to the current input code point.
		fmt.Println("Parse Error: Encountered invalid escape")
		return Token{kind: DELIM_TOKEN, value: []rune{t.current_rune()}}
	// U+005D RIGHT SQUARE BRACKET (])
	case char == CLOSE_SQUARE_CHAR:
		return Token{kind: CLOSE_SQUARE_TOKEN}
	// U+007B LEFT CURLY BRACKET ({)
	case char == OPEN_CURLY_CHAR:
		return Token{kind: OPEN_CURLY_TOKEN}
	// U+007D RIGHT CURLY BRACKET (})
	case char == CLOSE_CURLY_CHAR:
		return Token{kind: CLOSE_CURLY_TOKEN}
	// digit
	case is_digit(char):
		// Reconsume the current input code point, consume a numeric token, and return it.
		t.reconsume_current()
		return t.consume_numeric_token()
	// U+0055 LATIN CAPITAL LETTER U (U), U+0075 LATIN LOWERCASE LETTER U (u)
	case char == UPPER_U_CHAR, char == LOWER_U_CHAR:
		// If unicode ranges allowed is true and the input stream would start a unicode-range,
		// reconsume the current input code point, consume a unicode-range token, and return it.
		// @TODO: implement unicode ranges
		if unicode_ranges_allowed {
			log.Fatal("Parse Error: Unicode Ranges are currently not supported")
		}
		// Otherwise, reconsume the current input code point, consume an ident-like token, and return it.
		t.reconsume_current()
		return t.consume_ident_like_token()
	// ident-start code point
	case is_ident_start(char):
		// Reconsume the current input code point, consume an ident-like token, and return it.
		t.reconsume_current()
		return t.consume_ident_like_token()
	// EOF
	case char == EOF_CHAR:
		return Token{kind: EOF_TOKEN}
	// anything else
	default:
		// Return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.current_rune()}}
	}
}

func (t *Tokenizer) peek_rune(number int) rune {
	index := t.index + number
	if index < 0 || index >= t.length {
		return EOF_CHAR
	}

	return t.input[index]
}

// The last code point to have been consumed.
// https://drafts.csswg.org/css-syntax/#current-input-code-point
func (t *Tokenizer) current_rune() rune {
	return t.peek_rune(0)
}

// The first code point in the input stream that has not yet been consumed.
// https://drafts.csswg.org/css-syntax/#next-input-code-point
func (t *Tokenizer) next_rune() rune {
	return t.peek_rune(1)
}

// The code point in the input stream immediately after the next rune.
func (t *Tokenizer) second_rune() rune {
	return t.peek_rune(2)
}

// The code point in the input stream immediately after the second rune.
func (t *Tokenizer) third_rune() rune {
	return t.peek_rune(3)
}

func (t *Tokenizer) consume_runes(number int) rune {
	t.index += number
	return t.current_rune()
}

func (t *Tokenizer) consume_next() rune {
	return t.consume_runes(1)
}

// Push the current input code point back onto the front of the input stream, so that the next time you are instructed to consume the next input code point,
// it will instead reconsume the current input code point.
// https://drafts.csswg.org/css-syntax/#reconsume-the-current-input-code-point
func (t *Tokenizer) reconsume_current() {
	t.consume_runes(-1)
}

// https://drafts.csswg.org/css-syntax/#consume-comment
func (t *Tokenizer) consume_comments() {
	// If the next two input code point are U+002F SOLIDUS (/) followed by a U+002A ASTERISK (*)
	for t.next_rune() == FORWARD_SLASH_CHAR && t.second_rune() == ASTERISK_CHAR {
		// consume them
		t.consume_runes(2)

		for {
			// and all following code points up to and including
			char := t.consume_next()
			// the first U+002A ASTERISK (*) followed by a U+002F SOLIDUS (/)
			if char == ASTERISK_CHAR && t.next_rune() == FORWARD_SLASH_CHAR {
				t.consume_runes(2)
				break
			} else if char == EOF_CHAR { // or up to an EOF code point
				// If the preceding paragraph ended by consuming an EOF code point, this is a parse error
				log.Fatal("Parse Error: Encountered unexpected EOF when parsing comment")
			}
		}
	}
}

// This algorithm may be called with an `ending` code point, which denotes the code point that ends the string.
// https://drafts.csswg.org/css-syntax/#consume-string-token
func (t *Tokenizer) consume_string_token(ending rune) Token {
	// Returns either a <string-token> or <bad-string-token>.

	// Initially create a <string-token> with its value set to the empty string.
	token := Token{kind: STRING_TOKEN}

	// Repeatedly consume the next input code point from the stream:
	for {
		char := t.consume_next()
		switch {
		// ending code point:
		case char == ending:
			// Return the <string-token>.
			return token
		// EOF
		case char == EOF_CHAR:
			// This is a parse error. Return the <string-token>.
			fmt.Println("Parse Error: Encountered unexpected EOF when parsing string")
			return token
		// newline:
		case is_newline(char):
			// This is a parse error. Reconsume the current input code point, create a <bad-string-token>, and return it.
			fmt.Println("Parse Error: Encountered unexpected newline when parsing string")
			t.reconsume_current()
			return Token{kind: BAD_STRING_TOKEN}
		// U+005C REVERSE SOLIDUS (\):
		case char == BACKWARD_SLASH_CHAR:
			// If the next input code point is EOF, do nothing.
			next := t.next_rune()
			if next != EOF_CHAR {
				// Otherwise, if the next input code point is a newline, consume it.
				if is_newline(next) {
					t.consume_next()
				} else {
					// Otherwise, consume an escaped code point and append the returned code point to the <string-token>’s value.
					// @ASSERTED: the stream starts with a valid escape
					token.value = append(token.value, t.consume_escaped())
				}
			}
		// anything else: Append the current input code point to the <string-token>’s value.
		default:
			token.value = append(token.value, char)
		}
	}
}

// https://drafts.csswg.org/css-syntax/#consume-string-token
func (t *Tokenizer) consume_string_token_default() Token {
	// If an ending code point is not specified, the current input code point is used.
	return t.consume_string_token(t.current_rune())
}

// https://drafts.csswg.org/css-syntax/#consume-escaped-code-point
func (t *Tokenizer) consume_escaped() rune {
	// Assume that the U+005C REVERSE SOLIDUS (\) has already been consumed
	// and that the next input code point has already been verified to be part of a valid escape.
	// Returns a code point.

	// Consume the next input code point.
	char := t.consume_next()
	switch {
	// hex digit
	case is_hex_digit(char):
		// Consume as many hex digits as possible, but no more than 5.
		// @Assertion: Note that this means 1-6 hex digits have been consumed in total.
		digits := make([]rune, 0, 6)
		digits = append(digits, char)

		for i := 0; i < 5; i += 1 {
			if is_hex_digit(t.next_rune()) {
				digits = append(digits, t.consume_next())
			} else {
				break
			}
		}

		// If the next input code point is whitespace, consume it as well.
		if is_whitespace(t.next_rune()) {
			t.consume_next()
		}

		// Interpret the hex digits as a hexadecimal number.
		value, err := strconv.ParseInt(string(digits[:]), 16, 64)
		if err != nil {
			log.Fatal(err)
		}

		// If this number is zero, or is for a surrogate, or is greater than the maximum allowed code point,
		// return U+FFFD REPLACEMENT CHARACTER (�).
		if value == 0 || utf16.IsSurrogate(rune(value)) || value > MAX_CODE_POINT {
			return REPLACEMENT_CHAR
		}

		// Otherwise, return the code point with that value.
		return rune(value)
	// EOF
	case char == EOF_CHAR:
		// This is a parse error. Return U+FFFD REPLACEMENT CHARACTER (�).
		fmt.Println("Parse Error: Encountered unexpected EOF when parsing escape")
		return REPLACEMENT_CHAR
	// anything else
	default:
		// Return the current input code point.
		return t.current_rune()
	}
}

// https://drafts.csswg.org/css-syntax/#consume-name
func (t *Tokenizer) consume_ident_sequence() []rune {
	// Returns a string containing the largest name that can be formed from adjacent code points in the stream, starting from the first.

	// Let result initially be an empty string.
	var result []rune
	// Repeatedly consume the next input code point from the stream:
	for {
		char := t.consume_next()
		switch {
		// ident code point
		case is_ident(char):
			// Append the code point to result.
			result = append(result, char)
		// the stream starts with a valid escape
		case t.starts_with_valid_escape():
			// Consume an escaped code point. Append the returned code point to result.
			result = append(result, t.consume_escaped())
		// anything else
		default:
			// Reconsume the current input code point. Return result.
			t.reconsume_current()
			return result
		}
	}
}

// https://drafts.csswg.org/css-syntax/#consume-numeric-token
func (t *Tokenizer) consume_numeric_token() Token {
	// Returns either a <number-token>, <percentage-token>, or <dimension-token>.

	// Consume a number and let number be the result.
	number, type_flag, sign := t.consume_number()
	// If the next 3 input code points would start an ident sequence
	if are_start_ident(t.next_rune(), t.second_rune(), t.third_rune()) {
		// 1. Create a <dimension-token> with the same value, type flag, and sign character as number, and a unit set initially to the empty string.
		token := Token{kind: DIMENSION_TOKEN, numeric: number, type_flag: type_flag, sign: sign}
		// 2. Consume an ident sequence. Set the <dimension-token>’s unit to the returned value.
		token.unit = t.consume_ident_sequence()
		// 3. Return the <dimension-token>.
		return token
	}
	// Otherwise, if the next input code point is U+0025 PERCENTAGE SIGN (%), consume it.
	if t.next_rune() == PERCENT_SIGN_CHAR {
		t.consume_next()
		// Create a <percentage-token> with the same value as number, and return it.
		return Token{kind: PERCENTAGE_TOKEN, numeric: number}
	}

	// Otherwise, create a <number-token> with the same value, type flag, and sign character as number, and return it.
	return Token{kind: NUMBER_TOKEN, numeric: number, type_flag: type_flag, sign: sign}
}

// https://drafts.csswg.org/css-syntax/#consume-number
func (t *Tokenizer) consume_number() (float64, TypeFlag, []rune) {
	// Returns a numeric value, a string type which is either "integer" or "number",
	// and an optional sign character which is either "+", "-", or missing.

	// 1. Let type be the string "integer". Let number part and exponent part be the empty string.
	type_flag := TYPE_INTEGER
	var number_part []rune
	var exponent_part []rune
	var sign []rune
	// 2. If the next input code point is U+002B PLUS SIGN (+) or U+002D HYPHEN-MINUS (-), consume it.
	next := t.next_rune()
	if next == PLUS_SIGN_CHAR || next == HYPHEN_MINUS_CHAR {
		t.consume_next()
		// Append it to number part and set sign character to it.
		number_part = append(number_part, next)
		sign = []rune{next}
	}

	// 3. While the next input code point is a digit, consume it and append it to number part.
	for is_digit(t.next_rune()) {
		number_part = append(number_part, t.consume_next())
	}

	// 4. If the next 2 input code points are U+002E FULL STOP (.) followed by a digit
	if t.next_rune() == FULL_STOP_CHAR && is_digit(t.second_rune()) {
		// 4.1 Consume the next input code point and append it to number part.
		number_part = append(number_part, t.consume_next())
		// 4.2 While the next input code point is a digit, consume it and append it to number part.
		for is_digit(t.next_rune()) {
			number_part = append(number_part, t.consume_next())
		}
		// 4.3 Set type to "number".
		type_flag = TYPE_NUMBER
	}

	// 5. If the next 2 or 3 input code points are U+0045 LATIN CAPITAL LETTER E (E) or U+0065 LATIN SMALL LETTER E (e),
	//    optionally followed by U+002D HYPHEN-MINUS (-) or U+002B PLUS SIGN (+),
	//    followed by a digit
	first, second, third := t.next_rune(), t.second_rune(), t.third_rune()
	lookahead := 2
	if first == UPPER_E_CHAR || first == LOWER_E_CHAR {
		if second == HYPHEN_MINUS_CHAR || second == PLUS_SIGN_CHAR {
			lookahead = 3
		}

		if (lookahead == 2 && is_digit(second)) || (lookahead == 3 && is_digit(third)) {
			// 5.1 Consume the next input code point.
			t.consume_next()
			// 5.2 If the next input code point is "+" or "-", consume it and append it to exponent part.
			switch t.next_rune() {
			case PLUS_SIGN_CHAR, HYPHEN_MINUS_CHAR:
				exponent_part = append(exponent_part, t.consume_next())
			}
			// 5.3 While the next input code point is a digit, consume it and append it to exponent part.
			for is_digit(t.next_rune()) {
				exponent_part = append(exponent_part, t.consume_next())
			}
			// 5.4 Set type to "number".
			type_flag = TYPE_NUMBER
		}
	}

	// 6. Let number value be the result of interpreting number part as a base-10 number.
	value, err := strconv.ParseFloat(string(number_part), 10)
	if err != nil {
		log.Fatal(err)
	}

	// If exponent part is non-empty, interpret it as a base-10 integer
	if len(exponent_part) > 0 {
		exponent_value, err := strconv.ParseInt(string(exponent_part), 10, 0)
		if err != nil {
			log.Fatal(err)
		}
		// Then raise 10 to the power of the result, multiply it by number value, and set value to that result.
		value = value * math.Pow10(int(exponent_value))
	}

	// 7. Return value, type, and sign character.
	return value, type_flag, sign
}

// https://drafts.csswg.org/css-syntax/#consume-ident-like-token
func (t *Tokenizer) consume_ident_like_token() Token {
	// Returns an <ident-token>, <function-token>, <url-token>, or <bad-url-token>.

	// Consume an ident sequence, and let string be the result.
	str := t.consume_ident_sequence()
	// If string’s value is an ASCII case-insensitive match for "url", and the next input code point is U+0028 LEFT PARENTHESIS ((), consume it.
	if strings.EqualFold(string(str), "url") && t.next_rune() == OPEN_PAREN_CHAR {
		t.consume_next()
		// While the next two input code points are whitespace, consume the next input code point.
		for is_whitespace(t.next_rune()) && is_whitespace(t.second_rune()) {
			t.consume_next()
		}
		// If the next one or two input code points are U+0022 QUOTATION MARK ("), U+0027 APOSTROPHE ('),
		// or whitespace followed by U+0022 QUOTATION MARK (") or U+0027 APOSTROPHE (')
		first, second := t.next_rune(), t.second_rune()
		switch {
		case first == QUOTATION_MARK_CHAR:
			fallthrough
		case first == APOSTROPHE_CHAR:
			fallthrough
		case is_whitespace(first) && second == QUOTATION_MARK_CHAR:
			fallthrough
		case is_whitespace(first) && second == APOSTROPHE_CHAR:
			// create a <function-token> with its value set to string and return it.
			return Token{kind: FUNCTION_TOKEN, value: str}
		// Otherwise, consume a url token, and return it.
		default:
			return t.consume_url_token()
		}
	}

	// Otherwise, if the next input code point is U+0028 LEFT PARENTHESIS ((), consume it.
	if t.next_rune() == OPEN_PAREN_CHAR {
		t.consume_next()
		// Create a <function-token> with its value set to string and return it.
		return Token{kind: FUNCTION_TOKEN, value: str}
	}

	// Otherwise, create an <ident-token> with its value set to string and return it.
	return Token{kind: IDENT_TOKEN, value: str}
}

// https://drafts.csswg.org/css-syntax/#consume-url-token
func (t *Tokenizer) consume_url_token() Token {
	// Returns either a <url-token> or a <bad-url-token>.

	// 1. Initially create a <url-token> with its value set to the empty string.
	token := Token{kind: URL_TOKEN}
	// 2. Consume as much whitespace as possible.
	for is_whitespace(t.next_rune()) {
		t.consume_next()
	}
	// 3. Repeatedly consume the next input code point from the stream:
	for {
		char := t.consume_next()
		switch {
		// U+0029 RIGHT PARENTHESIS ())
		case char == CLOSE_PAREN_CHAR:
			// Return the <url-token>.
			return token
		case char == EOF_CHAR:
			// This is a parse error. Return the <url-token>.
			fmt.Println("Parse Error: Encountered unexpected EOF while parsing URL")
			return token
		case is_whitespace(char):
			// Consume as much whitespace as possible.
			for is_whitespace(t.next_rune()) {
				t.consume_next()
			}
			// If the next input code point is U+0029 RIGHT PARENTHESIS ())
			// or EOF, consume it and return the <url-token> (if EOF was encountered, this is a parse error);
			switch t.next_rune() {
			case CLOSE_PAREN_CHAR:
				t.consume_next()
				return token
			case EOF_CHAR:
				fmt.Println("Parse Error: Encountered unexpected EOF while parsing URL")
				t.consume_next()
				return token
			// otherwise, consume the remnants of a bad url, create a <bad-url-token>, and return it.
			default:
				t.consume_bad_url_remnants()
				return Token{kind: BAD_URL_TOKEN}
			}
		// U+0022 QUOTATION MARK ("), U+0027 APOSTROPHE ('), U+0028 LEFT PARENTHESIS ((), non-printable code point
		case char == QUOTATION_MARK_CHAR, char == APOSTROPHE_CHAR, char == OPEN_PAREN_CHAR, is_non_printable(char):
			// This is a parse error. Consume the remnants of a bad url, create a <bad-url-token>, and return it.
			fmt.Printf("Parse Error: Encountered unexpected character '%x' while parsing URL\n", char)
			t.consume_bad_url_remnants()
			return Token{kind: BAD_URL_TOKEN}
		// U+005C REVERSE SOLIDUS (\)
		case char == BACKWARD_SLASH_CHAR:
			// If the stream starts with a valid escape, consume an escaped code point and append the returned code point to the <url-token>’s value.
			if t.starts_with_valid_escape() {
				token.value = append(token.value, t.consume_escaped())
			} else {
				// Otherwise, this is a parse error. Consume the remnants of a bad url, create a <bad-url-token>, and return it.
				fmt.Println("Parse Error: Encountered invalid escape while parsing URL")
				t.consume_bad_url_remnants()
				return Token{kind: BAD_URL_TOKEN}
			}
		// anything else
		default:
			// Append the current input code point to the <url-token>’s value.
			token.value = append(token.value, t.current_rune())
		}
	}
}

// Consume the remnants of a bad url from a stream of code points,
// "cleaning up" after the tokenizer realizes that it’s in the middle of a <bad-url-token> rather than a <url-token>.
// https://drafts.csswg.org/css-syntax/#consume-remnants-of-bad-url
func (t *Tokenizer) consume_bad_url_remnants() {
	// Returns nothing; its sole use is to consume enough of the input stream to reach a recovery point where normal tokenizing can resume.

	// Repeatedly consume the next input code point from the stream:
	for {
		char := t.consume_next()
		switch {
		// U+0029 RIGHT PARENTHESIS ()), EOF
		case char == CLOSE_PAREN_CHAR, char == EOF_CHAR:
			return
		// the input stream starts with a valid escape
		case t.starts_with_valid_escape():
			// Consume an escaped code point.
			// @NOTE: This allows an escaped right parenthesis ("\)") to be encountered without ending the <bad-url-token>.
			//        This is otherwise identical to the "anything else" clause.
			t.consume_escaped()
		// anything else
		default:
			// Do nothing
		}
	}
}

// https://infra.spec.whatwg.org/#stacks
type Stack[T any] struct {
	items []T
}

func NewStack[T any]() Stack[T] {
	return Stack[T]{items: nil}
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

func (s *Stack[T]) Push(elem T) {
	s.items = append(s.items, elem)
}

func (s *Stack[T]) Pop() (T, bool) {
	// To pop from a stack: if the stack is not empty, then remove its last item and return it; otherwise, return nothing.
	var elem T
	if s.IsEmpty() {
		return elem, false
	}

	elem, s.items = s.items[len(s.items)-1], s.items[:len(s.items)-1]
	return elem, true
}

// https://drafts.csswg.org/css-syntax/#parser-definitions
// A token stream is a struct representing a stream of tokens and/or component values.
type TokenStream struct {
	// A list of tokens and/or component values.
	// @NOTE: The specification assumes, for simplicity, that the input stream has been fully tokenized before parsing begins.
	//		  However, the parsing algorithms only use one token of "lookahead", so in practice tokenization and parsing can be done in lockstep.
	tokens []Token
	length int
	// An index into the tokens, representing the progress of parsing. It starts at 0 initially.
	// @NOTE: Aside from marking, the index never goes backwards. Thus the already-processed prefix of tokens can be eagerly discarded as it’s processed.
	index int
	// A stack of index values, representing points that the parser might return to. It starts empty initially.
	marked_indexes Stack[int]
}

func NewTokenStream(tokens []Token) TokenStream {
	return TokenStream{
		tokens:         tokens,
		length:         len(tokens),
		index:          0,
		marked_indexes: NewStack[int](),
	}
}

// https://drafts.csswg.org/css-syntax/#token-stream-next-token
func (ts *TokenStream) next_token() Token {
	// The item of tokens at index.
	// If that index would be out-of-bounds past the end of the list, it’s instead an <eof-token>.
	if ts.index >= ts.length {
		return Token{kind: EOF_TOKEN}
	}

	return ts.tokens[ts.index]
}

// https://drafts.csswg.org/css-syntax/#token-stream-empty
func (ts *TokenStream) empty() bool {
	// A token stream is empty if the next token is an <eof-token>.
	return ts.next_token().kind == EOF_TOKEN
}

// https://drafts.csswg.org/css-syntax/#token-stream-consume-a-token
func (ts *TokenStream) consume_token() Token {
	// Let token be the next token. Increment index, then return token.
	token := ts.next_token()
	ts.index += 1
	return token
}

// https://drafts.csswg.org/css-syntax/#token-stream-discard-a-token
func (ts *TokenStream) discard_token() {
	// If the token stream is not empty, increment index.
	if ts.empty() == false {
		ts.index += 1
	}
}

// https://drafts.csswg.org/css-syntax/#token-stream-mark
func (ts *TokenStream) mark() {
	// Append index to marked indexes.
	ts.marked_indexes.Push(ts.index)
}

// https://drafts.csswg.org/css-syntax/#token-stream-restore-a-mark
func (ts *TokenStream) restore_mark() {
	// Pop from marked indexes, and set index to the popped value.
	value, ok := ts.marked_indexes.Pop()
	if ok {
		ts.index = value
	}
}

// https://drafts.csswg.org/css-syntax/#token-stream-discard-a-mark
func (ts *TokenStream) discard_mark() {
	// Pop from marked indexes, and do nothing with the popped value.
	ts.marked_indexes.Pop()
}

// https://drafts.csswg.org/css-syntax/#token-stream-discard-whitespace
func (ts *TokenStream) discard_whitespace() {
	// While the next token is a <whitespace-token>, discard a token.
	for ts.next_token().kind == WHITESPACE_TOKEN {
		ts.discard_token()
	}
}

func ParseStylesheet(input io.Reader) {
	bytes, err := io.ReadAll(input)
	if err != nil {
		log.Fatal(err)
	}

	code_points := preprocess_input_stream(bytes)
	tokens := NewTokenizer(code_points).Tokenize()

	token_stream := NewTokenStream(tokens)
	rules := token_stream.consume_stylesheet_contents()

	fmt.Println("Rules:")
	for i, rule := range rules {
		fmt.Printf("\t%d: %s\n", i+1, rule)
	}
}

// https://drafts.csswg.org/css-syntax/#css-stylesheet
type Stylesheet struct {
	rules []Rule
}

// https://drafts.csswg.org/css-syntax/#css-rule
type Rule struct {
	kind RuleKind
	// <at-rule>
	name string
	// <at-rule>, <qualified_rule>
	prelude []ComponentValue
	// <at-rule>, <qualified_rule>
	decls    []Declaration
	children []Rule
}

type RuleKind uint8

// A rule is either an at-rule or a qualified rule.
const (
	AT_RULE RuleKind = iota
	QUALIFIED_RULE
)

func (k RuleKind) String() string {
	switch k {
	case AT_RULE:
		return "AT_RULE"
	case QUALIFIED_RULE:
		return "QUALIFIED_RULE"
	}
	return "<UNKNOWN RULE>"
}

func (r Rule) String() string {
	var sb strings.Builder
	sb.WriteString(r.kind.String())
	if r.kind == AT_RULE {
		sb.WriteString(fmt.Sprintf(" '%s'", r.name))
	}

	sb.WriteString(" PRELUDE: [")
	for _, elem := range r.prelude {
		sb.WriteString(fmt.Sprintf(" %s", elem.String()))
	}
	sb.WriteString(" ] DECLS: [")
	for _, decl := range r.decls {
		sb.WriteString(fmt.Sprintf(" %s", decl.String()))
	}
	sb.WriteString(" ]")

	return sb.String()
}

func (r Rule) is_valid() bool {
	// @TODO: implement this properly.
	// e.g. could use https://github.com/tabatkins/parse-css/blob/0c4d5540274a9e5bcf599732a13ff7ec581264f9/parse-css.js#L1128 as reference
	return true
}

func looks_like_custom_property(prelude []ComponentValue) bool {
	// Return true if the first two non-<whitespace-token> values of rule’s prelude are
	// an <ident-token> whose value starts with "--" followed by a <colon-token>
	i := 0
	for ; i < len(prelude); i += 1 {
		if prelude[i].token.kind != WHITESPACE_TOKEN {
			break
		}
	}

	if i+1 >= len(prelude) {
		return false
	}

	first := prelude[i].token
	if first.kind == IDENT_TOKEN && strings.HasPrefix(string(first.value), "--") {
		return prelude[i+1].token.kind == COLON_TOKEN
	}

	return false
}

// https://drafts.csswg.org/css-syntax/#component-value
type ComponentValue struct {
	kind ComponentValueKind
	// <preserved-token>, <simple-block>
	token Token
	// <function>
	name string
	// <function>, <simple-block>
	value []ComponentValue
}

type ComponentValueKind uint8

// A component value is one of the preserved tokens, a function, or a simple block.
const (
	PRESERVED_TOKEN ComponentValueKind = iota
	FUNCTION
	SIMPLE_BLOCK
)

func (k ComponentValueKind) String() string {
	switch k {
	case PRESERVED_TOKEN:
		return "PRESERVED_TOKEN"
	case FUNCTION:
		return "FUNCTION"
	case SIMPLE_BLOCK:
		return "SIMPLE_BLOCK"
	}
	return "<UNKNOWN COMPONENT VALUE>"
}

func (v ComponentValue) String() string {
	var sb strings.Builder

	switch v.kind {
	case PRESERVED_TOKEN:
		sb.WriteString(fmt.Sprintf("(%s)", v.token.String()))
	case FUNCTION:
		sb.WriteString(fmt.Sprintf("FUNCTION '%s'", v.name))
	case SIMPLE_BLOCK:
		sb.WriteString(fmt.Sprintf("(%s)", v.token.kind.String()))
	}

	return sb.String()
}

// https://drafts.csswg.org/css-syntax/#declaration
type Declaration struct {
	name          string
	value         []ComponentValue
	important     bool
	original_text string
}

func (d Declaration) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s (", d.name))
	for _, elem := range d.value {
		sb.WriteString(fmt.Sprintf(" %s", elem))
	}
	sb.WriteString(" ]")

	if d.important {
		sb.WriteString(fmt.Sprintf(" !important"))
	}

	return sb.String()
}

func (d Declaration) is_valid() bool {
	// @TODO: implement this properly.
	return true
}

func is_custom_property_name(name string) bool {
	// @TODO: implement
	return false
}

func is_unicode_range(name string) bool {
	// @TODO: implement
	return false
}

func ends_with_important(list []ComponentValue) bool {
	// Return true if the last two non-<whitespace-token>s are a <delim-token> with the value "!"
	// followed by an <ident-token> with a value that is an ASCII case-insensitive match for "important"

	// If we have less than 2 tokens, we can't be !important
	if len(list) < 2 {
		return false
	}
	// Iterate backwards through the list until we find the first non-whitespace token
	i := len(list) - 1
	for ; i > 0; i -= 1 {
		if list[i].token.kind != WHITESPACE_TOKEN {
			break
		}
	}
	// Check that we have another token behind us
	if i < 1 {
		return false
	}
	// Check the two tokens for the match
	second_last, last := list[i-1], list[i]
	if second_last.token.kind == DELIM_TOKEN && string(second_last.token.value) == "!" {
		return last.token.kind == IDENT_TOKEN && strings.EqualFold(string(last.token.value), "important")
	}

	return false
}

func contains_non_empty_block(list []ComponentValue) bool {
	// @TODO: implement
	return false
}

// https://drafts.csswg.org/css-syntax/#consume-stylesheet-contents
func (ts *TokenStream) consume_stylesheet_contents() []Rule {
	// Let rules be an initially empty list of rules.
	var rules []Rule

	for {
		switch next := ts.next_token(); next.kind {
		// <whitespace-token>
		case WHITESPACE_TOKEN:
			ts.discard_token()
		// <EOF-token>
		case EOF_TOKEN:
			return rules
		// <CDO-token>, <CDC-token>
		case CDO_TOKEN, CDC_TOKEN:
			ts.discard_token()
		// <at-keyword-token>
		case AT_KEYWORD_TOKEN:
			// Consume an at-rule from input. If anything is returned, append it to rules.
			rule, ok := ts.consume_at_rule_default()
			if ok {
				rules = append(rules, rule)
			}
		// anything else
		default:
			// Consume a qualified rule from input. If anything is returned, append it to rules.
			rule, ok := ts.consume_qualified_rule_default()
			if ok {
				rules = append(rules, rule)
			}
		}
	}
}

// https://drafts.csswg.org/css-syntax/#consume-at-rule
func (ts *TokenStream) consume_at_rule(nested bool) (Rule, bool) {
	// Assert the next token is an <at-keyword> token
	if ts.next_token().kind != AT_KEYWORD_TOKEN {
		panic("Attempted to consume at-rule with invalid token stream state")
	}

	// Consume a token from input, and let rule be a new at-rule with its name set to the returned token’s value,
	// its prelude initially set to an empty list, and no declarations or child rules.
	token := ts.consume_token()
	rule := Rule{kind: AT_RULE, name: string(token.value)}

	for {
		switch next := ts.next_token(); next.kind {
		// <semicolon-token>, <EOF-token>
		case SEMICOLON_TOKEN, EOF_TOKEN:
			// Discard a token from input. If rule is valid in the current context, return it; otherwise return nothing.
			ts.discard_token()
			return rule, rule.is_valid()
		// <}-token>
		case CLOSE_CURLY_TOKEN:
			// If nested is true
			if nested {
				// If rule is valid in the current context, return it; otherwise, return nothing.
				return rule, rule.is_valid()
			}
			// Otherwise, consume a token and append the result to rule’s prelude.
			component := ComponentValue{kind: PRESERVED_TOKEN, token: ts.consume_token()}
			rule.prelude = append(rule.prelude, component)
		// <{-token>
		case OPEN_CURLY_TOKEN:
			// Consume a block from input, and assign the results to rule’s lists of declarations and child rules.
			decls, children := ts.consume_block()
			rule.decls = decls
			rule.children = children
			// If rule is valid in the current context, return it. Otherwise, return nothing.
			return rule, rule.is_valid()
		// anything else
		default:
			// Consume a component value from input and append the returned value to rule’s prelude.
			component := ts.consume_component_value()
			rule.prelude = append(rule.prelude, component)
		}
	}
}

// https://drafts.csswg.org/css-syntax/#consume-at-rule
func (ts *TokenStream) consume_at_rule_default() (Rule, bool) {
	return ts.consume_at_rule(false)
}

// https://drafts.csswg.org/css-syntax/#consume-qualified-rule
func (ts *TokenStream) consume_qualified_rule(nested bool, params ...TokenKind) (Rule, bool) {
	// @NOTE: needed to get round Go's lack of function overloading or default parameters
	stop_token := extract_stop_token(params)
	// Let rule be a new qualified rule with its prelude, declarations, and child rules all initially set to empty lists.
	rule := Rule{kind: QUALIFIED_RULE}

	for {
		switch next := ts.next_token(); next.kind {
		// <EOF-token>
		case EOF_TOKEN, stop_token:
			// This is a parse error. Return nothing.
			fmt.Println("Parse Error: Encountered unexpected EOF or stop token while parsing qualified rule")
			return rule, false
		// <}-token>
		case CLOSE_CURLY_TOKEN:
			// This is a parse error. If nested is true, return nothing. Otherwise, consume a token and append the result to rule’s prelude.
			fmt.Println("Parse Error: Encountered unexpected '}' while parsing qualified rule")
			if nested {
				return rule, false
			}

			component := ComponentValue{kind: PRESERVED_TOKEN, token: ts.consume_token()}
			rule.prelude = append(rule.prelude, component)
		// <{-token>
		case OPEN_CURLY_TOKEN:
			// If the first two non-<whitespace-token> values of rule’s prelude are an <ident-token> whose value starts with "--"
			// followed by a <colon-token>
			if looks_like_custom_property(rule.prelude) {
				// If nested is true, consume the remnants of a bad declaration from input, with nested set to true, and return nothing.
				if nested {
					ts.consume_bad_declaration_remnants(true)
					return rule, false
				}

				// If nested is false, consume a block from input, and return nothing.
				ts.consume_block()
				return rule, false
			} else {
				// Otherwise, consume a block from input, and assign the results to rule’s lists of declarations and child rules.
				decls, rules := ts.consume_block()
				rule.decls = decls
				rule.children = rules
				// If rule is valid in the current context, return it; otherwise return nothing.
				return rule, rule.is_valid()
			}
		// anything else
		default:
			// Consume a component value from input and append the result to rule’s prelude.
			rule.prelude = append(rule.prelude, ts.consume_component_value())
		}
	}
}

func (ts *TokenStream) consume_qualified_rule_default(params ...TokenKind) (Rule, bool) {
	return ts.consume_qualified_rule(false, params...)
}

// https://drafts.csswg.org/css-syntax/#consume-block
func (ts *TokenStream) consume_block() ([]Declaration, []Rule) {
	// Assert the next token is an <open-curly> token
	if ts.next_token().kind != OPEN_CURLY_TOKEN {
		panic("Attempted to consume block with invalid token stream state")
	}
	// Let decls be an empty list of declarations, and rules be an empty list of rules.
	var decls []Declaration
	var rules []Rule

	// Discard a token from input. Consume a block’s contents from input and assign the results to decls and rules. Discard a token from input.
	ts.discard_token()
	decls, rules = ts.consume_block_contents()
	ts.discard_token()
	// Return decls and rules.
	return decls, rules
}

// https://drafts.csswg.org/css-syntax/#consume-block-contents
func (ts *TokenStream) consume_block_contents() ([]Declaration, []Rule) {
	// Let decls be an empty list of declarations, and rules be an empty list of rules.
	var decls []Declaration
	var rules []Rule

	for {
		switch next := ts.next_token(); next.kind {
		// <whitespace-token>, <semicolon-token>
		case WHITESPACE_TOKEN, SEMICOLON_TOKEN:
			// Discard a token from input.
			ts.discard_token()
		// <EOF-token>, <}-token>
		case EOF_TOKEN, CLOSE_CURLY_TOKEN:
			// Return decls and rules.
			return decls, rules
		// <at-keyword-token>
		case AT_KEYWORD_TOKEN:
			// Consume an at-rule from input, with nested set to true. If a rule was returned, append it to rules.
			rule, ok := ts.consume_at_rule(true)
			if ok {
				rules = append(rules, rule)
			}
		// anything else
		default:
			// @TODO: see note about parsing efficiency
			// Mark input.
			ts.mark()
			// Consume a declaration from input, with nested set to true.
			decl, ok := ts.consume_declaration(true)
			// If a declaration was returned, append it to decls, and discard a mark from input.
			if ok {
				decls = append(decls, decl)
				ts.discard_mark()
			} else {
				// Otherwise, restore a mark from input, then consume a qualified rule from input, with nested set to true, and <semicolon-token> as the stop token.
				ts.restore_mark()
				rule, ok := ts.consume_qualified_rule(true, SEMICOLON_TOKEN)
				// If a rule was returned, append it to rules.
				if ok {
					rules = append(rules, rule)
				}
			}
		}
	}
}

func (ts *TokenStream) consume_declaration(nested bool) (Declaration, bool) {
	// Let decl be a new declaration, with an initially empty name and a value set to an empty list.
	decl := Declaration{}
	// 1. If the next token is an <ident-token>, consume a token from input and set decl’s name to the token’s value.
	if ts.next_token().kind == IDENT_TOKEN {
		decl.name = string(ts.consume_token().value)
	} else {
		// Otherwise, consume the remnants of a bad declaration from input, with nested, and return nothing.
		ts.consume_bad_declaration_remnants(nested)
		return decl, false
	}
	// 2. Discard whitespace from input.
	ts.discard_whitespace()
	// 3. If the next token is a <colon-token>, discard a token from input.
	if ts.next_token().kind == COLON_TOKEN {
		ts.discard_token()
	} else {
		// Otherwise, consume the remnants of a bad declaration from input, with nested, and return nothing.
		ts.consume_bad_declaration_remnants(nested)
		return decl, false
	}
	// 4. Discard whitespace from input.
	ts.discard_whitespace()
	// 5. Consume a list of component values from input, with nested, and with <semicolon-token> as the stop token, and set decl’s value to the result.
	decl.value = ts.consume_component_value_list(nested, SEMICOLON_TOKEN)
	// 6. If the last two non-<whitespace-token>s in decl’s value are a <delim-token> with the value "!"
	//    followed by an <ident-token> with a value that is an ASCII case-insensitive match for "important",
	//    remove them from decl’s value and set decl’s important flag.
	if ends_with_important(decl.value) {
		decl.value = decl.value[:len(decl.value)-2]
		decl.important = true
	}
	// 7. While the last item in decl’s value is a <whitespace-token>, remove that token.
	for len(decl.value) > 0 && decl.value[len(decl.value)-1].token.kind == WHITESPACE_TOKEN {
		decl.value = decl.value[:len(decl.value)-1]
	}
	// 8. If decl’s name is a custom property name string, then set decl’s original text to the segment of the original source text string corresponding to the tokens of decl’s value.
	if is_custom_property_name(decl.name) {
		// @TODO
	} else if contains_non_empty_block(decl.value) {
		// Otherwise, if decl’s value contains a top-level simple block with an associated token of <{-token>, and also contains any other non-<whitespace-token> value, return nothing.
		// (That is, a top-level {}-block is only allowed as the entire value of a non-custom property.)
		return decl, false
	} else if is_unicode_range(decl.name) {
		// Otherwise, if decl’s name is an ASCII case-insensitive match for "unicode-range",
		// consume the value of a unicode-range descriptor from the segment of the original source text string corresponding to the tokens returned by the consume a list of component values call,
		// and replace decl’s value with the result.
		// @TODO: We currently do not support unicode ranges
		panic("Parse Error: Unicode Ranges are currently not supported")
	}
	// 9. If decl is valid in the current context, return it; otherwise return nothing.
	return decl, decl.is_valid()
}

func (ts *TokenStream) consume_bad_declaration_remnants(nested bool) {
	for {
		switch next := ts.next_token(); next.kind {
		// <eof-token>, <semicolon-token>
		case EOF_TOKEN, SEMICOLON_TOKEN:
			// Discard a token from input, and return nothing.
			ts.discard_token()
			return
		// <}-token>
		case CLOSE_CURLY_TOKEN:
			// If nested is true, return nothing. Otherwise, discard a token.
			if nested {
				return
			} else {
				ts.discard_token()
			}
		// anything else
		default:
			// Consume a component value from input, and do nothing.
			ts.consume_component_value()
		}
	}
}

// https://drafts.csswg.org/css-syntax/#consume-list-of-components
func (ts *TokenStream) consume_component_value_list(nested bool, params ...TokenKind) []ComponentValue {
	// @NOTE: needed to get round Go's lack of function overloading or default parameters
	stop_token := extract_stop_token(params)
	// Let values be an empty list of component values.
	var values []ComponentValue

	for {
		switch next := ts.next_token(); next.kind {
		// <eof-token>, stop token
		case EOF_TOKEN, stop_token:
			return values
		// <}-token>
		case CLOSE_CURLY_TOKEN:
			// If nested is true, return values.
			if nested {
				return values
			}
			// Otherwise, this is a parse error. Consume a token from input and append the result to values.
			fmt.Println("Parse Error: Encountered unexpected '}' while parsing component value list")
			component := ts.consume_component_value()
			values = append(values, component)
		// anything else
		default:
			// Consume a component value from input, and append the result to values.
			component := ts.consume_component_value()
			values = append(values, component)
		}
	}
}

// https://drafts.csswg.org/css-syntax/#consume-component-value
func (ts *TokenStream) consume_component_value() ComponentValue {
	for {
		switch next := ts.next_token(); next.kind {
		// <{-token>, <[-token>, <(-token>
		case OPEN_CURLY_TOKEN, OPEN_SQUARE_TOKEN, OPEN_PAREN_TOKEN:
			// Consume a simple block from input and return the result.
			return ts.consume_simple_block()
		// <function-token>
		case FUNCTION_TOKEN:
			// Consume a function from input and return the result.
			return ts.consume_function()
		// anything else
		default:
			// Consume a token from input and return the result.
			return ComponentValue{kind: PRESERVED_TOKEN, token: ts.consume_token()}
		}
	}
}

// https://drafts.csswg.org/css-syntax/#consume-simple-block
func (ts *TokenStream) consume_simple_block() ComponentValue {
	// Assert: the next token of input is <{-token>, <[-token>, or <(-token>.
	next := ts.next_token()
	if next.kind != OPEN_CURLY_TOKEN && next.kind != OPEN_SQUARE_TOKEN && next.kind != OPEN_PAREN_TOKEN {
		panic("Attempted to consume simple block with invalid token stream state")
	}

	// Let ending token be the mirror variant of the next token. (E.g. if it was called with <[-token>, the ending token is <]-token>.)
	ending_token := mirror(next.kind)
	// Let block be a new simple block with its associated token set to the next token and with its value initially set to an empty list.
	block := ComponentValue{kind: SIMPLE_BLOCK, token: next}
	// Discard a token from input.
	ts.discard_token()

	for {
		switch next = ts.next_token(); next.kind {
		// <eof-token>, ending token
		case EOF_TOKEN, ending_token:
			// Discard a token from input. Return block.
			ts.discard_token()
			return block
		// anything else
		default:
			// Consume a component value from input and append the result to block’s value.
			block.value = append(block.value, ts.consume_component_value())
		}
	}
}

// https://drafts.csswg.org/css-syntax/#consume-function
func (ts *TokenStream) consume_function() ComponentValue {
	// Assert: The next token is a <function-token>.
	if ts.next_token().kind != FUNCTION_TOKEN {
		panic("Attempted to consume function with invalid token stream state")
	}

	// Consume a token from input, and let function be a new function with its name equal the returned token’s value, and a value set to an empty list.
	token := ts.consume_token()
	function := ComponentValue{kind: FUNCTION, name: string(token.value)}

	for {
		switch next := ts.next_token(); next.kind {
		// <eof-token>, <)-token>
		case EOF_TOKEN, CLOSE_PAREN_TOKEN:
			// Discard a token from input. Return function.
			ts.discard_token()
			return function
		// anything else
		default:
			// Consume a component value from input and append the result to function’s value.
			function.value = append(function.value, ts.consume_component_value())
		}
	}
}

func extract_stop_token(params []TokenKind) TokenKind {
	if len(params) > 0 {
		return params[0]
	}

	return EOF_TOKEN
}
