package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}$`)

// validateTunnelName checks that a tunnel name is safe for use in filenames and config directives.
func validateTunnelName(name string) error {
	if !validNameRe.MatchString(name) {
		return fmt.Errorf("invalid tunnel name %q: must match [a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}", name)
	}
	return nil
}

// validateConfigName checks that a config file name is safe (no path traversal, must end with .ovpn).
func validateConfigName(name string) error {
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("invalid config name %q: path traversal not allowed", name)
	}
	if filepath.Base(name) != name {
		return fmt.Errorf("invalid config name %q: must be a simple filename", name)
	}
	if !strings.HasSuffix(name, ".ovpn") {
		return fmt.Errorf("invalid config name %q: must end with .ovpn", name)
	}
	return nil
}
