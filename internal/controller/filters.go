package controller

import (
	"errors"
	"strings"
)

type Filters struct {
	Exact  []string
	Prefix []string
	Suffix []string

	flagSet bool
}

func (f *Filters) IsEmpty() bool {
	return len(f.Exact) == 0 && len(f.Prefix) == 0 && len(f.Suffix) == 0
}

func (f *Filters) Clear() {
	f.Exact = nil
	f.Prefix = nil
	f.Suffix = nil
}

func (f *Filters) Set(value string) error {
	if !f.flagSet {
		f.Clear()
		f.flagSet = true
	}

	for _, filter := range strings.Split(value, ",") {
		pattern := strings.TrimSpace(filter)
		slice := &f.Exact

		if prefix, ok := strings.CutSuffix(pattern, "*"); ok {
			pattern = prefix
			slice = &f.Prefix
		} else if suffix, ok := strings.CutPrefix(pattern, "*"); ok {
			pattern = suffix
			slice = &f.Suffix
		} else if pattern == "" {
			continue
		}

		if strings.ContainsAny(pattern, "* ") {
			return errors.New("invalid filter")
		}

		*slice = append(*slice, pattern)
	}

	return nil
}

func (f *Filters) String() string {
	var patterns []string

	patterns = append(patterns, f.Exact...)

	for _, filter := range f.Prefix {
		patterns = append(patterns, filter+"*")
	}

	for _, filter := range f.Suffix {
		patterns = append(patterns, "*"+filter)
	}

	return strings.Join(patterns, ", ")
}

func EvaluateFilters(excluded Filters, included Filters, value string) bool {
	for _, filter := range excluded.Exact {
		if value == filter {
			return false
		}
	}

	for _, f := range included.Exact {
		if value == f {
			return true
		}
	}

	for _, f := range excluded.Prefix {
		if strings.HasPrefix(value, f) {
			return false
		}
	}

	for _, f := range excluded.Suffix {
		if strings.HasSuffix(value, f) {
			return false
		}
	}

	for _, f := range included.Prefix {
		if strings.HasPrefix(value, f) {
			return true
		}
	}

	for _, f := range included.Suffix {
		if strings.HasSuffix(value, f) {
			return true
		}
	}

	return included.IsEmpty()
}
