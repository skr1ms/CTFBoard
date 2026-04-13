package slug

import "regexp"

// PageSlugPatternString is the raw regex that defines a valid page slug:
// lowercase alphanumeric segments separated by single hyphens, no leading or trailing hyphen.
const PageSlugPatternString = `^[a-z0-9]+(?:-[a-z0-9]+)*$`

var pageSlugPattern = regexp.MustCompile(PageSlugPatternString)

// MatchPageSlug reports whether s is a valid page slug.
func MatchPageSlug(s string) bool {
	return pageSlugPattern.MatchString(s)
}
