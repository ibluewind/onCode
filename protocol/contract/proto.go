package contract

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var jsonNameRe = regexp.MustCompile(`json_name\s*=\s*"([^"]+)"`)

func ProtoJSONNames(src string) map[string][]string {
	out := map[string][]string{}
	var current string
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "message ") {
			name := strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(t, "message ")), "{")
			current = strings.TrimSpace(name)
			continue
		}
		if current == "" {
			continue
		}
		if t == "}" {
			current = ""
			continue
		}
		m := jsonNameRe.FindStringSubmatch(t)
		if len(m) == 2 {
			out[current] = append(out[current], m[1])
		}
	}
	return out
}

func LoadProtoJSONNames() (map[string][]string, error) {
	b, err := os.ReadFile(ProtoPath())
	if err != nil {
		return nil, err
	}
	names := ProtoJSONNames(string(b))
	if len(names) == 0 {
		return nil, fmt.Errorf("no proto json_name fields found")
	}
	return names, nil
}

func ContainsAll(have []string, want ...string) error {
	set := map[string]struct{}{}
	for _, h := range have {
		set[h] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return fmt.Errorf("missing field %q", w)
		}
	}
	return nil
}
