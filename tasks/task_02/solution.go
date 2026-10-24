package main

func rotateRunes(s string, shift int) string {
	symbols := []rune(s)

	if len(symbols) == 0 {
		return ""
	}

	shift %= len(symbols)

	if shift < 0 {
		shift += len(symbols)
	}

	if shift == 0 || len(symbols) == 1 {
		return string(symbols) 
	}

	return string(append(symbols[shift:], symbols[:shift]...))


}

