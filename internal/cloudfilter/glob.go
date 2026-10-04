package cloudfilter

import (
	"fmt"
	"unicode/utf8"
)

type globRange struct{ low, high rune }
type globToken struct {
	kind    byte
	literal rune
	negated bool
	any     bool
	ranges  []globRange
}

// compileGlob follows CPython fnmatch.translate's bracket construction. A
// rune-aware dynamic matcher supplies its full-string DOTALL behavior without
// filesystem separators, backslash escaping or atomic-regexp syntax.
func compileGlob(pattern string) ([]globToken, error) {
	if !utf8.ValidString(pattern) {
		return nil, fmt.Errorf("identifier pattern must be valid UTF-8")
	}
	runes := []rune(pattern)
	tokens := make([]globToken, 0, len(runes))
	for i := 0; i < len(runes); {
		c := runes[i]
		i++
		switch c {
		case '*':
			tokens = append(tokens, globToken{kind: '*'})
			for i < len(runes) && runes[i] == '*' {
				i++
			}
		case '?':
			tokens = append(tokens, globToken{kind: '?'})
		case '[':
			j := i
			if j < len(runes) && runes[j] == '!' {
				j++
			}
			if j < len(runes) && runes[j] == ']' {
				j++
			}
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j == len(runes) {
				tokens = append(tokens, globToken{kind: 'l', literal: '['})
				continue
			}
			tokens = append(tokens, compileClass(runes[i:j]))
			i = j + 1
		default:
			tokens = append(tokens, globToken{kind: 'l', literal: c})
		}
	}
	return tokens, nil
}

type classRune struct {
	value     rune
	separator bool
}

func compileClass(stuff []rune) globToken {
	chunks := [][]rune{}
	containsHyphen := false
	for _, c := range stuff {
		if c == '-' {
			containsHyphen = true
			break
		}
	}
	if !containsHyphen {
		chunks = append(chunks, stuff)
	} else {
		start, search := 0, 1
		if stuff[0] == '!' {
			search = 2
		}
		for {
			position := -1
			for k := search; k < len(stuff); k++ {
				if stuff[k] == '-' {
					position = k
					break
				}
			}
			if position == -1 {
				break
			}
			chunks = append(chunks, stuff[start:position])
			start, search = position+1, position+3
		}
		if start < len(stuff) {
			chunks = append(chunks, stuff[start:])
		} else {
			last := len(chunks) - 1
			chunks[last] = append(append([]rune{}, chunks[last]...), '-')
		}
		// Invalid descending ranges are removed exactly as in translate,
		// retaining the surrounding chunks and their literal characters.
		for k := len(chunks) - 1; k > 0; k-- {
			left, right := chunks[k-1], chunks[k]
			if left[len(left)-1] > right[0] {
				merged := append(append([]rune{}, left[:len(left)-1]...), right[1:]...)
				chunks[k-1] = merged
				chunks = append(chunks[:k], chunks[k+1:]...)
			}
		}
	}
	characters := []classRune{}
	for i, chunk := range chunks {
		if i > 0 {
			characters = append(characters, classRune{value: '-', separator: true})
		}
		for _, c := range chunk {
			characters = append(characters, classRune{value: c})
		}
	}
	token := globToken{kind: 'c'}
	if len(characters) == 0 {
		return token
	}
	if characters[0].value == '!' {
		token.negated = true
		characters = characters[1:]
		if len(characters) == 0 {
			token.any = true
			return token
		}
	}
	for i := 0; i < len(characters); {
		if i+2 < len(characters) && characters[i+1].separator {
			token.ranges = append(token.ranges, globRange{low: characters[i].value, high: characters[i+2].value})
			i += 3
		} else {
			token.ranges = append(token.ranges, globRange{low: characters[i].value, high: characters[i].value})
			i++
		}
	}
	return token
}

func (token globToken) accepts(c rune) bool {
	switch token.kind {
	case '?':
		return true
	case 'l':
		return token.literal == c
	case 'c':
		if token.any {
			return true
		}
		matched := false
		for _, interval := range token.ranges {
			if interval.low <= c && c <= interval.high {
				matched = true
				break
			}
		}
		if token.negated {
			return !matched
		}
		return matched
	}
	return false
}

func matchGlob(tokens []globToken, value string, guard func() error) (bool, error) {
	runes := []rune(value)
	previous := make([]bool, len(runes)+1)
	previous[0] = true
	for _, token := range tokens {
		if err := check(guard); err != nil {
			return false, err
		}
		next := make([]bool, len(previous))
		if token.kind == '*' {
			next[0] = previous[0]
		}
		for j, c := range runes {
			if j%256 == 0 {
				if err := check(guard); err != nil {
					return false, err
				}
			}
			if token.kind == '*' {
				next[j+1] = previous[j+1] || next[j]
			} else {
				next[j+1] = previous[j] && token.accepts(c)
			}
		}
		previous = next
	}
	return previous[len(runes)], check(guard)
}
