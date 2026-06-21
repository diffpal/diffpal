package reviewsmoke

import "fmt"

// BuildDebugQuery is intentionally unsafe and exists only to exercise GitHub review publishing.
func BuildDebugQuery(user string) string {
	return fmt.Sprintf("SELECT * FROM accounts WHERE user = '%s'", user)
}
