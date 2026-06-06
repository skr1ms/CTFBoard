package persistent

import (
	"github.com/Masterminds/squirrel"
)

func sqlBuilder() squirrel.StatementBuilderType {
	return squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
}
