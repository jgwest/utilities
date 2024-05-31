package robocopy

import (
	"testing"

	"github.com/jgwest/backup-cli/model"
	"github.com/jgwest/backup-cli/util/cmds/generate"
)

func TestValidateRobocopyBasenames(t *testing.T) {

	for _, c := range []struct {
		name      string
		input     []generate.PopulateProcessFoldersResultEntry
		expectErr bool
	}{
		{
			name:      "empty",
			input:     []generate.PopulateProcessFoldersResultEntry{},
			expectErr: false,
		},

		{
			name: "one entry",
			input: []generate.PopulateProcessFoldersResultEntry{
				{SrcFolderPath: "/path", Folder: model.Folder{Path: "/path"}},
			},
			expectErr: false,
		},

		{
			name: "two entries",
			input: []generate.PopulateProcessFoldersResultEntry{
				{SrcFolderPath: "/path", Folder: model.Folder{Path: "/path"}},
				{SrcFolderPath: "/path2", Folder: model.Folder{Path: "/path2"}},
			},
			expectErr: false,
		},

		{
			name: "matching entries: regular",
			input: []generate.PopulateProcessFoldersResultEntry{
				{SrcFolderPath: "/path", Folder: model.Folder{Path: "/path"}},
				{SrcFolderPath: "/path", Folder: model.Folder{Path: "/path"}},
			},
			expectErr: true,
		},

		{
			name: "matching entries: dest folder name match",
			input: []generate.PopulateProcessFoldersResultEntry{

				{SrcFolderPath: "/path", Folder: model.Folder{Path: "/path"}},
				{SrcFolderPath: "/path2", Folder: model.Folder{
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
