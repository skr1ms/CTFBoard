package persistent

import (
	"context"
	"fmt"
)

var csvImportAllowedTables = map[string]bool{
	"users":       true,
	"teams":       true,
	"challenges":  true,
	"submissions": true,
	"solves":      true,
	"awards":      true,
}

var csvAllowedColumns = map[string]map[string]bool{
	"challenges":  toSet(backupChallengeImportCols),
	"teams":       toSet(backupTeamImportCols),
	"users":       toSet(backupUserImportCols),
	"awards":      toSet(backupAwardImportCols),
	"solves":      toSet(backupSolveImportCols),
	"submissions": toSet(backupSubmissionImportCols),
}

func toSet(cols []string) map[string]bool {
	m := make(map[string]bool, len(cols))
	for _, c := range cols {
		m[c] = true
	}

	return m
}

// validateCSVColumns checks that every column name in header is present in the
// per-table allowlist derived from the backup import column sets. Returns an
// error on the first unknown column to prevent arbitrary column injection.
func validateCSVColumns(table string, header []string) error {
	allowed, ok := csvAllowedColumns[table]
	if !ok {
		return fmt.Errorf("BackupRepo - validateCSVColumns: no allowed columns defined for table %q", table)
	}

	for _, col := range header {
		if !allowed[col] {
			return fmt.Errorf("BackupRepo - validateCSVColumns: column %q is not allowed for table %q", col, table)
		}
	}

	return nil
}

// ImportCSV imports CSV rows into an allowlisted table.
// Rows are expected to be validated by the usecase before insert.
// Valid rows are inserted in batches with ON CONFLICT (id) DO NOTHING.
// When a batch fails, each row is retried individually so partial imports succeed.
// Returns (importedCount, rowErrors, error).
func (r *BackupRepo) ImportCSV(ctx context.Context, tableName string, header []string, rows [][]string) (int, []string, error) {
	if !csvImportAllowedTables[tableName] {
		return 0, nil, fmt.Errorf("BackupRepo - ImportCSV: unsupported table %q", tableName)
	}

	err := validateCSVColumns(tableName, header)
	if err != nil {
		return 0, nil, fmt.Errorf("BackupRepo - ImportCSV: %w", err)
	}

	var csvErrors []string

	type validRow struct {
		index int
		vals  []any
	}

	var validRows []validRow

	for i, row := range rows {
		if len(row) != len(header) {
			csvErrors = append(csvErrors, fmt.Sprintf("row %d: expected %d columns, got %d", i+1, len(header), len(row)))

			continue
		}

		vals := make([]any, len(row))
		for j, v := range row {
			vals[j] = v
		}

		validRows = append(validRows, validRow{index: i + 1, vals: vals})
	}

	var imported int

	for i := 0; i < len(validRows); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(validRows))

		batch := validRows[i:end]
		q := sqlBuilder().Insert(tableName).Columns(header...)

		for _, rw := range batch {
			q = q.Values(rw.vals...)
		}

		q = q.Suffix("ON CONFLICT (id) DO NOTHING")

		err := r.exec(ctx, q)
		if err != nil {
			for _, rw := range batch {
				single := sqlBuilder().Insert(tableName).Columns(header...).Values(rw.vals...).Suffix("ON CONFLICT (id) DO NOTHING")

				err := r.exec(ctx, single)
				if err != nil {
					csvErrors = append(csvErrors, fmt.Sprintf("row %d: %v", rw.index, err))
				} else {
					imported++
				}
			}

			continue
		}

		imported += len(batch)
	}

	return imported, csvErrors, nil
}
