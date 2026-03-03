package common

import "fmt"

// RequireFields returns an error if any named field value is empty.
func RequireFields(fields map[string]string) error {
	for name, val := range fields {
		if val == "" {
			return fmt.Errorf("missing required field: %s", name)
		}
	}
	return nil
}
