package versioning

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	Name    = "oncode-tool"
	Major   = 1
	Minor   = 0
	Current = "oncode-tool/1.0"
)

type Version struct {
	Major int
	Minor int
}

func (v Version) String() string {
	return fmt.Sprintf("%s/%d.%d", Name, v.Major, v.Minor)
}

func Parse(s string) (Version, error) {
	s = strings.TrimSpace(s)
	prefix := Name + "/"
	if !strings.HasPrefix(s, prefix) {
		return Version{}, fmt.Errorf("protocol version %q: %w", s, errUnsupported)
	}
	rest := s[len(prefix):]
	parts := strings.Split(rest, ".")
	if len(parts) != 2 {
		return Version{}, fmt.Errorf("protocol version %q: %w", s, errUnsupported)
	}
	maj, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("protocol version %q: %w", s, errUnsupported)
	}
	min, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("protocol version %q: %w", s, errUnsupported)
	}
	if maj < 1 || min < 0 {
		return Version{}, fmt.Errorf("protocol version %q: %w", s, errUnsupported)
	}
	return Version{Major: maj, Minor: min}, nil
}

var errUnsupported = fmt.Errorf("unsupported")

func Compatible(peer Version) error {
	if peer.Major != Major {
		return fmt.Errorf("%s: incompatible major version", peer)
	}
	if peer.Major == Major && peer.Minor < 0 {
		return fmt.Errorf("%s: invalid minor", peer)
	}
	return nil
}

func ParseCompatible(s string) (Version, error) {
	v, err := Parse(s)
	if err != nil {
		return Version{}, err
	}
	if err := Compatible(v); err != nil {
		return Version{}, err
	}
	return v, nil
}
