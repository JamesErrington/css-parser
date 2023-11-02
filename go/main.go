package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"unicode"
)

const (
	// The greatest code point defined by Unicode: U+10FFFF.
	MAX_CODE_POINT int64 = 0x10FFFF
)

const (
	CHAR_EOF             rune = 0
	CHAR_CONTROL         rune = '\u0080'
	CHAR_LOW_LINE        rune = '\u005F'
	CHAR_HYPHEN_MINUS    rune = '\u002D'
	CHAR_LINE_FEED       rune = '\u000A'
	CHAR_TAB             rune = '\u0009'
	CHAR_SPACE           rune = '\u0020'
	CHAR_FORWARD_SLASH   rune = '\u002F'
	CHAR_BACKWARD_SLASH  rune = '\u005C'
	CHAR_ASTERISK        rune = '\u002A'
	CHAR_CARRIAGE_RETURN rune = '\u000D'
	CHAR_FORM_FEED       rune = '\u000C'
	CHAR_NULL            rune = '\u0000'
	CHAR_REPLACEMENT     rune = '\uFFFD'
	CHAR_QUOTATION_MARK  rune = '\u0022'
)

type TokenType uint8

// https://www.w3.org/TR/css-syntax-3/#tokenization
const (
	TOKEN_IDENT TokenType = iota
	TOKEN_FUNCTION
	TOKEN_KEYWORD_AT
	TOKEN_HASH
	TOKEN_STRING
	TOKEN_BAD_STRING
	TOKEN_URL
	TOKEN_BAD_URL
	TOKEN_DELIM
	TOKEN_NUMBER
	TOKEN_PERCENTAGE
	TOKEN_DIMENSION
	TOKEN_WHITESPACE
	TOKEN_CDO
	TOKEN_CDC
	TOKEN_COLON
	TOKEN_SEMICOLON
	TOKEN_COMMA
	TOKEN_OPEN_SQUARE
	TOKEN_CLOSE_SQUARE
	TOKEN_OPEN_PAREN
	TOKEN_CLOSE_PAREN
	TOKEN_OPEN_CURLY
	TOKEN_CLOSE_CURLY
)

type Token struct {
	kind  TokenType
	value string
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
	return char > CHAR_CONTROL
}

func is_ident_start(char rune) bool {
	// A letter,
	// a non-ASCII code point,
	// or U+005F LOW LINE (_).
	return is_letter(char) || is_non_ascii(char) || char == CHAR_LOW_LINE
}

func is_ident(char rune) bool {
	// An ident-start code point,
	// a digit,
	// or U+002D HYPHEN-MINUS (-).
	return is_ident_start(char) || is_digit(char) || char == CHAR_HYPHEN_MINUS
}

func is_newline(char rune) bool {
	// U+000A LINE FEED

	// Note that U+000D CARRIAGE RETURN and U+000C FORM FEED are not included in this definition,
	// as they are converted to U+000A LINE FEED during preprocessing
	return char == CHAR_LINE_FEED
}

func is_whitespace(char rune) bool {
	// newline,
	// U+0009 CHARACTER TABULATION,
	// or U+0020 SPACE.
	return is_newline(char) || char == CHAR_TAB || char == CHAR_SPACE
}

func is_surrogate(char rune) bool {
	return unicode.Is(unicode.Cs, char)
}

type Scanner struct {
	reader  *bufio.Reader
	current rune
}

// https://www.w3.org/TR/css-syntax-3/#input-preprocessing
func (s *Scanner) read_filtered_rune() (rune, error) {
	char, _, err := s.reader.ReadRune()
	if err != nil {
		return CHAR_EOF, err
	}

	switch char {
	// Replace any U+000C FORM FEED (FF) code points by a single U+000A LINE FEED (LF) code point.
	case CHAR_FORM_FEED:
		return CHAR_LINE_FEED, nil
	// Replace any U+000D CARRIAGE RETURN (CR) code points,
	// or pairs of U+000D CARRIAGE RETURN (CR) followed by U+000A LINE FEED (LF)
	// by a single U+000A LINE FEED (LF) code point.
	case CHAR_LINE_FEED:
		next, _, err := s.reader.ReadRune()
		if err != nil {
			return CHAR_EOF, err
		}

		if next == CHAR_LINE_FEED {
			return CHAR_LINE_FEED, nil
		} else {
			err = s.reader.UnreadRune() // Unread the Rune that might have been a U+000A LINE FEED (LF)
			if err != nil {
				return CHAR_EOF, err
			} else {
				return CHAR_LINE_FEED, nil
			}
		}
	// Replace any U+0000 NULL with U+FFFD REPLACEMENT CHARACTER (�).
	case CHAR_NULL:
		return CHAR_REPLACEMENT, nil
	}

	// Replace any surrogate code points in input with U+FFFD REPLACEMENT CHARACTER (�).
	if is_surrogate(char) {
		return CHAR_REPLACEMENT, nil
	}

	return char, nil
}

func (s *Scanner) ReadRune() {
	char, err := s.read_filtered_rune()
	if err != nil {
		s.current = CHAR_EOF
	} else {
		s.current = char
	}
}

