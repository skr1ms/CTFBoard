package persistent

import (
	"github.com/Masterminds/squirrel"
)

var SB = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
