package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// runLesson handles the lesson command: a Lesson is an ordered sequence of
// steps, each a text that may show a collection or a position (ADR-0066).
// The CLI writes and reads them through the same Database methods as the
// desktop, and exports them for a student.
func (cli *CLI) runLesson(args []string) error {
	if len(args) < 1 {
		cli.printLessonUsage()
		return fmt.Errorf("missing lesson sub-command")
	}
	sub := strings.ToLower(args[0])
	if sub == "--help" || sub == "-h" || sub == "help" {
		cli.printLessonUsage()
		return nil
	}
	run, ok := cli.lessonHandlers()[sub]
	if !ok {
		cli.printLessonUsage()
		return fmt.Errorf("unknown lesson sub-command: %s", args[0])
	}
	return run(args[1:])
}

// lessonHandlers returns the sub-command table of `blunderdb lesson`.
func (cli *CLI) lessonHandlers() map[string]func([]string) error {
	return map[string]func([]string) error{
		"list":        cli.runLessonList,
		"show":        cli.runLessonShow,
		"create":      cli.runLessonCreate,
		"edit":        cli.runLessonEdit,
		"delete":      cli.runLessonDelete,
		"add-step":    cli.runLessonAddStep,
		"edit-step":   cli.runLessonEditStep,
		"remove-step": cli.runLessonRemoveStep,
		"reorder":     cli.runLessonReorder,
		"done":        cli.runLessonDone,
		"progress":    cli.runLessonProgress,
		"export":      cli.runLessonExport,
	}
}

