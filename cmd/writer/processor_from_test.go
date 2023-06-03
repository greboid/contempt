package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func dummyDigestLookup(ref string) (string, string, error) {
	return fmt.Sprintf("image.example.com/%s", ref), "sha256:aabbccddeeffggetcetc", nil
}

func dummyDigestLookupWithError(ref string) (string, string, error) {
	return "", "", fmt.Errorf("error")
}

func TestFromProcessor_Take(t *testing.T) {
	type fields struct {
		digestLookup func(ref string) (string, string, error)
	}
	type args struct {
		buf *bytes.Buffer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name:    "Unrelated line",
			fields:  fields{dummyDigestLookup},
			args:    args{bytes.NewBufferString("RUN some-command\nADD other file")},
			want:    0,
			wantErr: false,
		},
		{
			name:    "Unrelated line followed by a FROM",
			fields:  fields{dummyDigestLookup},
			args:    args{bytes.NewBufferString("RUN some-command\nFROM otherimage")},
			want:    0,
			wantErr: false,
		},
		{
			name:    "FROM at EOF",
			fields:  fields{dummyDigestLookup},
			args:    args{bytes.NewBufferString("FROM someimage")},
			want:    14,
			wantErr: false,
		},
		{
			name:    "FROM with lines after",
			fields:  fields{dummyDigestLookup},
			args:    args{bytes.NewBufferString("FROM someimage\nRUN otherstuff")},
			want:    15,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFromProcessor(tt.fields.digestLookup)
			got, err := f.Take(tt.args.buf)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFromProcessor_Write(t *testing.T) {
	type fields struct {
		digestLookup func(ref string) (string, string, error)
	}
	type args struct {
		args []string
		buf  *bytes.Buffer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Digest func errors",
			fields:  fields{dummyDigestLookupWithError},
			args:    args{[]string{"imagename"}, &bytes.Buffer{}},
			want:    "",
			wantErr: true,
		},
		{
			name:    "Not enough args",
			fields:  fields{dummyDigestLookup},
			args:    args{[]string{}, &bytes.Buffer{}},
			want:    "",
			wantErr: true,
		},
		{
			name:    "Wrong second arg",
			fields:  fields{dummyDigestLookup},
			args:    args{[]string{"image", "not-as"}, &bytes.Buffer{}},
			want:    "",
			wantErr: true,
		},
		{
			name:    "Second arg with no third",
			fields:  fields{dummyDigestLookup},
			args:    args{[]string{"image", "as"}, &bytes.Buffer{}},
			want:    "",
			wantErr: true,
		},
		{
			name:    "Too many args",
			fields:  fields{dummyDigestLookup},
			args:    args{[]string{"image", "as", "something", "else"}, &bytes.Buffer{}},
			want:    "",
			wantErr: true,
		},
		{
			name:    "Simple substitution",
			fields:  fields{dummyDigestLookup},
			args:    args{[]string{"image123"}, &bytes.Buffer{}},
			want:    "FROM image.example.com/image123@sha256:aabbccddeeffggetcetc\n",
			wantErr: false,
		},
		{
			name:    "Substitution with alias",
			fields:  fields{dummyDigestLookup},
			args:    args{[]string{"image123", "as", "base"}, &bytes.Buffer{}},
			want:    "FROM image.example.com/image123@sha256:aabbccddeeffggetcetc AS base\n",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFromProcessor(tt.fields.digestLookup)

			err := f.Write(tt.args.args, tt.args.buf)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, tt.args.buf.String())
		})
	}
}
