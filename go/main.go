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

	// bytes, err := io.ReadAll(file)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// input := preprocess_byte_stream(bytes)
	// tokenizer := Tokenizer{input: input, length: len(input), index: -1}

	// var tokens []Token
	// for tokenizer.HasNext() {
	// 	tokens = append(tokens, tokenizer.NextToken())
	// }

	// for i, token := range tokens {
	// 	fmt.Printf("[%d]: %s\n", i, token.ToString())
	// }
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
	return are_valid_escape(t.CurrentRune(), t.NextRune())
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
	return are_start_ident(t.CurrentRune(), t.NextRune(), t.SecondRune())
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
	return are_number(t.CurrentRune(), t.NextRune(), t.SecondRune())
}

// https://www.w3.org/TR/css-syntax-3/#convert-string-to-number
// func convert_string_to_number(repr []rune) float64 {
// 	// @NOTE: This algorithm does not do any verification to ensure that the string contains only a number.
// 	//        Ensure that the string contains only a valid CSS number before calling this algorithm.

// 	// Divide the string into seven components, in order from left to right
// 	index := 0
// 	length := len(repr)

// 	// 1. A sign: a single U+002B PLUS SIGN (+) or U+002D HYPHEN-MINUS (-), or the empty string.
// 	//    Let s be the number -1 if the sign is U+002D HYPHEN-MINUS (-); otherwise, let s be the number 1.
// 	s := 1

// 	switch repr[index] {
// 	case HYPHEN_MINUS_CHAR:
// 		s = -1
// 		index += 1
// 	case PLUS_SIGN_CHAR:
// 		index += 1
// 	}

// 	// 2. An integer part: zero or more digits.
// 	//    If there is at least one digit, let i be the number formed by interpreting the digits as a base-10 integer;
// 	//    otherwise, let i be the number 0.
// 	i := 0

// 	start_index := index
// 	for index < length && is_digit(repr[index]) {
// 		index += 1
// 	}

// 	if index > start_index {
// 		value, err := strconv.ParseInt(string(repr[start_index:index]), 10, 0)
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		i = int(value)
// 	}

// 	// 3. A decimal point: a single U+002E FULL STOP (.), or the empty string.
// 	if index < length && repr[index] == FULL_STOP_CHAR {
// 		index += 1
// 	}

// 	// 4. A fractional part: zero or more digits.
// 	//    If there is at least one digit, let f be the number formed by interpreting the digits as a base-10 integer and d be the number of digits;
// 	//    otherwise, let f and d be the number 0.
// 	f := 0
// 	d := 0

// 	start_index = index
// 	for index < length && is_digit(repr[index]) {
// 		index += 1
// 	}

// 	if index > start_index {
// 		value, err := strconv.ParseInt(string(repr[start_index:index]), 10, 0)
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		f = int(value)
// 		d = index - start_index
// 	}

// 	// 5. An exponent indicator: a single U+0045 LATIN CAPITAL LETTER E (E) or U+0065 LATIN SMALL LETTER E (e), or the empty string.
// 	if index < length && (repr[index] == UPPER_E_CHAR || repr[index] == LOWER_E_CHAR) {
// 		index += 1
// 	}

// 	// 6. An exponent sign: a single U+002B PLUS SIGN (+) or U+002D HYPHEN-MINUS (-), or the empty string.
// 	//    Let t be the number -1 if the sign is U+002D HYPHEN-MINUS (-); otherwise, let t be the number 1.
// 	t := 0

// 	if index < length {
// 		switch repr[index] {
// 		case HYPHEN_MINUS_CHAR:
// 			t = -1
// 			index += 1
// 		case PLUS_SIGN_CHAR:
// 			index += 1
// 		}
// 	}

// 	// 7. An exponent: zero or more digits.
// 	//    If there is at least one digit, let e be the number formed by interpreting the digits as a base-10 integer;
// 	//    otherwise, let e be the number 0.
// 	e := 0

// 	start_index = index
// 	for index < length && is_digit(repr[index]) {
// 		index += 1
// 	}

