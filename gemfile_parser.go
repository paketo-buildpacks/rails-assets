package railsassets

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

// GemfileParser parses the Gemfile to confirm that the application is using
// Rails.
type GemfileParser struct{}

// NewGemfileParser initializes a GemfileParser instance.
func NewGemfileParser() GemfileParser {
	return GemfileParser{}
}

// Parse scans the Gemfile to find the "rails" gem.
func (p GemfileParser) Parse(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to parse Gemfile: %w", err)
	}
	defer file.Close()

	quotes := `["']`
	railsRe := regexp.MustCompile(fmt.Sprintf(`gem %srails%s`, quotes, quotes))
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := []byte(scanner.Text())
		if railsRe.Match(line) {
			return true, nil
		}
	}

	return false, nil
}
