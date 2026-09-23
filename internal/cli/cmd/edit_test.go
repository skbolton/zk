package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zk-org/zk/internal/adapter/fs"
	"github.com/zk-org/zk/internal/adapter/handlebars"
	"github.com/zk-org/zk/internal/adapter/markdown"
	"github.com/zk-org/zk/internal/core"
	"github.com/zk-org/zk/internal/util"
	"github.com/zk-org/zk/internal/util/opt"
	"github.com/zk-org/zk/internal/util/test/assert"
)

// Simulates the "create a note" fzf binding of `zk edit --interactive`, which
// creates a note in the directory resolved for the user's location.
func TestEditInteractiveNewNote(t *testing.T) {
	notebookDir := t.TempDir()

	for _, dir := range []string{"journal", ".zk/templates"} {
		err := os.MkdirAll(filepath.Join(notebookDir, dir), 0o700)
		assert.Equal(t, err, nil)
	}

	write := func(path, content string) {
		err := os.WriteFile(filepath.Join(notebookDir, path), []byte(content), 0o600)
		assert.Equal(t, err, nil)
	}
	write(".zk/templates/journal.md", "Journal: {{title}}\n")

	newNotebook := func(workingDir string) *core.Notebook {
		fs, err := fs.NewFileStorage(workingDir, &util.NullLogger)
		assert.Equal(t, err, nil)
		config := core.NewDefaultConfig()
		config.Note.BodyTemplatePath = opt.NewString("default.md")
		journalNote := config.Note
		journalNote.BodyTemplatePath = opt.NewString("journal.md")
		config.Groups["journal"] = core.GroupConfig{Paths: []string{"journal"}, Note: journalNote}
		return core.NewNotebook(notebookDir, config, core.NotebookPorts{
			FS:                fs,
			NoteContentParser: markdown.NewParser(markdown.ParserOpts{}, &util.NullLogger),
			TemplateLoaderFactory: func(lang string) (core.TemplateLoader, error) {
				return handlebars.NewLoader(handlebars.LoaderOpts{
					LookupPaths: []string{filepath.Join(notebookDir, ".zk/templates")},
					Styler:      core.NullStyler,
				}), nil
			},
			IDGeneratorFactory: func(opts core.IDOptions) func() string {
				return func() string { return "test" }
			},
			OSEnv: func() map[string]string { return map[string]string{} },
		})
	}

	createNote := func(t *testing.T, workingDir string) (*core.Note, error) {
		notebook := newNotebook(workingDir)
		dir := (&Edit{}).newNoteDir(notebook)
		return notebook.NewNote(core.NewNoteOpts{
			Directory: opt.NewNotEmptyString(dir.Path),
			Title:     opt.NewNotEmptyString("Test"),
			DryRun:    true,
		})
	}

	t.Run("a note created from a group dir is a note of that group", func(t *testing.T) {
		note, err := createNote(t, notebookDir+"/journal")
		assert.Equal(t, err, nil)
		assert.Equal(t, note.Path, "journal/test.md")
		assert.Equal(t, note.RawContent, "Journal: Test\n")
	})

	t.Run("a note created from the notebook root uses the root config", func(t *testing.T) {
		_, err := createNote(t, notebookDir)
		assert.Err(t, err, "cannot locate template at default.md")
	})
}
