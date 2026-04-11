package reputation

import "regexp"

var trigger = regexp.MustCompile(`(?i)^([+]+|[-]+)(rep|реп)(?:$|\s|[[:punct:]])`)

type Trigger struct {
	Delta int
}

func Parse(text string) *Trigger {
	m := trigger.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	signs := m[1]
	n := len(signs)
	if signs[0] == '-' {
		n = -n
	}
	return &Trigger{Delta: n}
}
