package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUploadFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "safe plain text", filename: "task.txt", want: true},
		{name: "safe archive", filename: "challenge.tar.gz", want: true},
		{name: "forbidden extension", filename: "shell.php", want: false},
		{name: "forbidden extension uppercase", filename: "payload.EXE", want: false},
		{name: "double extension", filename: "shell.php.txt", want: false},
		{name: "no extension", filename: "README", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ValidateUploadFilename(tt.filename))
		})
	}
}

func TestValidateFileMagic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header []byte
		want   bool
	}{
		{name: "empty header", header: nil, want: true},
		{name: "text header", header: []byte("hello world"), want: true},
		{name: "pe executable", header: []byte("MZ\x90\x00"), want: false},
		{name: "elf executable", header: []byte("\x7fELF\x02\x01"), want: false},
		{name: "shebang script", header: []byte("#!/bin/sh"), want: false},
		{name: "pdf", header: []byte("%PDF-1.7"), want: false},
		{name: "ole compound", header: []byte("\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1"), want: false},
		{name: "partial non-match", header: []byte("M"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ValidateFileMagic(tt.header))
		})
	}
}
