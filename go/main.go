package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	// https://unicodebook.readthedocs.io/unicode_encodings.html#utf-8
	MULTIBYTE_START_MARKER = 0b11000000
	MULTIBYTE_BODY_MARKER  = 0b10000000
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

	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	input := preprocess_byte_stream(bytes)
	tokenizer := Tokenizer{input: input, length: len(input), index: -1}

	var tokens []Token
	for tokenizer.HasNext() {
		tokens = append(tokens, tokenizer.NextToken())
	}

	for i, token := range tokens {
		fmt.Printf("[%d]: %s\n", i, token.ToString())
	}
}

// https://www.w3.org/TR/css-syntax-3/#input-preprocessing
func preprocess_byte_stream(bytes []byte) []rune {
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
			log.Fatalf("Invalid UTF-8 encoding: %v", bytes)
		}

		char = filter_code_point(char)
		result = append(result, char)
	}

	return result
}

// https://www.w3.org/TR/css-syntax-3/#input-preprocessing
func filter_code_point(char rune) rune {
	switch char {
	// Replace any
	// U+000D CARRIAGE RETURN (CR) code points,
	// U+000C FORM FEED (FF) code points,
	// or pairs of U+000D CARRIAGE RETURN (CR) followed by U+000A LINE FEED (LF) in input
	// by a single U+000A LINE FEED (LF) code point.
	// @NOTE: We currently don't merge the CR LF into a single LF, instead emitting LF LF
	case FORM_FEED_CHAR, CARRIAGE_RETURN_CHAR:
		return LINE_FEED_CHAR
	// Replace any U+0000 NULL or surrogate code points in input with U+FFFD REPLACEMENT CHARACTER (�).
	// @NOTE: We don't support utf16 encoding so there can be no surrogate code points
	case NULL_CHAR:
		return REPLACEMENT_CHAR
	default:
		return char
	}
}

func is_multibyte_start(bite byte) bool {
	return bite&MULTIBYTE_START_MARKER == MULTIBYTE_START_MARKER
}

func is_multibyte_body(bite byte) bool {
	return bite&MULTIBYTE_BODY_MARKER == MULTIBYTE_BODY_MARKER
}

func is_newline(char rune) bool {
	// U+000A LINE FEED

	// Note that U+000D CARRIAGE RETURN and U+000C FORM FEED are not included in this definition,
	// as they are converted to U+000A LINE FEED during preprocessing
	return char == LINE_FEED_CHAR
}

func is_whitespace(char rune) bool {
	// newline,
	// U+0009 CHARACTER TABULATION,
	// or U+0020 SPACE.
	return is_newline(char) || char == TAB_CHAR || char == SPACE_CHAR
}

func is_digit(char rune) bool {
	// A code point between U+0030 DIGIT ZERO (0) and U+0039 DIGIT NINE (9) inclusive.
	return char >= '0' && char <= '9'
}

func is_hex_digit(char rune) bool {
	// A digit,
	// or a code point between U+0041 LATIN CAPITAL LETTER A (A) and U+0046 LATIN CAPITAL LETTER F (F) inclusive,
	// or a code point between U+0061 LATIN SMALL LETTER A (a) and U+0066 LATIN SMALL LETTER F (f) inclusive.
	return is_digit(char) || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')
}

func is_uppercase(char rune) bool {
	// A code point between U+0041 LATIN CAPITAL LETTER A (A) and U+005A LATIN CAPITAL LETTER Z (Z) inclusive.
	return char >= 'A' && char <= 'Z'
}

func is_lowercase(char rune) bool {
	// A code point between U+0061 LATIN SMALL LETTER A (a) and U+007A LATIN SMALL LETTER Z (z) inclusive.
	return char >= 'a' && char <= 'z'
}

func is_letter(char rune) bool {
	// An uppercase letter or a lowercase letter.
	return is_uppercase(char) || is_lowercase(char)
}

func is_non_ascii(char rune) bool {
	// A code point with a value equal to or greater than U+0080 <control>.
	return char > CONTROL_CHAR
}

