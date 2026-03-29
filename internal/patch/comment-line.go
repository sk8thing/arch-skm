package patch

import "regexp"

func commentLine(block string, re *regexp.Regexp) string {
	return re.ReplaceAllStringFunc(block, func(line string) string {
		wsRe := regexp.MustCompile(`^(\s*)`)
		leading := wsRe.FindString(line)
		return leading + "# " + line[len(leading):]
	})
}