// LessonSubcommands returns the sub-commands of `blunderdb lesson`, sorted —
// the exported view cmd/cli-doc-gen walks.
func (cli *CLI) LessonSubcommands() []string {
	h := cli.lessonHandlers()
	names := make([]string, 0, len(h))
	for name := range h {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (cli *CLI) printLessonUsage() {
	fmt.Println("Usage: blunderdb lesson <sub-command> [options]")
	fmt.Println()
	fmt.Println("Manage lessons (ordered steps of text, each showing a collection or a position).")
	fmt.Println()
	fmt.Println("Sub-commands:")
	fmt.Println("  list         List lessons (id, name, number of steps)")
	fmt.Println("  show         Show the steps of one lesson, in order")
	fmt.Println("  create       Create an empty lesson")
	fmt.Println("  edit         Rename a lesson or change its description")
	fmt.Println("  delete       Delete a lesson (its collections and positions stay)")
	fmt.Println("  add-step     Append a step to a lesson")
	fmt.Println("  edit-step    Change the title, text, collection or position of a step")
	fmt.Println("  remove-step  Remove a step")
	fmt.Println("  reorder      Set the order of a lesson's steps")
	fmt.Println("  export       Export lessons, with what their steps show, to a new database file")
	fmt.Println("  done         Mark a step done (or --undo the mark) in this database")
	fmt.Println("  progress     Show which steps of a lesson are marked done, and when")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  blunderdb lesson create --db database.db --name \"Playing against a prime\"")
	fmt.Println("  blunderdb lesson add-step --db database.db --lesson 1 --title \"Timing\" --text \"Count the pips.\" --collection 3")
	fmt.Println("  blunderdb lesson show --db database.db --id 1")
	fmt.Println("  blunderdb lesson export --db database.db --id 1 --out lesson.dbx --password secret --watermark \"Course of 12 March\"")
	fmt.Println()
	fmt.Println("Use 'blunderdb lesson <sub-command> --help' for the options of a sub-command.")
}

func lessonFlagSet(sub, summary string, examples ...string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet("lesson "+sub, flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to the database file (required)")
	fs.Usage = func() {
		fmt.Printf("Usage: blunderdb lesson %s [options]\n\n%s\n\nOptions:\n", sub, summary)
		fs.PrintDefaults()
		if len(examples) > 0 {
			fmt.Println()
			fmt.Println("Examples:")
			for _, ex := range examples {
				fmt.Println("  " + ex)
			}
		}
	}
	return fs, dbPath
}

func (cli *CLI) lessonOpen(fs *flag.FlagSet, dbPath *string, args []string) error {
	return cli.collectionOpen(fs, dbPath, args)
}

// requireID reports a missing or non-positive required id flag.
func requireID(fs *flag.FlagSet, name string, v int64) error {
	if v <= 0 {
		fs.Usage()
		return fmt.Errorf("missing required flag: --%s", name)
	}
	return nil
}

// stepText returns --text, or the content of --text-file when given: a step's
// text is often several paragraphs, easier written in a file.
func stepText(text, file string) (string, error) {
	if file == "" {
		return text, nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read --text-file: %w", err)
	}
	return string(b), nil
}

func (cli *CLI) runLessonList(args []string) error {
	fs, dbPath := lessonFlagSet("list", "List the lessons of the database.",
		"blunderdb lesson list --db database.db --format json")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	lessons, err := cli.db.ListLessons()
	if err != nil {
		return fmt.Errorf("failed to list lessons: %w", err)
	}
	if strings.EqualFold(*format, "json") {
		return printJSON(lessons)
	}
	if len(lessons) == 0 {
		fmt.Println("No lessons.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTEPS")
	for _, l := range lessons {
		fmt.Fprintf(w, "%d\t%s\t%d\n", l.ID, l.Name, l.StepCount)
	}
	return w.Flush()
}

func (cli *CLI) runLessonShow(args []string) error {
	fs, dbPath := lessonFlagSet("show", "Show a lesson's steps in reading order.",
		"blunderdb lesson show --db database.db --id 1",
		"blunderdb lesson show --db database.db --id 1 --format json")
	id := fs.Int64("id", 0, "Lesson ID (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "id", *id); err != nil {
		return err
	}
	l, err := cli.db.GetLesson(*id)
	if err != nil {
		return fmt.Errorf("failed to read lesson %d: %w", *id, err)
	}
	if strings.EqualFold(*format, "json") {
		return printJSON(l)
	}
	printLesson(l)
	return nil
}

func printLesson(l *domain.Lesson) {
	fmt.Printf("Lesson %d: %s\n", l.ID, l.Name)
	if l.Description != "" {
		fmt.Println(l.Description)
	}
	for i, st := range l.Steps {
		fmt.Printf("\n%d. [step %d] %s\n", i+1, st.ID, st.Title)
		if st.CollectionID != 0 {
			fmt.Printf("   collection %d\n", st.CollectionID)
		}
		if st.PositionID != 0 {
			fmt.Printf("   position %d\n", st.PositionID)
		}
		for _, line := range strings.Split(strings.TrimRight(st.Text, "\n"), "\n") {
			if line != "" {
				fmt.Println("   " + line)
			}
		}
	}
}

func (cli *CLI) runLessonCreate(args []string) error {
	fs, dbPath := lessonFlagSet("create", "Create an empty lesson.",
		"blunderdb lesson create --db database.db --name \"Playing against a prime\"")
	name := fs.String("name", "", "Lesson name (required)")
	description := fs.String("description", "", "Lesson description")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --name")
	}
	id, err := cli.db.CreateLesson(strings.TrimSpace(*name), *description)
	if err != nil {
		return fmt.Errorf("failed to create lesson: %w", err)
	}
	fmt.Printf("Successfully created lesson %q (ID: %d)\n", strings.TrimSpace(*name), id)
	return nil
}

// setFlags returns the names of the flags given on the command line, so an
// edit changes only what was asked.
func setFlags(fs *flag.FlagSet) map[string]bool {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}

func (cli *CLI) runLessonEdit(args []string) error {
	fs, dbPath := lessonFlagSet("edit", "Rename a lesson or change its description; flags not given keep their value.",
		"blunderdb lesson edit --db database.db --id 1 --name \"Primes\"")
	id := fs.Int64("id", 0, "Lesson ID (required)")
	name := fs.String("name", "", "New name")
	description := fs.String("description", "", "New description")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "id", *id); err != nil {
		return err
	}
	l, err := cli.db.GetLesson(*id)
	if err != nil {
		return fmt.Errorf("failed to read lesson %d: %w", *id, err)
	}
	set := setFlags(fs)
	if set["name"] {
		l.Name = *name
	}
	if set["description"] {
		l.Description = *description
	}
	if err := cli.db.UpdateLesson(*id, l.Name, l.Description); err != nil {
		return fmt.Errorf("failed to edit lesson %d: %w", *id, err)
	}
	fmt.Printf("Successfully edited lesson %d\n", *id)
	return nil
}

func (cli *CLI) runLessonDelete(args []string) error {
	fs, dbPath := lessonFlagSet("delete", "Delete a lesson and its steps; the collections and positions they show stay.",
		"blunderdb lesson delete --db database.db --id 1 --confirm")
	id := fs.Int64("id", 0, "Lesson ID (required)")
	confirm := fs.Bool("confirm", false, "Confirm the deletion (required)")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "id", *id); err != nil {
		return err
	}
	if !*confirm {
		return fmt.Errorf("refusing to delete lesson %d without --confirm", *id)
	}
	if err := cli.db.DeleteLesson(*id); err != nil {
		return fmt.Errorf("failed to delete lesson %d: %w", *id, err)
	}
	fmt.Printf("Successfully deleted lesson %d\n", *id)
	return nil
}