func (s *Scanner) UnreadRune() {
	err := s.reader.UnreadRune()
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Scanner) PeekRune() rune {
	char, err := s.read_filtered_rune()
	if err != nil {
		return CHAR_EOF
	}

	s.UnreadRune()
	return char
}

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

	scanner := Scanner{reader: bufio.NewReader(file), current: CHAR_EOF}
	// https://www.w3.org/TR/css-syntax-3/#consume-token
	for {
		scanner.ReadRune()
		if scanner.current == CHAR_EOF {
			break
		}

		token := consume_token(&scanner)
		fmt.Printf("%+v\n", token)
	}

}

func consume_token(scanner *Scanner) Token {
	consume_comments(scanner)

	// Consume as much whitespace as possible. Return a <whitespace-token>.
	if is_whitespace(scanner.current) {
		fmt.Println("Found whitespace")
		for is_whitespace(scanner.current) {
			scanner.ReadRune()
		}

		return Token{kind: TOKEN_WHITESPACE}
	}

	if scanner.current == CHAR_QUOTATION_MARK {
		fmt.Println("Found string")
		return consume_string_token_default(scanner)
	}

	return Token{kind: TOKEN_DELIM, value: string(scanner.current)}
}

// https://www.w3.org/TR/css-syntax-3/#consume-comment
// Scanner has been primed by calling ReadRune() before entering this function
func consume_comments(scanner *Scanner) {
	// If the next two input code point are U+002F SOLIDUS (/) followed by a U+002A ASTERISK (*),
	// consume them and all following code points up to and including the first U+002A ASTERISK (*) followed by a U+002F SOLIDUS (/),
	// or up to an EOF code point. Return to the start of this step.

	// If the preceding paragraph ended by consuming an EOF code point, this is a parse error.

	for scanner.current == CHAR_FORWARD_SLASH && scanner.PeekRune() == CHAR_ASTERISK {
		fmt.Println("Found comment")
		scanner.ReadRune() // Consume the '*'
		for {
			scanner.ReadRune()
			if scanner.current == CHAR_ASTERISK && scanner.PeekRune() == CHAR_FORWARD_SLASH {
				scanner.ReadRune() // Consume the '/'
				scanner.ReadRune() // Prime for the next loop iteration
				break
			} else if scanner.current == CHAR_EOF {
				log.Fatal("Unexpected EOF when parsing comment")
			}
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-escaped-code-point
func consume_escaped(scanner *Scanner) rune {
	// Assumes U+005C REVERSE SOLIDUS (\) has already been consumed
	// and that the next input code point has already been verified to be part of a valid escape.
	// It will return a code point.

	// hex digit:
	if is_hex_digit(scanner.current) {
		digits := make([]rune, 0, 6)
		digits = append(digits, scanner.current)

		// Consume as many hex digits as possible, but no more than 5.
		for i := 0; i < 5; i += 1 {
			scanner.ReadRune()
			if is_hex_digit(scanner.current) {
				digits = append(digits, scanner.current)
			} else {
				break
			}
		}

		// If the next input code point is whitespace, consume it as well.
		if is_whitespace(scanner.current) {
			scanner.ReadRune()
		}
		scanner.UnreadRune()

		// Interpret the hex digits as a hexadecimal number.
		value, err := strconv.ParseInt(string(digits[:]), 16, 64)
		if err != nil {
			log.Fatal(err)
		}

		// If this number is zero,
		// or is for a surrogate,
		// or is greater than the maximum allowed code point,
		// return U+FFFD REPLACEMENT CHARACTER (�).
		if value == 0 || is_surrogate(rune(value)) || value > MAX_CODE_POINT {
			return CHAR_REPLACEMENT
		}

		// Otherwise, return the code point with that value.
		return rune(value)
	}

	// EOF: This is a parse error. Return U+FFFD REPLACEMENT CHARACTER (�).
	if scanner.current == CHAR_EOF {
		fmt.Println("Parse Error: Unexpected EOF when parsing escape")
		return CHAR_REPLACEMENT
	}

	return scanner.current
}

// https://www.w3.org/TR/css-syntax-3/#consume-string-token
func consume_string_token(scanner *Scanner, ending rune) Token {
	// Initially create a <string-token> with its value set to the empty string.
	token := Token{kind: TOKEN_STRING, value: ""}

	// Repeatedly consume the next input code point from the stream:
	for {
		scanner.ReadRune()

		switch {
		// ending code point: Return the <string-token>.
		case scanner.current == ending:
			return token
		// EOF: This is a parse error. Return the <string-token>.
		case scanner.current == CHAR_EOF:
			fmt.Println("Parse Error: Unexpected EOF when parsing string")
			return token
		// newline: This is a parse error. Reconsume the current input code point, create a <bad-string-token>, and return it.
		case is_newline(scanner.current):
			fmt.Println("Parse Error: Unexpected newline when parsing string")
			scanner.UnreadRune()
			return Token{kind: TOKEN_BAD_STRING}
		// U+005C REVERSE SOLIDUS (\):
		case scanner.current == CHAR_BACKWARD_SLASH:
			scanner.ReadRune()
			// If the next input code point is EOF, do nothing.
			if scanner.current != CHAR_EOF {
				// Otherwise, if the next input code point is a newline, consume it.
				if is_newline(scanner.current) {
					// noop, since we have already consumed the newline
				} else {
					// Otherwise, consume an escaped code point and append the returned code point to the <string-token>’s value.
					token.value += string(consume_escaped(scanner))
				}
			}
		// anything else: Append the current input code point to the <string-token>’s value.
		default:
			token.value += string(scanner.current)
		}
	}
}

func consume_string_token_default(scanner *Scanner) Token {
	return consume_string_token(scanner, scanner.current)
}
