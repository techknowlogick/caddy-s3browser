package s3browser

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
)

type lessThanFunc func(l, r string, reverse bool) bool

type S3FsSorter struct {
	ltFunc  lessThanFunc
	reverse bool
}

// semVerRegex is the regular expression used to parse a partial semantic version.
// We rely on github.com/Masterminds/semver for the actual parsing, but
// we want to consider the edge cases like 1.0.0 vs. 1.0 vs 1.
var semVerRegex = regexp.MustCompile(`^v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?`)

func NewS3FsSorter(algorithm string, reverse bool) (*S3FsSorter, error) {
	var ltFunc lessThanFunc
	switch algorithm {
	case "none":
		if reverse {
			return nil, errors.New("none cannot be reversed")
		}
		return nil, nil
	case "case-insensitive":
		ltFunc = caseInsensitiveLessThan
	case "semver":
		ltFunc = semanticVersioningLessThan
	default:
		return nil, errors.New("unknown sort algorithm")
	}

	return &S3FsSorter{
		ltFunc:  ltFunc,
		reverse: reverse,
	}, nil
}

func (s *S3FsSorter) Sort(names []string) {
	sort.Slice(names, func(l_idx, r_idx int) bool {
		return s.ltFunc(names[l_idx], names[r_idx], s.reverse)
	})
}

func caseInsensitiveLessThan(l, r string, reverse bool) bool {
	if reverse {
		l, r = r, l
	}

	return strings.ToLower(l) < strings.ToLower(r)
}

func semanticVersioningLessThan(l, r string, reverse bool) bool {
	l_version, l_err := semver.NewVersion(l)
	r_version, r_err := semver.NewVersion(r)

	if l_err == nil && r_err == nil {
		if reverse {
			l_version, r_version = r_version, l_version
		}

		if l_version.Equal(r_version) {
			// 1.1 is less than 1.1.0
			l_groups := semVerRegex.FindStringSubmatch(l)
			r_groups := semVerRegex.FindStringSubmatch(r)
			if l_groups != nil && r_groups != nil {
				return len(r_groups[0]) < len(l_groups[0])
			}
			return false
		}

		return l_version.LessThan(r_version)
	}

	if l_err != nil && r_err != nil {
		// Neither is a semver, fallback to case-insensitive, but never reverse
		return caseInsensitiveLessThan(l, r, false)
	}

	// Only one is a semver, ignore reverse
	return r_err != nil // l < r <=> r is semver
}