// 	if index > start_index {
// 		value, err := strconv.ParseInt(string(repr[start_index:index]), 10, 0)
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		e = int(value)
// 	}

// 	// Return the number s·(i + f·10^(-d))·10^(te).
// 	exponent := math.Pow10(t * e)
// 	fraction := float64(f) * math.Pow10(-1*d)
// 	return float64(s) * (float64(i) + fraction) * exponent
// }

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

type TokenType uint8

// https://drafts.csswg.org/css-syntax/#tokenization
const (
	IDENT_TOKEN TokenType = iota
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

func token_type_name(kind TokenType) string {
	switch kind {
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

	return ""
}

func mirror(kind TokenType) TokenType {
	switch kind {
	case OPEN_SQUARE_TOKEN:
		return CLOSE_SQUARE_TOKEN
	case OPEN_PAREN_TOKEN:
		return CLOSE_PAREN_TOKEN
	case OPEN_CURLY_TOKEN:
		return CLOSE_CURLY_TOKEN
	default:
		log.Panicf("Invalid call to mirror with TokenType '%d'", kind)
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
	kind TokenType
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

func (t Token) ToString() string {
	kind := t.kind

	str := fmt.Sprintf("%s", token_type_name(kind))

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

func (t *Tokenizer) HasNext() bool {
	return t.index < t.length
}

// https://drafts.csswg.org/css-syntax/#consume-token
func (t *Tokenizer) ConsumeToken() Token {
	// Additionally takes an optional boolean unicode ranges allowed, defaulting to false.
	unicode_ranges_allowed := false
	// Consume comments.
	t.consume_comments()
	// Consume the next input code point.
	char := t.ConsumeNext()
	switch {
	case is_whitespace(char):
		// Consume as much whitespace as possible.
		for is_whitespace(t.NextRune()) {
			t.ConsumeNext()
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
		if is_ident(t.NextRune()) || are_valid_escape(t.NextRune(), t.SecondRune()) {
			// 1. Create a <hash-token>.
			token := Token{kind: HASH_TOKEN}
			// 2. If the next 3 input code points would start an ident sequence, set the <hash-token>’s type flag to "id".
			if are_start_ident(t.NextRune(), t.SecondRune(), t.ThirdRune()) {
				token.hash_flag = HASH_ID
			}
			// 3. Consume an ident sequence, and set the <hash-token>’s value to the returned string.
			token.value = t.consume_ident_sequence()
			// 4. Return the <hash-token>.
			return token
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.CurrentRune()}}
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
			t.ReconsumeCurrent()
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
			t.ReconsumeCurrent()
			return t.consume_numeric_token()
		}

		// Otherwise, if the next 2 input code points are U+002D HYPHEN-MINUS U+003E GREATER-THAN SIGN (->), consume them and return a <CDC-token>.
		if t.NextRune() == HYPHEN_MINUS_CHAR && t.SecondRune() == GREATER_THAN_CHAR {
			t.ConsumeRunes(2)
			return Token{kind: CDC_TOKEN}
		}

		// Otherwise, if the input stream starts with an ident sequence, reconsume the current input code point, consume an ident-like token, and return it.
		if t.starts_with_ident() {
			t.ReconsumeCurrent()
			return t.consume_ident_like_token()
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.CurrentRune()}}
	// U+002E FULL STOP (.)
	case char == FULL_STOP_CHAR:
		// If the input stream starts with a number, reconsume the current input code point, consume a numeric token, and return it.
		if t.starts_with_number() {
			t.ReconsumeCurrent()
			return t.consume_numeric_token()
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.CurrentRune()}}
	// U+003A COLON (:)
	case char == COLON_CHAR:
		return Token{kind: COLON_TOKEN}
	// U+003B SEMICOLON (;)
	case char == SEMICOLON_CHAR:
		return Token{kind: SEMICOLON_TOKEN}
	// U+003C LESS-THAN SIGN (<)
	case char == LESS_THAN_CHAR:
		// If the next 3 input code points are U+0021 EXCLAMATION MARK U+002D HYPHEN-MINUS U+002D HYPHEN-MINUS (!--), consume them and return a <CDO-token>.
		first, second, third := t.NextRune(), t.SecondRune(), t.ThirdRune()
		if first == EXCLAMATON_MARK_CHAR && second == HYPHEN_MINUS_CHAR && third == HYPHEN_MINUS_CHAR {
			t.ConsumeRunes(3)
			return Token{kind: CDO_TOKEN}
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.CurrentRune()}}
	// U+0040 COMMERCIAL AT (@)
	case char == AT_CHAR:
		// If the next 3 input code points would start an ident sequence
		if are_start_ident(t.NextRune(), t.SecondRune(), t.ThirdRune()) {
			// Consume an ident sequence, create an <at-keyword-token> with its value set to the returned value, and return it.
			return Token{kind: AT_KEYWORD_TOKEN, value: t.consume_ident_sequence()}
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.CurrentRune()}}
	// U+005B LEFT SQUARE BRACKET ([)
	case char == OPEN_SQUARE_CHAR:
		return Token{kind: OPEN_SQUARE_TOKEN}
	// U+005C REVERSE SOLIDUS (\)
	case char == BACKWARD_SLASH_CHAR:
		// If the input stream starts with a valid escape, reconsume the current input code point, consume an ident-like token, and return it.
		if t.starts_with_valid_escape() {
			t.ReconsumeCurrent()
			return t.consume_ident_like_token()
		}

		// Otherwise, this is a parse error. Return a <delim-token> with its value set to the current input code point.
		fmt.Println("Parse Error: Encountered invalid escape")
		return Token{kind: DELIM_TOKEN, value: []rune{t.CurrentRune()}}
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
		t.ReconsumeCurrent()
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
		t.ReconsumeCurrent()
		return t.consume_ident_like_token()
	// ident-start code point
	case is_ident_start(char):
		// Reconsume the current input code point, consume an ident-like token, and return it.
		t.ReconsumeCurrent()
		return t.consume_ident_like_token()
	// EOF
	case char == EOF_CHAR:
		return Token{kind: EOF_TOKEN}
	// anything else
	default:
		// Return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.CurrentRune()}}
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
func (t *Tokenizer) CurrentRune() rune {
	return t.peek_rune(0)
}

// The first code point in the input stream that has not yet been consumed.
// https://drafts.csswg.org/css-syntax/#next-input-code-point
func (t *Tokenizer) NextRune() rune {
	return t.peek_rune(1)
}

// The code point in the input stream immediately after the next rune.
func (t *Tokenizer) SecondRune() rune {
	return t.peek_rune(2)
}

// // The code point in the input stream immediately after the second rune.
func (t *Tokenizer) ThirdRune() rune {
	return t.peek_rune(3)
}

func (t *Tokenizer) ConsumeRunes(number int) rune {
	t.index += number
	return t.CurrentRune()
}

func (t *Tokenizer) ConsumeNext() rune {
	return t.ConsumeRunes(1)
}

// Push the current input code point back onto the front of the input stream, so that the next time you are instructed to consume the next input code point,
// it will instead reconsume the current input code point.
// https://drafts.csswg.org/css-syntax/#reconsume-the-current-input-code-point
func (t *Tokenizer) ReconsumeCurrent() {
	t.ConsumeRunes(-1)
}

// https://drafts.csswg.org/css-syntax/#consume-comment
func (t *Tokenizer) consume_comments() {
	// If the next two input code point are U+002F SOLIDUS (/) followed by a U+002A ASTERISK (*)
	for t.NextRune() == FORWARD_SLASH_CHAR && t.SecondRune() == ASTERISK_CHAR {
		// consume them
		t.ConsumeRunes(2)

		for {
			// and all following code points up to and including
			char := t.ConsumeNext()
			// the first U+002A ASTERISK (*) followed by a U+002F SOLIDUS (/)
			if char == ASTERISK_CHAR && t.NextRune() == FORWARD_SLASH_CHAR {
				t.ConsumeRunes(2)
				break
			} else if char == EOF_CHAR { // or up to an EOF code point
				// If the preceding paragraph ended by consuming an EOF code point, this is a parse error
				log.Fatal("Parse Error: Unexpected EOF when parsing comment")
			}
		}
	}
}

// This algorithm may be called with an `ending` code point, which denotes the code point that ends the string.
// https://drafts.csswg.org/css-syntax/#consume-string-token
func (t *Tokenizer) consume_string_token(ending rune) Token {
	// Returns either a <string-token> or <bad-string-token>.

	// Initially create a <string-token> with its value set to the empty string.
	token := Token{kind: STRING_TOKEN, value: nil}

	// Repeatedly consume the next input code point from the stream:
	for {
		char := t.ConsumeNext()
		switch {
		// ending code point:
		case char == ending:
			// Return the <string-token>.
			return token
		// EOF
		case char == EOF_CHAR:
			// This is a parse error. Return the <string-token>.
			fmt.Println("Parse Error: Unexpected EOF when parsing string")
			return token
		// newline:
		case is_newline(char):
			// This is a parse error. Reconsume the current input code point, create a <bad-string-token>, and return it.
			fmt.Println("Parse Error: Unexpected newline when parsing string")
			t.ReconsumeCurrent()
			return Token{kind: BAD_STRING_TOKEN}
		// U+005C REVERSE SOLIDUS (\):
		case char == BACKWARD_SLASH_CHAR:
			// If the next input code point is EOF, do nothing.
			next := t.NextRune()
			if next != EOF_CHAR {
				// Otherwise, if the next input code point is a newline, consume it.
				if is_newline(next) {
					t.ConsumeNext()
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
	return t.consume_string_token(t.CurrentRune())
}

// https://drafts.csswg.org/css-syntax/#consume-escaped-code-point
func (t *Tokenizer) consume_escaped() rune {
	// Assume that the U+005C REVERSE SOLIDUS (\) has already been consumed
	// and that the next input code point has already been verified to be part of a valid escape.
	// Returns a code point.

	// Consume the next input code point.
	char := t.ConsumeNext()
	switch {
	// hex digit
	case is_hex_digit(char):
		// Consume as many hex digits as possible, but no more than 5.
		// @Assertion: Note that this means 1-6 hex digits have been consumed in total.
		digits := make([]rune, 0, 6)
		digits = append(digits, char)

		for i := 0; i < 5; i += 1 {
			if is_hex_digit(t.NextRune()) {
				digits = append(digits, t.ConsumeNext())
			} else {
				break
			}
		}

		// If the next input code point is whitespace, consume it as well.
		if is_whitespace(t.NextRune()) {
			t.ConsumeNext()
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
		fmt.Println("Parse Error: Unexpected EOF when parsing escape")
		return REPLACEMENT_CHAR
	// anything else
	default:
		// Return the current input code point.
		return t.CurrentRune()
	}
}

// https://drafts.csswg.org/css-syntax/#consume-name
func (t *Tokenizer) consume_ident_sequence() []rune {
	// Returns a string containing the largest name that can be formed from adjacent code points in the stream, starting from the first.

	// Let result initially be an empty string.
	var result []rune
	// Repeatedly consume the next input code point from the stream:
	for {
		char := t.ConsumeNext()
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
			t.ReconsumeCurrent()
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
	if are_start_ident(t.NextRune(), t.SecondRune(), t.ThirdRune()) {
		// 1. Create a <dimension-token> with the same value, type flag, and sign character as number, and a unit set initially to the empty string.
		token := Token{kind: DIMENSION_TOKEN, numeric: number, type_flag: type_flag, sign: sign, unit: nil}
		// 2. Consume an ident sequence. Set the <dimension-token>’s unit to the returned value.
		token.unit = t.consume_ident_sequence()
		// 3. Return the <dimension-token>.
		return token
	}
	// Otherwise, if the next input code point is U+0025 PERCENTAGE SIGN (%), consume it.
	if t.NextRune() == PERCENT_SIGN_CHAR {
		t.ConsumeNext()
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
	next := t.NextRune()
	if next == PLUS_SIGN_CHAR || next == HYPHEN_MINUS_CHAR {
		t.ConsumeNext()
		// Append it to number part and set sign character to it.
		number_part = append(number_part, next)
		sign = []rune{next}
	}

	// 3. While the next input code point is a digit, consume it and append it to number part.
	for is_digit(t.NextRune()) {
		number_part = append(number_part, t.ConsumeNext())
	}

	// 4. If the next 2 input code points are U+002E FULL STOP (.) followed by a digit
	if t.NextRune() == FULL_STOP_CHAR && is_digit(t.SecondRune()) {
		// 4.1 Consume the next input code point and append it to number part.
		number_part = append(number_part, t.ConsumeNext())
		// 4.2 While the next input code point is a digit, consume it and append it to number part.
		for is_digit(t.NextRune()) {
			number_part = append(number_part, t.ConsumeNext())
		}
		// 4.3 Set type to "number".
		type_flag = TYPE_NUMBER
	}

	// 5. If the next 2 or 3 input code points are U+0045 LATIN CAPITAL LETTER E (E) or U+0065 LATIN SMALL LETTER E (e),
	//    optionally followed by U+002D HYPHEN-MINUS (-) or U+002B PLUS SIGN (+),
	//    followed by a digit
	first, second, third := t.NextRune(), t.SecondRune(), t.ThirdRune()
	lookahead := 2
	if first == UPPER_E_CHAR || first == LOWER_E_CHAR {
		if second == HYPHEN_MINUS_CHAR || second == PLUS_SIGN_CHAR {
			lookahead = 3
		}

		if (lookahead == 2 && is_digit(second)) || (lookahead == 3 && is_digit(third)) {
			// 5.1 Consume the next input code point.
			t.ConsumeNext()
			// 5.2 If the next input code point is "+" or "-", consume it and append it to exponent part.
			switch t.NextRune() {
			case PLUS_SIGN_CHAR, HYPHEN_MINUS_CHAR:
				exponent_part = append(exponent_part, t.ConsumeNext())
			}
			// 5.3 While the next input code point is a digit, consume it and append it to exponent part.
			for is_digit(t.NextRune()) {
				exponent_part = append(exponent_part, t.ConsumeNext())
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
	if strings.EqualFold(string(str), "url") && t.NextRune() == OPEN_PAREN_CHAR {
		t.ConsumeNext()
		// While the next two input code points are whitespace, consume the next input code point.
		for is_whitespace(t.NextRune()) && is_whitespace(t.SecondRune()) {
			t.ConsumeNext()
		}
		// If the next one or two input code points are U+0022 QUOTATION MARK ("), U+0027 APOSTROPHE ('),
		// or whitespace followed by U+0022 QUOTATION MARK (") or U+0027 APOSTROPHE (')
		first, second := t.NextRune(), t.SecondRune()
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
	if t.NextRune() == OPEN_PAREN_CHAR {
		t.ConsumeNext()
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
	token := Token{kind: URL_TOKEN, value: nil}
	// 2. Consume as much whitespace as possible.
	for is_whitespace(t.NextRune()) {
		t.ConsumeNext()
	}
	// 3. Repeatedly consume the next input code point from the stream:
	for {
		char := t.ConsumeNext()
		switch {
		// U+0029 RIGHT PARENTHESIS ())
		case char == CLOSE_PAREN_CHAR:
			// Return the <url-token>.
			return token
		case char == EOF_CHAR:
			// This is a parse error. Return the <url-token>.
			fmt.Println("Parse Error: Unexpected EOF while parsing URL")
			return token
		case is_whitespace(char):
			// Consume as much whitespace as possible.
			for is_whitespace(t.NextRune()) {
				t.ConsumeNext()
			}
			// If the next input code point is U+0029 RIGHT PARENTHESIS ())
			// or EOF, consume it and return the <url-token> (if EOF was encountered, this is a parse error);
			switch t.NextRune() {
			case CLOSE_PAREN_CHAR:
				t.ConsumeNext()
				return token
			case EOF_CHAR:
				fmt.Println("Parse Error: Unexpected EOF while parsing URL")
				t.ConsumeNext()
				return token
			// otherwise, consume the remnants of a bad url, create a <bad-url-token>, and return it.
			default:
				t.consume_bad_url_remnants()
				return Token{kind: BAD_URL_TOKEN}
			}
		// U+0022 QUOTATION MARK ("), U+0027 APOSTROPHE ('), U+0028 LEFT PARENTHESIS ((), non-printable code point
		case char == QUOTATION_MARK_CHAR, char == APOSTROPHE_CHAR, char == OPEN_PAREN_CHAR, is_non_printable(char):
			// This is a parse error. Consume the remnants of a bad url, create a <bad-url-token>, and return it.
			fmt.Printf("Parse Error: Unexpected character '%x' while parsing URL\n", char)
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
			token.value = append(token.value, t.CurrentRune())
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
		char := t.ConsumeNext()
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

type Parser struct {
	tokens    []Token
	length    int
	index     int
	reconsume bool
}

func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:    tokens,
		length:    len(tokens),
		index:     -1,
		reconsume: false,
	}
}

type ParserOutputType uint8

// https://www.w3.org/TR/css-syntax-3/#parsing
const (
	AT_RULE ParserOutputType = iota
	// Most qualified rules will be style rules, where the prelude is a selector [SELECT] and the block a list of declarations.
	QUALIFIED_RULE
	DECLARATION
	PRESERVED_TOKEN
	FUNCTION
	SIMPLE_BLOCK
)

type ParserOutput struct {
	kind ParserOutputType
	// <at-rule>, <declaration>, <function>
	name []rune
	// <at-rule>, <qualified-rule>
	prelude []ParserOutput
	// <at-rule> (optional), <qualified-rule>
	block *ParserOutput
	// <declaration>, <function>, <simple-block>
	value []ParserOutput
	// <declaration>
	important bool
	// <preserved-token>, <simple-block>
	token Token
}

func (p ParserOutput) ToString() string {
	kind := p.kind

	var type_string string
	switch kind {
	case AT_RULE:
		type_string = "AT_RULE"
	case QUALIFIED_RULE:
		type_string = "QUALIFIED_RULE"
	case DECLARATION:
		type_string = "DECLARATION"
	case PRESERVED_TOKEN:
		type_string = "PRESERVED_TOKEN"
	case FUNCTION:
		type_string = "FUNCTION"
	case SIMPLE_BLOCK:
		type_string = "SIMPLE_BLOCK"
	}

	str := fmt.Sprintf("<%s>", type_string)

	if kind == AT_RULE || kind == DECLARATION || kind == FUNCTION {
		str = str + fmt.Sprintf(" '%s'", string(p.name))
	}

	if kind == AT_RULE || kind == QUALIFIED_RULE {
		str = str + "\n\tprelude: ("
		for _, item := range p.prelude {
			str = str + fmt.Sprintf("%s, ", item.ToString())
		}
	}

	if kind == AT_RULE || kind == QUALIFIED_RULE {
		if p.block != nil {
			str = str + fmt.Sprintf("\n\tblock: %s", p.block.ToString())
		}
	}

	if kind == DECLARATION || kind == FUNCTION || kind == SIMPLE_BLOCK {
		str = str + "\n\t\tvalue:\n"
		for _, item := range p.value {
			str = str + fmt.Sprintf("\t\t\t%s,\n ", item.ToString())
		}
	}

	if kind == PRESERVED_TOKEN || kind == SIMPLE_BLOCK {
		str = str + fmt.Sprintf(" token: %s", p.token.ToString())
	}

	return str
}

func (p *Parser) peek_token(number int) Token {
	index := p.index + number
	// If there isn’t a token following the current input token, the next input token is an <EOF-token>.
	if index >= p.length {
		return Token{kind: EOF_TOKEN}
	}

	return p.tokens[index]
}

// The token or component value currently being operated on, from the list of tokens produced by the tokenizer.
func (p *Parser) CurrentToken() Token {
	return p.peek_token(0)
}

// The token or component value following the current input token in the list of tokens produced by the tokenizer.
func (p *Parser) NextToken() Token {
	return p.peek_token(1)
}

func (p *Parser) consume_token(number int) {
	p.index += number
}

// Let the current input token be the current next input token, adjusting the next input token accordingly.
func (p *Parser) ConsumeNext() {
	if p.reconsume == false {
		p.consume_token(1)
	}

	p.reconsume = false
}

// The next time an algorithm instructs you to consume the next input token, instead do nothing (retain the current input token unchanged).
func (p *Parser) ReconsumeCurrent() {
	p.reconsume = true
}

func ParseStylesheet(input io.Reader) {
	bytes, err := io.ReadAll(input)
	if err != nil {
		log.Fatal(err)
	}

	stream := preprocess_input_stream(bytes)
	tokenizer := NewTokenizer(stream)

	var tokens []Token
	for tokenizer.HasNext() {
		token := tokenizer.ConsumeToken()
		fmt.Println(token.ToString())
		tokens = append(tokens, token)
	}

	// parser := NewParser(tokens)
	// output := parser.consume_rules_list(true)
	// for i, elem := range output {
	// 	fmt.Printf("[%d]: %s\n", i, elem.ToString())
	// }
}

// https://www.w3.org/TR/css-syntax-3/#consume-list-of-rules
func (p *Parser) consume_rules_list(top_level bool) []ParserOutput {
	// Create an initially empty list of rules.
	var rules []ParserOutput

	// Repeatedly consume the next input token:
	for {
		p.ConsumeNext()

		token := p.CurrentToken()
		switch token.kind {
		// <whitespace-token>
		case WHITESPACE_TOKEN:
			// Do nothing
			continue
		// <EOF-token>
		case EOF_TOKEN:
			// Return the list of rules.
			return rules
		// <CDO-token>, <CDC-token>
		case CDO_TOKEN:
			fallthrough
		case CDC_TOKEN:
			// If the top-level flag is set, do nothing.
			if top_level {
				continue
			}

			// Otherwise, reconsume the current input token. Consume a qualified rule. If anything is returned, append it to the list of rules.
			p.ReconsumeCurrent()
			rule, ok := p.consume_qualified_rule()
			if ok {
				rules = append(rules, rule)
			}
		// <at-keyword-token>
		case AT_KEYWORD_TOKEN:
			// Reconsume the current input token. Consume an at-rule, and append the returned value to the list of rules.
			p.ReconsumeCurrent()
			rules = append(rules, p.consume_at_rule())
		// anything else
		default:
			// Reconsume the current input token. Consume a qualified rule. If anything is returned, append it to the list of rules.
			p.ReconsumeCurrent()
			rule, ok := p.consume_qualified_rule()
			if ok {
				rules = append(rules, rule)
			}
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-qualified-rule
func (p *Parser) consume_qualified_rule() (ParserOutput, bool) {
	// Create a new qualified rule with its prelude initially set to an empty list, and its value initially set to nothing.
	rule := ParserOutput{kind: QUALIFIED_RULE, prelude: nil, value: nil}
	// Repeatedly consume the next input token:
	for {
		p.ConsumeNext()

		token := p.CurrentToken()
		switch {
		// <EOF-token>
		case token.kind == EOF_TOKEN:
			// This is a parse error. Return nothing.
			fmt.Println("Parse Error: Unexpected EOF when parsing qualified rule")
			return rule, false
		// <{-token>
		case token.kind == OPEN_CURLY_TOKEN:
			// Consume a simple block and assign it to the qualified rule’s block. Return the qualified rule.
			block := p.consume_simple_block()
			rule.block = &block
			return rule, true
		// @FIXME: work out how this works
		// simple block with an associated token of <{-token>
		// case false:
		// 	// Assign the block to the qualified rule’s block. Return the qualified rule.
		// 	rule.block = 0
		// 	return rule, true
		// anything else
		default:
			// Reconsume the current input token. Consume a component value. Append the returned value to the qualified rule’s prelude.
			p.ReconsumeCurrent()
			rule.prelude = append(rule.prelude, p.consume_component_value())
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-at-rule
func (p *Parser) consume_at_rule() ParserOutput {
	// Consume the next input token.
	p.ConsumeNext()
	// Create a new at-rule with its name set to the value of the current input token,
	// its prelude initially set to an empty list, and its value initially set to nothing.
	rule := ParserOutput{kind: AT_RULE, name: p.CurrentToken().value, prelude: nil, value: nil}

	// Repeatedly consume the next input token:
	for {
		p.ConsumeNext()

		token := p.CurrentToken()
		switch {
		// <semicolon-token>
		case token.kind == SEMICOLON_TOKEN:
			// Return the at-rule.
			return rule
		// <EOF-token>
		case token.kind == EOF_TOKEN:
			// This is a parse error. Return the at-rule.
			fmt.Println("Parse Error: Unexpected EOF when parsing at rule")
			return rule
		// <{-token>
		case token.kind == OPEN_CURLY_TOKEN:
			// Consume a simple block and assign it to the at-rule’s block. Return the at-rule.
			block := p.consume_simple_block()
			rule.block = &block
			return rule
		// @FIXME: work out how this works
		// simple block with an associated token of <{-token>
		// case false:
		// 	// Assign the block to the at-rule’s block. Return the at-rule.
		// 	rule.block = 0
		// 	return rule
		// anything else
		default:
			// Reconsume the current input token. Consume a component value. Append the returned value to the at-rule’s prelude.
			p.ReconsumeCurrent()
			rule.prelude = append(rule.prelude, p.consume_component_value())
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-simple-block
func (p *Parser) consume_simple_block() ParserOutput {
	// @ASSERTION: This algorithm assumes that the current input token has already been checked to be an <{-token>, <[-token>, or <(-token>.

	current := p.CurrentToken()
	// The ending token is the mirror variant of the current input token. (E.g. if it was called with <[-token>, the ending token is <]-token>.)
	ending_type := mirror(current.kind)
	// Create a simple block with its associated token set to the current input token and with its value initially set to an empty list.
	block := ParserOutput{kind: SIMPLE_BLOCK, token: current, value: nil}
	// Repeatedly consume the next input token and process it as follows:
	for {
		p.ConsumeNext()

		token := p.CurrentToken()
		switch token.kind {
		// ending token
		case ending_type:
			// Return the block.
			return block
		// <EOF-token>
		case EOF_TOKEN:
			// This is a parse error. Return the block.
			fmt.Println("Parse Error: Unexpected EOF when parsing simple block")
			return block
		// anything else
		default:
			// Reconsume the current input token. Consume a component value and append it to the value of the block.
			p.ReconsumeCurrent()
			block.value = append(block.value, p.consume_component_value())
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-component-value
func (p *Parser) consume_component_value() ParserOutput {
	// Consume the next input token.
	p.ConsumeNext()
	// If the current input token is a <{-token>, <[-token>, or <(-token>, consume a simple block and return it.
	switch p.CurrentToken().kind {
	case OPEN_CURLY_TOKEN:
		fallthrough
	case OPEN_SQUARE_TOKEN:
		fallthrough
	case OPEN_PAREN_TOKEN:
		return p.consume_simple_block()
	// Otherwise, if the current input token is a <function-token>, consume a function and return it.
	case FUNCTION_TOKEN:
		return p.consume_function()
	// Otherwise, return the current input token.
	default:
		return ParserOutput{kind: PRESERVED_TOKEN, token: p.CurrentToken()}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-function
func (p *Parser) consume_function() ParserOutput {
	// @ASSERTION: This algorithm assumes that the current input token has already been checked to be a <function-token>.

	// Create a function with its name equal to the value of the current input token and with its value initially set to an empty list.
	function := ParserOutput{kind: FUNCTION, name: p.CurrentToken().value, value: nil}
	// Repeatedly consume the next input token
	for {
		p.ConsumeNext()

		token := p.CurrentToken()
		switch token.kind {
		// <)-token>
		case CLOSE_PAREN_TOKEN:
			// Return the function.
			return function
		// <EOF-token>
		case EOF_TOKEN:
			// This is a parse error. Return the function.
			fmt.Println("Parse Error: Unexpected EOF when parsing function")
			return function
		// anything else
		default:
			// Reconsume the current input token. Consume a component value and append the returned value to the function’s value.
			p.ReconsumeCurrent()
			function.value = append(function.value, p.consume_component_value())
		}
	}
}
