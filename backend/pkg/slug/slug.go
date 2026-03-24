package slug

import "regexp"

var PageSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func MatchPageSlug(s string) bool {
	return PageSlugPattern.MatchString(s)
}
