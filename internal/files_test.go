package internal

import (
	"github.com/stretchr/testify/assert"
	"io/fs"
	"testing"
	"testing/fstest"
)

func Test_findFiles(t *testing.T) {
	type args struct {
		f       fs.FS
		matcher func(string) bool
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"Returns empty slice if nothing matches",
			args{
				f: fstest.MapFS{
					"foo": &fstest.MapFile{},
				},
				matcher: func(s string) bool {
					return false
				},
			},
			[]string{},
			false,
		},
		{
			"Returns files in root",
			args{
				f: fstest.MapFS{
					"foo": &fstest.MapFile{},
					"bar": &fstest.MapFile{},
				},
				matcher: func(s string) bool {
					return true
				},
			},
			[]string{"foo", "bar"},
			false,
		},
		{
			"Returns files with paths",
			args{
				f: fstest.MapFS{
					"some/folder/foo": &fstest.MapFile{},
					"other/dir/bar":   &fstest.MapFile{},
				},
				matcher: func(s string) bool {
					return true
				},
			},
			[]string{"some/folder/foo", "other/dir/bar"},
			false,
		},
		{
			"Ignores directories prefixed with a dot",
			args{
				f: fstest.MapFS{
					"some/.folder/foo": &fstest.MapFile{},
					".other/dir/bar":   &fstest.MapFile{},
					"no/dots/here":     &fstest.MapFile{},
				},
				matcher: func(s string) bool {
					return true
				},
			},
			[]string{"no/dots/here"},
			false,
		},
		{
			"Returns only files matched by the matcher",
			args{
				f: fstest.MapFS{
					"some/folder/foo": &fstest.MapFile{},
					"other/dir/bar":   &fstest.MapFile{},
					"blah/here":       &fstest.MapFile{},
				},
				matcher: func(s string) bool {
					return len(s) == 3
				},
			},
			[]string{"some/folder/foo", "other/dir/bar"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindFiles(tt.args.f, tt.args.matcher)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.ElementsMatch(t, got, tt.want)
		})
	}
}
