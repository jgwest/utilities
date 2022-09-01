package generate

import (
	"testing"

	"github.com/jgwest/backup-cli/model"
)

func TestValidateRobocopyBasenames(t *testing.T) {

	for _, c := range []struct {
		name      string
		input     [][]interface{}
		expectErr bool
	}{
		{
			name:      "empty",
			input:     [][]interface{}{},
			expectErr: false,
		},

		{
			name: "one entry",
			input: [][]interface{}{
				{"/path", model.Folder{Path: "/path"}},
			},
			expectErr: false,
		},

		{
			name: "two entries",
			input: [][]interface{}{
				{"/path", model.Folder{Path: "/path"}},
				{"/path2", model.Folder{Path: "/path2"}},
			},
			expectErr: false,
		},

		{
			name: "matching entries: regular",
			input: [][]interface{}{
				{"/path", model.Folder{Path: "/path"}},
				{"/path", model.Folder{Path: "/path"}},
			},
			expectErr: true,
		},

		{
			name: "matching entries: dest folder name match",
			input: [][]interface{}{

				{"/path", model.Folder{Path: "/path"}},
				{"/path2", model.Folder{
					Path: "/path2",
					Robocopy: &model.RobocopyFolderSettings{
						DestFolderName: "path",
					},
				}},
			},
			expectErr: true,
		},
	} {

		t.Run(c.name, func(t *testing.T) {
			err := robocopyValidateBasenames(c.input)

			pass := (err != nil) == c.expectErr
			if !pass {
				t.Errorf("Error values do not match: %v %v", err, pass)
			}
		})

	}

}
