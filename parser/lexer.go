package parser

// scanner walks runes within a single line.
type scanner struct {
	runes []rune
	pos   int
}

func newScanner(s string) *scanner {
	return &scanner{runes: []rune(s)}
}

func (s *scanner) done() bool {
	return s.pos >= len(s.runes)
}

func (s *scanner) peek() rune {
	if s.done() {
		return 0
	}
	return s.runes[s.pos]
}

func (s *scanner) next() rune {
	r := s.runes[s.pos]
	s.pos++
	return r
}

func (s *scanner) skipWS() {
	for !s.done() && (s.peek() == ' ' || s.peek() == '\t') {
		s.pos++
	}
}

// isEmojiStart returns true if r is the start of an emoji codepoint.
// Covers the major Unicode emoji blocks.
func isEmojiStart(r rune) bool {
	return (r >= 0x1F300 && r <= 0x1FAFF) || // Misc Symbols and Pictographs, Emoticons, Transport, etc.
		(r >= 0x2600 && r <= 0x26FF) || // Miscellaneous Symbols
		(r >= 0x2700 && r <= 0x27BF) || // Dingbats
		(r >= 0x1F000 && r <= 0x1F02F) || // Mahjong Tiles
		(r >= 0x1F0A0 && r <= 0x1F0FF) || // Playing Cards
		(r >= 0x1F100 && r <= 0x1F2FF) || // Enclosed Alphanumeric Supplement
		r == 0x2049 || r == 0x203C // ‼️ ⁉️
}

// isEmojiContinue returns true for codepoints that extend an emoji cluster.
func isEmojiContinue(r rune) bool {
	return r == 0x200D || // Zero Width Joiner
		(r >= 0xFE00 && r <= 0xFE0F) || // Variation Selectors
		(r >= 0x1F3FB && r <= 0x1F3FF) || // Skin Tone Modifiers
		r == 0x20E3 // Combining Enclosing Keycap
}

// readEmoji reads a single emoji cluster: base codepoint + any modifiers/ZWJ sequences.
func (s *scanner) readEmoji() string {
	if s.done() || !isEmojiStart(s.peek()) {
		return ""
	}
	start := s.pos
	s.next() // base emoji
	for !s.done() {
		r := s.peek()
		if isEmojiContinue(r) {
			s.next()
			// After ZWJ, consume the next emoji base too
			if r == 0x200D && !s.done() && isEmojiStart(s.peek()) {
				s.next()
			}
		} else {
			break
		}
	}
	return string(s.runes[start:s.pos])
}

// isIdentStart returns true for valid identifier-start characters (not emoji).
func isIdentStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

// isIdentCont returns true for valid identifier continuation characters.
func isIdentCont(r rune) bool {
	return isIdentStart(r) || (r >= '0' && r <= '9') || r == '-' || r == '.'
}

// readIdent reads an identifier token.
func (s *scanner) readIdent() string {
	start := s.pos
	for !s.done() && isIdentCont(s.peek()) {
		s.pos++
	}
	return string(s.runes[start:s.pos])
}
