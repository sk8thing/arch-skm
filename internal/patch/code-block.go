package patch

import (
	"fmt"
	"regexp"
)

func getArrayBlock(varName string, content string) (block string, start int, end int, err error) {
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(varName) + `=`)
	return getDelimitedBlock(re, content, '(', ')')
}

func getFunctionBlock(funcName string, content string) (block string, start int, end int, err error) {
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(funcName) + `\(\)\s*`)
	return getDelimitedBlock(re, content, '{', '}')
}

func getDelimitedBlock(startRe *regexp.Regexp, content string, openChar, closeChar byte) (block string, start int, end int, err error) {
	loc := startRe.FindStringIndex(content)
	if loc == nil {
		return "", -1, -1, fmt.Errorf("block start not found")
	}

	start = loc[0]

	open := -1
	for i := loc[1]; i < len(content); i++ {
		if content[i] == openChar {
			open = i
			break
		}
	}
	if open == -1 {
		return "", start, -1, fmt.Errorf("no opening delimiter found")
	}

	depth := 0
	end = -1

	for i := open; i < len(content); i++ {
		switch content[i] {
		case openChar:
			depth++
		case closeChar:
			depth--
		}

		if depth == 0 {
			end = i
			break
		}
	}

	if end == -1 {
		return "", start, -1, fmt.Errorf("no matching closing delimiter found")
	}

	block = content[start : end+1]
	return block, start, end + 1, nil
}
