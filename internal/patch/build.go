package patch

import (
	"fmt"
	"regexp"
)

var (
	reMakeDocs = regexp.MustCompile(`(?m)^\s*make\s+htmldocs.*$`)
	rePidDocs  = regexp.MustCompile(`(?m)^\s*local\s+pid_docs=.*$`)
	reWait     = regexp.MustCompile(`(?m)^\s*wait\s+"\$\{pid_docs\}".*$`)
)

func patchBuildFunction(content string) (string, error) {
	block, start, end, err := getFunctionBlock("build", content)
	if err != nil {
		return "", fmt.Errorf("failed to read build function block: %w", err)
	}

	block = commentLine(block, reMakeDocs)
	block = commentLine(block, rePidDocs)
	block = commentLine(block, reWait)

	updatedContent := content[:start] + block + content[end:]
	return updatedContent, nil
}