func is_ident_start(char rune) bool {
	// A letter,
	// a non-ASCII code point,
	// or U+005F LOW LINE (_).
	return is_letter(char) || is_non_ascii(char) || char == LOW_LINE_CHAR
}

func is_ident(char rune) bool {
	// An ident-start code point,
	// a digit,
	// or U+002D HYPHEN-MINUS (-).
	return is_ident_start(char) || is_digit(char) || char == HYPHEN_MINUS_CHAR
}

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
// https://www.w3.org/TR/css-syntax-3/#starts-with-a-valid-escape
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

// https://www.w3.org/TR/css-syntax-3/#starts-with-a-valid-escape
func (t *Tokenizer) starts_with_valid_escape() bool {
	// The two code points in question are the current input code point and the next input code point, in that order.
	return are_valid_escape(t.CurrentRune(), t.NextRune())
}

// Check if three code points would start an ident sequence.
// https://www.w3.org/TR/css-syntax-3/#would-start-an-identifier
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
	default:
		return false
	}
}

// https://www.w3.org/TR/css-syntax-3/#would-start-an-identifier
func (t *Tokenizer) starts_with_ident() bool {
	// The three code points in question are the current input code point and the next two input code points, in that order.
	return are_start_ident(t.CurrentRune(), t.NextRune(), t.SecondRune())
}

