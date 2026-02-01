package persistent

type rowScanner interface {
	Scan(dest ...any) error
}
