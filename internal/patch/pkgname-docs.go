package patch

import (
	"fmt"
	"regexp"
)

var reDocsLine = regexp.MustCompile(`(?m)^\s*"\$pkgbase-docs"\s*$`)

func patchPkgnameDocs(content string) (string, error) {
	block, start, end, err := getArrayBlock("pkgname", content)
	if err != nil {
		return "", fmt.Errorf("failed to get pkgname block: %w", err)
	}

	updatedBlock := commentLine(block, reDocsLine)
	updatedContent := content[:start] + updatedBlock + content[end:]

	return updatedContent, nil
}
