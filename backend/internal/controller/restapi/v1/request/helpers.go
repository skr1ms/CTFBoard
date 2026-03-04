package request

func derefOr[T any](v *T, def T) T {
	if v == nil {
		return def
	}
	return *v
}
