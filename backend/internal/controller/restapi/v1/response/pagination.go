package response

// TotalPages returns the number of pages for pagination given total count and per-page size.
// perPage must be positive; returns 0 if perPage <= 0.
func TotalPages(total int64, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	n := int(total) / perPage
	if int(total)%perPage != 0 {
		n++
	}
	return n
}
