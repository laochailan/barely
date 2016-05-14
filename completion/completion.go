// Package completion implements functions to complete a partial path string to
// a full path based on the file system contents. This allows implementing
// shell-like completion features.
package completion

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Query takes the last part of str and tries to complete it to a proper path.
//
// e.g. Query(":a /tmp/t") will return {":a /tmp/test", ":a /tmp/tested"} if
// the files "/tmp/test" and "/tmp/tested" exist.
//
func Query(str string) (matches []string) {
	lastSpace := strings.LastIndex(str, " ")

	path := str[lastSpace+1:]

	// Expand ~/
	if user, err := user.Current(); err == nil && path[:2] == "~/" {
		path = filepath.Join(user.HomeDir, path[2:])
	}

	matches, err := filepath.Glob(path + "*")
	// Glob only returns an error when the pattern is malformed. That should not happen.
	if err != nil {
		panic(err)
	}

	for i := range matches {
		// If the file ends in a directory we want to end it with slash
		// so the user can make it more specific without entering the
		// slash themselves.
		info, err := os.Stat(matches[i])
		if err == nil && info.Mode()&os.ModeDir != 0 {
			matches[i] += string(os.PathSeparator)
		}

		matches[i] = str[:lastSpace+1] + matches[i]
	}

	return matches
}