// Check if three code points would start a number.
// https://www.w3.org/TR/css-syntax-3/#starts-with-a-number
func are_number(first rune, second rune, third rune) bool {
	// @ASSERTION: This algorithm will not consume any additional code points.

	// Look at the first code point
	switch {
	// U+002B PLUS SIGN (+), U+002D HYPHEN-MINUS (-)
	case first == PLUS_SIGN_CHAR || first == HYPHEN_MINUS_CHAR:
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

func (t *Tokenizer) starts_with_number() bool {
	// The three code points in question are the current input code point and the next two input code points, in that order.
	return are_number(t.CurrentRune(), t.NextRune(), t.SecondRune())
}

// https://www.w3.org/TR/css-syntax-3/#convert-string-to-number
func convert_string_to_number(repr []rune) float64 {
	// @NOTE: This algorithm does not do any verification to ensure that the string contains only a number.
	//        Ensure that the string contains only a valid CSS number before calling this algorithm.

	// Divide the string into seven components, in order from left to right
	index := 0
	length := len(repr)

	// 1. A sign: a single U+002B PLUS SIGN (+) or U+002D HYPHEN-MINUS (-), or the empty string.
	//    Let s be the number -1 if the sign is U+002D HYPHEN-MINUS (-); otherwise, let s be the number 1.
	s := 1

	switch repr[index] {
	case HYPHEN_MINUS_CHAR:
		s = -1
		index += 1
	case PLUS_SIGN_CHAR:
		index += 1
	}

	// 2. An integer part: zero or more digits.
	//    If there is at least one digit, let i be the number formed by interpreting the digits as a base-10 integer;
	//    otherwise, let i be the number 0.
	i := 0

	start_index := index
	for index < length && is_digit(repr[index]) {
		index += 1
	}

	if index > start_index {
		value, err := strconv.ParseInt(string(repr[start_index:index]), 10, 0)
		if err != nil {
			log.Fatal(err)
		}

		i = int(value)
	}

	// 3. A decimal point: a single U+002E FULL STOP (.), or the empty string.
	if index < length && repr[index] == FULL_STOP_CHAR {
		index += 1
	}

	// 4. A fractional part: zero or more digits.
	//    If there is at least one digit, let f be the number formed by interpreting the digits as a base-10 integer and d be the number of digits;
	//    otherwise, let f and d be the number 0.
	f := 0
	d := 0

	start_index = index
	for index < length && is_digit(repr[index]) {
		index += 1
	}

	if index > start_index {
		value, err := strconv.ParseInt(string(repr[start_index:index]), 10, 0)
		if err != nil {
			log.Fatal(err)
		}

		f = int(value)
		d = index - start_index
	}

	// 5. An exponent indicator: a single U+0045 LATIN CAPITAL LETTER E (E) or U+0065 LATIN SMALL LETTER E (e), or the empty string.
	if index < length && (repr[index] == UPPER_E_CHAR || repr[index] == LOWER_E_CHAR) {
		index += 1
	}

	// 6. An exponent sign: a single U+002B PLUS SIGN (+) or U+002D HYPHEN-MINUS (-), or the empty string.
	//    Let t be the number -1 if the sign is U+002D HYPHEN-MINUS (-); otherwise, let t be the number 1.
	t := 0

	if index < length {
		switch repr[index] {
		case HYPHEN_MINUS_CHAR:
			t = -1
			index += 1
		case PLUS_SIGN_CHAR:
			index += 1
		}
	}

	// 7. An exponent: zero or more digits.
	//    If there is at least one digit, let e be the number formed by interpreting the digits as a base-10 integer;
	//    otherwise, let e be the number 0.
	e := 0

	start_index = index
	for index < length && is_digit(repr[index]) {
		index += 1
	}

	if index > start_index {
		value, err := strconv.ParseInt(string(repr[start_index:index]), 10, 0)
		if err != nil {
			log.Fatal(err)
		}

		e = int(value)
	}

	// Return the number s·(i + f·10^(-d))·10^(te).
	exponent := math.Pow10(t * e)
	fraction := float64(f) * math.Pow10(-1*d)
	return float64(s) * (float64(i) + fraction) * exponent
}

type Tokenizer struct {
	input  []rune
	length int
	index  int
}

type TokenType uint8

// https://www.w3.org/TR/css-syntax-3/#tokenization
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

type Token struct {
	kind TokenType
	// <ident-token>, <function-token>, <at-keyword-token>, <hash-token>, <string-token>, and <url-token> have a value composed of zero or more code points.
	// <delim-token> has a value composed of a single code point.
	value []rune
	// <number-token>, <percentage-token>, and <dimension-token> have a numeric value.
	numeric float64
	// <hash-token> have a type flag set to either "id" or "unrestricted".
	hash_flag HashFlag
	// <number-token> and <dimension-token> additionally have a type flag set to either "integer" or "number".
	type_flag TypeFlag
	// <dimension-token> additionally have a unit composed of one or more code points.
	unit []rune
}

func (t Token) ToString() string {
	var type_string string

	kind := t.kind
	switch kind {
	case IDENT_TOKEN:
		type_string = "IDENT_TOKEN"
	case FUNCTION_TOKEN:
		type_string = "FUNCTION_TOKEN"
	case AT_KEYWORD_TOKEN:
		type_string = "AT_KEYWORD_TOKEN"
	case HASH_TOKEN:
		type_string = "HASH_TOKEN"
	case STRING_TOKEN:
		type_string = "STRING_TOKEN"
	case BAD_STRING_TOKEN:
		type_string = "BAD_STRING_TOKEN"
	case URL_TOKEN:
		type_string = "URL_TOKEN"
	case BAD_URL_TOKEN:
		type_string = "BAD_URL_TOKEN"
	case DELIM_TOKEN:
		type_string = "DELIM_TOKEN"
	case NUMBER_TOKEN:
		type_string = "NUMBER_TOKEN"
	case PERCENTAGE_TOKEN:
		type_string = "PERCENTAGE_TOKEN"
	case DIMENSION_TOKEN:
		type_string = "DIMENSION_TOKEN"
	case WHITESPACE_TOKEN:
		type_string = "WHITESPACE_TOKEN"
	case CDO_TOKEN:
		type_string = "CDO_TOKEN"
	case CDC_TOKEN:
		type_string = "CDC_TOKEN"
	case COLON_TOKEN:
		type_string = "COLON_TOKEN"
	case SEMICOLON_TOKEN:
		type_string = "SEMICOLON_TOKEN"
	case COMMA_TOKEN:
		type_string = "COMMA_TOKEN"
	case OPEN_SQUARE_TOKEN:
		type_string = "OPEN_SQUARE_TOKEN"
	case CLOSE_SQUARE_TOKEN:
		type_string = "CLOSE_SQUARE_TOKEN"
	case OPEN_PAREN_TOKEN:
		type_string = "OPEN_PAREN_TOKEN"
	case CLOSE_PAREN_TOKEN:
		type_string = "CLOSE_PAREN_TOKEN"
	case OPEN_CURLY_TOKEN:
		type_string = "OPEN_CURLY_TOKEN"
	case CLOSE_CURLY_TOKEN:
		type_string = "CLOSE_CURLY_TOKEN"
	case EOF_TOKEN:
		type_string = "EOF_TOKEN"
	}

	// return fmt.Sprintf("<%s> (%s) (%f) (%s)", type_string, string(t.value), t.numeric, string(t.unit))
	str := fmt.Sprintf("<%s>", type_string)

	if kind == IDENT_TOKEN || kind == FUNCTION_TOKEN || kind == AT_KEYWORD_TOKEN || kind == HASH_TOKEN || kind == STRING_TOKEN || kind == URL_TOKEN || kind == DELIM_TOKEN {
		str = str + fmt.Sprintf(" '%s'", string(t.value))
	}

	if kind == NUMBER_TOKEN || kind == PERCENTAGE_TOKEN || kind == DIMENSION_TOKEN {
		str = str + fmt.Sprintf(" %f", t.numeric)
	}

	if kind == PERCENTAGE_TOKEN {
		str = str + "%"
	}

	if kind == DIMENSION_TOKEN {
		str = str + string(t.unit)
	}

	return str
}

func (t *Tokenizer) HasNext() bool {
	return t.index < t.length
}

func (t *Tokenizer) NextToken() Token {
	t.consume_comments()

	t.ConsumeRune()

	char := t.CurrentRune()
	switch {
	case is_whitespace(char):
		// Consume as much whitespace as possible.
		for is_whitespace(t.NextRune()) {
			t.ConsumeRune()
		}
		// Return a <whitespace-token>.
		return Token{kind: WHITESPACE_TOKEN}
	// U+0022 QUOTATION MARK (")
	case char == QUOTATION_MARK_CHAR:
		return t.consume_string_token_default()
	// U+0023 NUMBER SIGN (#)
	case char == NUMBER_SIGN_CHAR:
		return t.consume_hash()
	// U+0027 APOSTROPHE (')
	case char == APOSTROPHE_CHAR:
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
			t.ReconsumeRune()
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
			t.ReconsumeRune()
			return t.consume_numeric_token()
		}

		// Otherwise, if the next 2 input code points are U+002D HYPHEN-MINUS U+003E GREATER-THAN SIGN (->), consume them and return a <CDC-token>.
		if t.NextRune() == HYPHEN_MINUS_CHAR && t.SecondRune() == GREATER_THAN_CHAR {
			t.ConsumeRunes(2)
			return Token{kind: CDC_TOKEN}
		}

		// Otherwise, if the input stream starts with an ident sequence, reconsume the current input code point, consume an ident-like token, and return it.
		if t.starts_with_ident() {
			t.ReconsumeRune()
			return t.consume_ident_like_token()
		}

		// Otherwise, return a <delim-token> with its value set to the current input code point.
		return Token{kind: DELIM_TOKEN, value: []rune{t.CurrentRune()}}
	// U+002E FULL STOP (.)
	case char == FULL_STOP_CHAR:
		// If the input stream starts with a number, reconsume the current input code point, consume a numeric token, and return it.
		if t.starts_with_number() {
			t.ReconsumeRune()
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
		// If the next 3 input code points would start an ident sequence,
		// consume an ident sequence, create an <at-keyword-token> with its value set to the returned value, and return it.
		if are_start_ident(t.NextRune(), t.SecondRune(), t.ThirdRune()) {
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
			t.ReconsumeRune()
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
		t.ReconsumeRune()
		return t.consume_numeric_token()
	// ident-start code point
	case is_ident_start(char):
		// Reconsume the current input code point, consume an ident-like token, and return it.
		t.ReconsumeRune()
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
	if index >= t.length {
		return EOF_CHAR
	}

	return t.input[index]
}

// The last code point to have been consumed.
func (t *Tokenizer) CurrentRune() rune {
	return t.peek_rune(0)
}

// The first code point in the input stream that has not yet been consumed.
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

func (t *Tokenizer) ConsumeRunes(number int) {
	t.index += number
}

func (t *Tokenizer) ConsumeRune() {
	t.ConsumeRunes(1)
}

// Push the current input code point back onto the front of the input stream, so that the next time you are instructed to consume the next input code point,
// it will instead reconsume the current input code point.
func (t *Tokenizer) ReconsumeRune() {
	t.ConsumeRunes(-1)
}

// https://www.w3.org/TR/css-syntax-3/#consume-comment
func (t *Tokenizer) consume_comments() {
	// If the next two input code point are U+002F SOLIDUS (/) followed by a U+002A ASTERISK (*),
	// consume them and all following code points up to and including the first U+002A ASTERISK (*) followed by a U+002F SOLIDUS (/),
	// or up to an EOF code point. Return to the start of this step.

	// If the preceding paragraph ended by consuming an EOF code point, this is a parse error.
	// Return nothing.

	for t.NextRune() == FORWARD_SLASH_CHAR && t.SecondRune() == ASTERISK_CHAR {
		t.ConsumeRunes(2)
		for {
			if t.CurrentRune() == ASTERISK_CHAR && t.NextRune() == FORWARD_SLASH_CHAR {
				t.ConsumeRunes(2)
				break
			} else if t.CurrentRune() == EOF_CHAR {
				log.Fatal("Parse Error: Unexpected EOF when parsing comment")
			} else {
				t.ConsumeRunes(1)
			}
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-string-token
func (t *Tokenizer) consume_string_token(ending rune) Token {
	// It returns either a <string-token> or <bad-string-token>.
	// This algorithm may be called with an ending code point, which denotes the code point that ends the string.

	// Initially create a <string-token> with its value set to the empty string.
	token := Token{kind: STRING_TOKEN, value: nil}

	// Repeatedly consume the next input code point from the stream:
	for {
		t.ConsumeRunes(1)

		char := t.CurrentRune()
		switch {
		// ending code point:
		case char == ending:
			// Return the <string-token>.
			// t.ConsumeRunes(1)
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
			t.ReconsumeRune()
			return Token{kind: BAD_STRING_TOKEN}
		// U+005C REVERSE SOLIDUS (\):
		case char == BACKWARD_SLASH_CHAR:
			// If the next input code point is EOF, do nothing.
			next := t.NextRune()
			if next != EOF_CHAR {
				// Otherwise, if the next input code point is a newline, consume it.
				if is_newline(next) {
					t.ConsumeRunes(2)
				} else {
					// Otherwise, consume an escaped code point and append the returned code point to the <string-token>’s value.
					// @ASSERTED: (the stream starts with a valid escape)
					token.value = append(token.value, t.consume_escaped())
				}
			}
		// anything else: Append the current input code point to the <string-token>’s value.
		default:
			token.value = append(token.value, char)
		}
	}
}

func (t *Tokenizer) consume_string_token_default() Token {
	// If an ending code point is not specified, the current input code point is used.
	return t.consume_string_token(t.CurrentRune())
}

// https://www.w3.org/TR/css-syntax-3/#consume-escaped-code-point
func (t *Tokenizer) consume_escaped() rune {
	// It assumes that the U+005C REVERSE SOLIDUS (\) has already been consumed
	// and that the next input code point has already been verified to be part of a valid escape.
	// It will return a code point.

	// Consume the next input code point.
	t.ConsumeRune()
	char := t.CurrentRune()
	switch {
	// hex digit:
	case is_hex_digit(char):
		// Consume as many hex digits as possible, but no more than 5.
		digits := make([]rune, 0, 6)
		digits = append(digits, char)

		for i := 0; i < 5; i += 1 {
			t.ConsumeRune()
			digit := t.CurrentRune()

			if is_hex_digit(digit) {
				digits = append(digits, digit)
			} else {
				t.ReconsumeRune()
				break
			}
		}

		// @Assertion: Note that this means 1-6 hex digits have been consumed in total.
		// If the next input code point is whitespace, consume it as well.
		if is_whitespace(t.NextRune()) {
			t.ConsumeRune()
		}

		// Interpret the hex digits as a hexadecimal number.
		value, err := strconv.ParseInt(string(digits[:]), 16, 64)
		if err != nil {
			log.Fatal(err)
		}

		// If this number is zero,
		// or is for a surrogate,
		// or is greater than the maximum allowed code point,
		// return U+FFFD REPLACEMENT CHARACTER (�).
		// @NOTE: We don't support utf16 encoding so there can be no surrogate code points
		if value == 0 || value > MAX_CODE_POINT {
			return REPLACEMENT_CHAR
		}

		// Otherwise, return the code point with that value.
		return rune(value)
	// EOF: This is a parse error. Return U+FFFD REPLACEMENT CHARACTER (�).
	case char == EOF_CHAR:
		fmt.Println("Parse Error: Unexpected EOF when parsing escape")
		return REPLACEMENT_CHAR
	default:
		return char
	}
}

func (t *Tokenizer) consume_hash() Token {
	// If the next input code point is an ident code point or the next two input code points are a valid escape
	if is_ident(t.NextRune()) || are_valid_escape(t.NextRune(), t.SecondRune()) {
		// Create a <hash-token>.
		token := Token{kind: HASH_TOKEN}
		// If the next 3 input code points would start an ident sequence, set the <hash-token>’s type flag to "id".
		if are_start_ident(t.NextRune(), t.SecondRune(), t.ThirdRune()) {
			token.hash_flag = HASH_ID
		}
		// Consume an ident sequence, and set the <hash-token>’s value to the returned string.
		token.value = t.consume_ident_sequence()
		// Return the <hash-token>.
		return token
	}

	// Otherwise, return a <delim-token> with its value set to the current input code point.
	token := Token{kind: DELIM_TOKEN, value: []rune{t.CurrentRune()}}
	return token
}

// https://www.w3.org/TR/css-syntax-3/#consume-name
func (t *Tokenizer) consume_ident_sequence() []rune {
	// Returns a string containing the largest name that can be formed from adjacent code points in the stream, starting from the first.

	// Let result initially be an empty string.
	var result []rune
	// Repeatedly consume the next input code point from the stream:
	for {
		t.ConsumeRunes(1)

		char := t.CurrentRune()
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
			t.ReconsumeRune()
			return result
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-numeric-token
func (t *Tokenizer) consume_numeric_token() Token {
	// Returns either a <number-token>, <percentage-token>, or <dimension-token>.

	// Consume a number and let number be the result.
	number, type_flag := t.consume_number()
	// If the next 3 input code points would start an ident sequence, then:
	if are_start_ident(t.NextRune(), t.SecondRune(), t.ThirdRune()) {
		// 1. Create a <dimension-token> with the same value and type flag as number, and a unit set initially to the empty string.
		token := Token{kind: DIMENSION_TOKEN, numeric: number, type_flag: type_flag, unit: nil}
		// 2. Consume an ident sequence. Set the <dimension-token>’s unit to the returned value.
		token.unit = t.consume_ident_sequence()
		// 3. Return the <dimension-token>.
		return token
	}

	// Otherwise, if the next input code point is U+0025 PERCENTAGE SIGN (%), consume it.
	// Create a <percentage-token> with the same value as number, and return it.
	if t.NextRune() == PERCENT_SIGN_CHAR {
		t.ConsumeRune()
		return Token{kind: PERCENTAGE_TOKEN, numeric: number}
	}

	// Otherwise, create a <number-token> with the same value and type flag as number, and return it.
	return Token{kind: NUMBER_TOKEN, numeric: number, type_flag: type_flag}
}

// https://www.w3.org/TR/css-syntax-3/#consume-number
func (t *Tokenizer) consume_number() (float64, TypeFlag) {
	// Returns a numeric value, and a type which is either "integer" or "number".

	// 1. Initially set type to "integer". Let repr be the empty string.
	type_flag := TYPE_INTEGER
	var repr []rune

	// 2. If the next input code point is U+002B PLUS SIGN (+) or U+002D HYPHEN-MINUS (-), consume it and append it to repr.
	next := t.NextRune()
	if next == PLUS_SIGN_CHAR || next == HYPHEN_MINUS_CHAR {
		t.ConsumeRune()
		repr = append(repr, next)
	}

	// 3. While the next input code point is a digit, consume it and append it to repr.
	for is_digit(t.NextRune()) {
		t.ConsumeRune()
		repr = append(repr, t.CurrentRune())
	}

	// 4. If the next 2 input code points are U+002E FULL STOP (.) followed by a digit, then:
	if t.NextRune() == FULL_STOP_CHAR && is_digit(t.SecondRune()) {
		// Consume them.
		// Append them to repr.
		repr = append(repr, t.NextRune(), t.SecondRune())
		t.ConsumeRunes(2)
		// Set type to "number".
		type_flag = TYPE_NUMBER
		// While the next input code point is a digit, consume it and append it to repr.
		for is_digit(t.NextRune()) {
			t.ConsumeRune()
			repr = append(repr, t.CurrentRune())
		}
	}

	// 5. If the next 2 or 3 input code points are U+0045 LATIN CAPITAL LETTER E (E) or U+0065 LATIN SMALL LETTER E (e),
	//    optionally followed by U+002D HYPHEN-MINUS (-) or U+002B PLUS SIGN (+), followed by a digit
	first, second, third := t.NextRune(), t.SecondRune(), t.ThirdRune()
	target := 2
	if first == UPPER_E_CHAR || first == LOWER_E_CHAR {
		if second == HYPHEN_MINUS_CHAR || second == PLUS_SIGN_CHAR {
			target = 3
		}

		if (target == 2 && is_digit(second)) || (target == 3 && is_digit(third)) {
			// 5.1 Consume them.
			t.ConsumeRunes(target)
			// 5.2 Append them to repr.
			repr = append(repr, first, second)
			if target == 3 {
				repr = append(repr, third)
			}
			// 5.3 Set type to "number".
			type_flag = TYPE_NUMBER
			// 5.4 While the next input code point is a digit, consume it and append it to repr.
			for is_digit(t.NextRune()) {
				t.ConsumeRune()
				repr = append(repr, t.CurrentRune())
			}
		}
	}
	// 6. Convert repr to a number, and set the value to the returned value.
	value := convert_string_to_number(repr)
	// 7. Return value and type.
	return value, type_flag
}

// https://www.w3.org/TR/css-syntax-3/#consume-ident-like-token
func (t *Tokenizer) consume_ident_like_token() Token {
	// Returns an <ident-token>, <function-token>, <url-token>, or <bad-url-token>.

	// Consume an ident sequence, and let string be the result.
	str := t.consume_ident_sequence()
	// If string’s value is an ASCII case-insensitive match for "url", and the next input code point is U+0028 LEFT PARENTHESIS ((), consume it.
	if strings.EqualFold(string(str), "url") && t.NextRune() == OPEN_PAREN_CHAR {
		t.ConsumeRune()
		// While the next two input code points are whitespace, consume the next input code point.
		for is_whitespace(t.NextRune()) && is_whitespace(t.SecondRune()) {
			t.ConsumeRune()
		}
		// If the next one or two input code points are U+0022 QUOTATION MARK ("), U+0027 APOSTROPHE ('),
		// or whitespace followed by U+0022 QUOTATION MARK (") or U+0027 APOSTROPHE ('),
		// then create a <function-token> with its value set to string and return it.
		first, second := t.NextRune(), t.SecondRune()
		switch {
		case first == QUOTATION_MARK_CHAR:
			fallthrough
		case first == APOSTROPHE_CHAR:
			fallthrough
		case is_whitespace(first) && second == QUOTATION_MARK_CHAR:
			fallthrough
		case is_whitespace(first) && second == APOSTROPHE_CHAR:
			return Token{kind: FUNCTION_TOKEN, value: str}
		// Otherwise, consume a url token, and return it.
		default:
			return t.consume_url_token()
		}
	}

	// Otherwise, if the next input code point is U+0028 LEFT PARENTHESIS ((), consume it.
	if t.NextRune() == OPEN_PAREN_CHAR {
		t.ConsumeRune()
		// Create a <function-token> with its value set to string and return it.
		return Token{kind: FUNCTION_TOKEN, value: str}
	}

	// Otherwise, create an <ident-token> with its value set to string and return it.
	return Token{kind: IDENT_TOKEN, value: str}
}

// https://www.w3.org/TR/css-syntax-3/#consume-url-token
func (t *Tokenizer) consume_url_token() Token {
	// Returns either a <url-token> or a <bad-url-token>.

	// @NOTE: This algorithm assumes that the initial "url(" has already been consumed.
	//		  This algorithm also assumes that it’s being called to consume an "unquoted" value, like url(foo).
	//        A quoted value, like url("foo"), is parsed as a <function-token>.
	//        Consume an ident-like token automatically handles this distinction; this algorithm shouldn’t be called directly otherwise.

	// 1. Initially create a <url-token> with its value set to the empty string.
	token := Token{kind: URL_TOKEN, value: nil}
	// 2. Consume as much whitespace as possible.
	for is_whitespace(t.NextRune()) {
		t.ConsumeRune()
	}
	// 3. Repeatedly consume the next input code point from the stream:
	for {
		t.ConsumeRune()

		char := t.CurrentRune()
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
				t.ConsumeRune()
			}
			// If the next input code point is U+0029 RIGHT PARENTHESIS ())
			switch t.NextRune() {
			case CLOSE_PAREN_CHAR:
				t.ConsumeRune()
				return token
			// or EOF, consume it and return the <url-token> (if EOF was encountered, this is a parse error);
			case EOF_CHAR:
				fmt.Println("Parse Error: Unexpected EOF while parsing URL")
				t.ConsumeRune()
				return token
			// otherwise, consume the remnants of a bad url, create a <bad-url-token>, and return it.
			default:
				t.consume_bad_url_remnants()
				return Token{kind: BAD_URL_TOKEN}
			}
		// U+0022 QUOTATION MARK ("), U+0027 APOSTROPHE ('), U+0028 LEFT PARENTHESIS ((), non-printable code point
		case char == QUOTATION_MARK_CHAR:
			fallthrough
		case char == APOSTROPHE_CHAR:
			fallthrough
		case char == OPEN_PAREN_CHAR:
			fallthrough
		case is_non_printable(char):
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
// https://www.w3.org/TR/css-syntax-3/#consume-remnants-of-bad-url
func (t *Tokenizer) consume_bad_url_remnants() {
	// Returns nothing; its sole use is to consume enough of the input stream to reach a recovery point where normal tokenizing can resume.

	// Repeatedly consume the next input code point from the stream:
	for {
		t.ConsumeRune()

		char := t.CurrentRune()
		switch {
		// U+0029 RIGHT PARENTHESIS ()), EOF
		case char == CLOSE_PAREN_CHAR:
			fallthrough
		case char == EOF_CHAR:
			return
		// the input stream starts with a valid escape
		case t.starts_with_valid_escape():
			// Consume an escaped code point.
			// @NOTE: This allows an escaped right parenthesis ("\)") to be encountered without ending the <bad-url-token>.
			//        This is otherwise identical to the "anything else" clause.
			t.consume_escaped()
		}
	}
}
