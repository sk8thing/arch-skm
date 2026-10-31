package patch

import "regexp"

var leadingWhitespace = regexp.MustCompile(`^(\s*)`)

func commentLine(block string, re *regexp.Regexp) string {
	return re.ReplaceAllStringFunc(block, func(line string) string {
		leading := leadingWhitespace.FindString(line)
		return leading + "# " + line[len(leading):]
	})
}
