package common

import (
	"regexp"
	"strings"
)

var nonDigitOrPlus = regexp.MustCompile(`[^\d+]`)

// NormalizePhone strips formatting and ensures E.164-style leading '+'.
func NormalizePhone(p string) string {
	p = strings.TrimSpace(p)
	p = nonDigitOrPlus.ReplaceAllString(p, "")
	if !strings.HasPrefix(p, "+") {
		p = "+" + p
	}
	return p
}