func (cli *CLI) runLessonAddStep(args []string) error {
	fs, dbPath := lessonFlagSet("add-step", "Append a step to a lesson. A step may show a collection, a position, both or neither.",
		"blunderdb lesson add-step --db database.db --lesson 1 --title \"Timing\" --text-file timing.txt --collection 3",
		"blunderdb lesson add-step --db database.db --lesson 1 --title \"The key position\" --position 412")
	lessonID := fs.Int64("lesson", 0, "Lesson ID (required)")
	title := fs.String("title", "", "Step title")
	text := fs.String("text", "", "Step text")
	textFile := fs.String("text-file", "", "Read the step text from this file (wins over --text)")
	collection := fs.Int64("collection", 0, "Collection the step shows (0: none)")
	position := fs.Int64("position", 0, "Position the step shows (0: none)")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "lesson", *lessonID); err != nil {
		return err
	}
	body, err := stepText(*text, *textFile)
	if err != nil {
		return err
	}
	id, err := cli.db.AddLessonStep(*lessonID, *title, body, *collection, *position)
	if err != nil {
		return fmt.Errorf("failed to add a step to lesson %d: %w", *lessonID, err)
	}
	fmt.Printf("Successfully added step %d to lesson %d\n", id, *lessonID)
	return nil
}

func (cli *CLI) runLessonEditStep(args []string) error {
	fs, dbPath := lessonFlagSet("edit-step", "Change a step; flags not given keep their value, 0 clears a collection or a position.",
		"blunderdb lesson edit-step --db database.db --lesson 1 --step 7 --text \"Count again.\"",
		"blunderdb lesson edit-step --db database.db --lesson 1 --step 7 --position 0")
	lessonID := fs.Int64("lesson", 0, "Lesson ID (required)")
	stepID := fs.Int64("step", 0, "Step ID, as `lesson show` prints it (required)")
	title := fs.String("title", "", "New title")
	text := fs.String("text", "", "New text")
	textFile := fs.String("text-file", "", "Read the new text from this file (wins over --text)")
	collection := fs.Int64("collection", 0, "Collection the step shows (0: none)")
	position := fs.Int64("position", 0, "Position the step shows (0: none)")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "lesson", *lessonID); err != nil {
		return err
	}
	if err := requireID(fs, "step", *stepID); err != nil {
		return err
	}
	l, err := cli.db.GetLesson(*lessonID)
	if err != nil {
		return fmt.Errorf("failed to read lesson %d: %w", *lessonID, err)
	}
	var st *domain.LessonStep
	for i := range l.Steps {
		if l.Steps[i].ID == *stepID {
			st = &l.Steps[i]
		}
	}
	if st == nil {
		return fmt.Errorf("lesson %d has no step %d", *lessonID, *stepID)
	}
	set := setFlags(fs)
	if set["title"] {
		st.Title = *title
	}
	if set["text"] || set["text-file"] {
		if st.Text, err = stepText(*text, *textFile); err != nil {
			return err
		}
	}
	if set["collection"] {
		st.CollectionID = *collection
	}
	if set["position"] {
		st.PositionID = *position
	}
	if err := cli.db.UpdateLessonStep(st.ID, st.Title, st.Text, st.CollectionID, st.PositionID); err != nil {
		return fmt.Errorf("failed to edit step %d: %w", st.ID, err)
	}
	fmt.Printf("Successfully edited step %d\n", st.ID)
	return nil
}

func (cli *CLI) runLessonRemoveStep(args []string) error {
	fs, dbPath := lessonFlagSet("remove-step", "Remove a step from its lesson.",
		"blunderdb lesson remove-step --db database.db --step 7")
	stepID := fs.Int64("step", 0, "Step ID (required)")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "step", *stepID); err != nil {
		return err
	}
	if err := cli.db.RemoveLessonStep(*stepID); err != nil {
		return fmt.Errorf("failed to remove step %d: %w", *stepID, err)
	}
	fmt.Printf("Successfully removed step %d\n", *stepID)
	return nil
}

func (cli *CLI) runLessonReorder(args []string) error {
	fs, dbPath := lessonFlagSet("reorder", "Set the order of a lesson's steps; --steps names every step once.",
		"blunderdb lesson reorder --db database.db --lesson 1 --steps 9,7,8")
	lessonID := fs.Int64("lesson", 0, "Lesson ID (required)")
	steps := fs.String("steps", "", "Step IDs in the new order, comma-separated (required)")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "lesson", *lessonID); err != nil {
		return err
	}
	ids, err := parseIDList(*steps)
	if err != nil || len(ids) == 0 {
		fs.Usage()
		return fmt.Errorf("invalid or missing --steps")
	}
	if err := cli.db.ReorderLessonSteps(*lessonID, ids); err != nil {
		return fmt.Errorf("failed to reorder lesson %d: %w", *lessonID, err)
	}
	fmt.Printf("Successfully reordered lesson %d\n", *lessonID)
	return nil
}

