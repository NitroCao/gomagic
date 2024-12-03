package libmagic

import (
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MagicTestSuite struct {
	suite.Suite
	magic *Magic
}

func (s *MagicTestSuite) SetupTest() {
	var err error
	s.magic, err = NewMagic(MagicMimeType | MagicError)
	require.NoError(s.T(), err)
	require.NoError(s.T(), s.magic.MagicLoad([]string{"../testdata/magic.mgc"}))
}

func (s *MagicTestSuite) TestMagicLoadBuffers() {
	m, err := NewMagic(MagicMimeType | MagicError)
	require.NoError(s.T(), err, "failed to create magic descriptor")
	defer m.Close()
	dbBuffers := make([][]byte, 2)
	dbBuffers[0], err = os.ReadFile("../testdata/magic.mgc")
	require.NoError(s.T(), err, "failed to read first database")
	dbBuffers[1], err = os.ReadFile("../testdata/magic2.mgc")
	require.NoError(s.T(), err, "failed to read second database")

	err = m.MagicLoadBuffers(dbBuffers)
	s.NoError(err, "MagicLoadBuffers() failed")

	var fileType string
	fileType, err = m.MagicFile("../testdata/magic.mgc")
	s.NoError(err, "MagicFile() failed")
	s.Equal("application/x-file", fileType)
}

func (s *MagicTestSuite) TestMagicLoad() {
	t := s.T()
	t.Parallel()
	type args struct {
		flags int
		files []string
	}
	tests := []struct {
		name      string
		args      args
		wantError bool
	}{
		{
			name: "invalid database files",
			args: args{
				flags: MagicPreserveAtime,
				files: []string{"../testdata/nonexist.mgc"},
			},
			wantError: true,
		},
		{
			name: "happy path with empty database files",
			args: args{
				flags: MagicMimeType,
				files: []string{},
			},
			wantError: false,
		},
		{
			name: "happy path",
			args: args{
				flags: MagicMimeType,
				files: []string{"../testdata/magic.mgc", "../testdata/magic2.mgc"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			magic, err := NewMagic(tt.args.flags)
			require.NoError(t, err)
			err = magic.MagicLoad(tt.args.files)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (s *MagicTestSuite) TestClose() {
	magic, err := NewMagic(MagicNone)
	require.NoError(s.T(), err)
	assert.NotPanics(s.T(), func() { magic.Close() })

	magic = &Magic{
		handle: nil,
		lock:   &sync.Mutex{},
	}
	assert.NotPanics(s.T(), func() { magic.Close() })
}

func (s *MagicTestSuite) TestMagicFile() {
	t := s.T()
	t.Parallel()
	type args struct {
		filename string
	}
	tests := []struct {
		name      string
		args      args
		wantError bool
		want      string
	}{
		{
			name: "happy path",
			args: args{
				filename: "../testdata/magic.mgc",
			},
			wantError: false,
			want:      "application/x-file",
		},
		{
			name: "invalid input file",
			args: args{
				filename: "../testdata/nonexist.mgc",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := s.magic.MagicFile(tt.args.filename)
			if tt.wantError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func (s *MagicTestSuite) TestMagicBuffer() {
	t := s.T()
	t.Parallel()
	type args struct {
		input []byte
	}
	tests := []struct {
		name      string
		args      args
		wantError bool
		want      string
	}{
		{
			name: "happy path",
			args: args{
				input: []byte(`
<html>
  <body>
  </body>
<html>
`),
			},
			want: "text/html",
		},
		{
			name: "happy path with nil byte slice",
			args: args{
				input: nil,
			},
			want: "application/x-empty",
		},
		{
			name: "happy path with empty buffer",
			args: args{
				input: []byte(``),
			},
			want: "application/x-empty",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result, err := s.magic.MagicBuffer(tt.args.input)
			if tt.wantError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func (s *MagicTestSuite) TestMagicDescriptor() {
	t := s.T()
	t.Parallel()
	type args struct {
		filename string
	}
	tests := []struct {
		name      string
		args      args
		wantError bool
		want      string
	}{
		{
			name: "happy path",
			args: args{
				filename: "../testdata/magic.mgc",
			},
			wantError: false,
			want:      "application/x-file",
		},
		{
			name: "invalid input file",
			args: args{
				filename: "../testdata/nonexist.mgc",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			fd, err := syscall.Open(tt.args.filename, syscall.O_RDONLY, 0600)
			if err != nil {
				fd = 233
			}
			result, err := s.magic.MagicDescriptor(fd)
			if tt.wantError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func (s *MagicTestSuite) TestMagicCompile() {
	t := s.T()
	t.Parallel()
	type args struct {
		filenames []string
	}
	magic, err := NewMagic(MagicMimeType | MagicError)
	require.NoError(t, err)
	tests := []struct {
		name      string
		args      args
		wantError bool
		want      string
	}{
		{
			name: "happy path",
			args: args{
				filenames: []string{"../testdata/lua:../testdata/rpm"},
			},
			wantError: false,
			want:      "application/octet-stream",
		},
		{
			name: "invalid input file",
			args: args{
				filenames: []string{"../testdata/nonexist"},
			},
			wantError: true,
		},
		{
			name: "empty input file",
			args: args{
				filenames: []string{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := magic.MagicCompile(tt.args.filenames)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (s *MagicTestSuite) TestMagicList() {
	t := s.T()
	t.Parallel()
	type args struct {
		filenames []string
	}
	magic, err := NewMagic(MagicMimeType | MagicError)
	require.NoError(t, err)
	tests := []struct {
		name      string
		args      args
		wantError bool
		want      string
	}{
		{
			name: "happy path",
			args: args{
				filenames: []string{"../testdata/magic.mgc"},
			},
			wantError: false,
			want:      "application/octet-stream",
		},
		{
			name: "invalid input file",
			args: args{
				filenames: []string{"../testdata/nonexist"},
			},
			wantError: true,
		},
		{
			name: "empty input file",
			args: args{
				filenames: []string{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := magic.MagicList(tt.args.filenames)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (s *MagicTestSuite) TestMagicCheck() {
	t := s.T()
	t.Parallel()
	type args struct {
		filenames []string
	}
	magic, err := NewMagic(MagicMimeType | MagicError)
	require.NoError(t, err)
	tests := []struct {
		name      string
		args      args
		wantError bool
	}{
		{
			name: "happy path",
			args: args{
				filenames: []string{"../testdata/magic.mgc"},
			},
			wantError: false,
		},
		{
			name: "invalid input file",
			args: args{
				filenames: []string{"../testdata/nonexist"},
			},
			wantError: true,
		},
		{
			name: "empty input file",
			args: args{
				filenames: []string{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := magic.MagicCheck(tt.args.filenames)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (s *MagicTestSuite) TestMagicGetFlags() {
	t := s.T()
	t.Parallel()
	assert.Equal(t, MagicMimeType|MagicError, s.magic.MagicGetFlags())
}

func (s *MagicTestSuite) TestMagicSetFlags() {
	t := s.T()
	t.Parallel()
	magic, err := NewMagic(MagicNone)
	require.NoError(s.T(), err)
	tests := []struct {
		name  string
		flags int
		want  int
	}{
		{
			name:  "MagicMimeType",
			flags: MagicMimeType,
			want:  MagicMimeType,
		},
		{
			name:  "MagicMimeType|MagicError",
			flags: MagicMimeType | MagicError,
			want:  MagicMimeType | MagicError,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.NoError(t, magic.MagicSetFlags(tt.flags))
			assert.Equal(t, tt.want, magic.MagicGetFlags())
		})
	}
}

func (s *MagicTestSuite) TestMagicError() {
	magic, err := NewMagic(MagicNone)
	require.NoError(s.T(), err)
	assert.NotPanics(s.T(), func() { magic.magicError("") })
}

func TestMagic(t *testing.T) {
	suite.Run(t, new(MagicTestSuite))
}