func (cli *CLI) runLessonExport(args []string) error {
	fs, dbPath := lessonFlagSet("export", "Export lessons, with the collections and positions their steps show, to a new database file.",
		"blunderdb lesson export --db database.db --id 1 --out lesson.db --watermark \"Course of 12 March\"",
		"blunderdb lesson export --db database.db --id 1,2 --out lessons.dbx --password secret")
	ids := fs.String("id", "", "Lesson ID(s) to export, comma-separated (required)")
	out := fs.String("out", "", "Path of the database file to write (required)")
	includeAnalysis := fs.Bool("analysis", true, "Include analyses")
	includeComments := fs.Bool("comments", true, "Include comments")
	watermark := fs.String("watermark", "", "Mark the exported file with where it comes from")
	watermarkNote := fs.String("watermark-note", "", "Free text attached to the watermark (terms of use, contact)")
	password := fs.String("password", "", "Encrypt the export into a protected .dbx file")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	lessonIDs, err := parseIDList(*ids)
	if err != nil || len(lessonIDs) == 0 {
		fs.Usage()
		return fmt.Errorf("invalid or missing --id")
	}
	if *out == "" {
		fs.Usage()
		return fmt.Errorf("missing required flag: --out")
	}
	for _, id := range lessonIDs {
		if _, err := cli.db.GetLesson(id); err != nil {
			return fmt.Errorf("lesson with ID %d not found", id)
		}
	}
	err = cli.db.ExportDatabase(domain.ExportOptions{
		ExportPath:      *out,
		Metadata:        map[string]string{},
		IncludeLessons:  true,
		LessonIDs:       lessonIDs,
		IncludeAnalysis: *includeAnalysis,
		IncludeComments: *includeComments,
		Watermark:       *watermark,
		WatermarkNote:   *watermarkNote,
		Password:        *password,
	})
	if err != nil {
		return fmt.Errorf("failed to export lessons: %w", err)
	}
	fmt.Printf("Successfully exported %d lesson(s) to %s\n", len(lessonIDs), *out)
	return nil
}

func (cli *CLI) runLessonDone(args []string) error {
	fs, dbPath := lessonFlagSet("done",
		"Mark a step done, the reader's own progress (ADR-0069). It is written only by this gesture, in this database; no export carries it.",
		"blunderdb lesson done --db database.db --step 7",
		"blunderdb lesson done --db database.db --step 7 --undo")
	stepID := fs.Int64("step", 0, "Step ID (required)")
	undo := fs.Bool("undo", false, "Withdraw the mark instead of setting it")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "step", *stepID); err != nil {
		return err
	}
	if err := cli.db.SetLessonStepDone(*stepID, !*undo); err != nil {
		return fmt.Errorf("failed to mark step %d: %w", *stepID, err)
	}
	if *undo {
		fmt.Printf("Step %d is no longer marked done\n", *stepID)
	} else {
		fmt.Printf("Step %d marked done\n", *stepID)
	}
	return nil
}

func (cli *CLI) runLessonProgress(args []string) error {
	fs, dbPath := lessonFlagSet("progress", "Show which steps of a lesson are marked done, and the date of the gesture.",
		"blunderdb lesson progress --db database.db --id 1",
		"blunderdb lesson progress --db database.db --id 1 --format json")
	id := fs.Int64("id", 0, "Lesson ID (required)")
	format := fs.String("format", "text", "Output format: text or json")
	if err := cli.lessonOpen(fs, dbPath, args); err != nil {
		return err
	}
	if err := requireID(fs, "id", *id); err != nil {
		return err
	}
	l, err := cli.db.GetLesson(*id)
	if err != nil {
		return fmt.Errorf("failed to read lesson %d: %w", *id, err)
	}
	done, err := cli.db.LessonDoneSteps(*id)
	if err != nil {
		return fmt.Errorf("failed to read the progress of lesson %d: %w", *id, err)
	}
	if strings.EqualFold(*format, "json") {
		type stepProgress struct {
			StepID int64  `json:"stepId"`
			Title  string `json:"title"`
			Done   bool   `json:"done"`
			DoneAt string `json:"doneAt,omitempty"`
		}
		rows := make([]stepProgress, 0, len(l.Steps))
		for _, st := range l.Steps {
			at, ok := done[st.ID]
			rows = append(rows, stepProgress{StepID: st.ID, Title: st.Title, Done: ok, DoneAt: at})
		}
		return printJSON(rows)
	}
	fmt.Printf("Lesson %d: %s — %d / %d steps done\n", l.ID, l.Name, len(done), len(l.Steps))
	for i, st := range l.Steps {
		mark := "[ ]"
		if at, ok := done[st.ID]; ok {
			mark = "[x] " + at
		}
		fmt.Printf("%d. [step %d] %s  %s\n", i+1, st.ID, st.Title, mark)
	}
	return nil
}
