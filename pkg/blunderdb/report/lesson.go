package report

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MaxCollectionBoards is how many boards a Step that shows a Collection
// prints: a document people read, not a dump of the collection.
const MaxCollectionBoards = 12

// maxAnalysisMoves is how many candidate moves the foldable analysis lists.
const maxAnalysisMoves = 5

// lessonLabels are the words of the Lesson document, in the languages of
// Languages(). They are the document's own, not the screen's, so they do not
// follow the frontend catalogues.
var lessonLabels = map[string]map[string]string{
	"fr": {"steps": "Étapes", "answer": "Voir l'analyse", "best": "Meilleur coup", "cube": "Videau", "more": "… et %d autres positions de la collection.", "noStep": "Cette leçon n'a pas d'étape.", "origin": "Origine du fichier :", "footer": "Document autonome établi par blunderDB."},
	"en": {"steps": "Steps", "answer": "Show the analysis", "best": "Best move", "cube": "Cube", "more": "… and %d more positions of the collection.", "noStep": "This lesson has no step.", "origin": "Origin of the file:", "footer": "Self-contained document produced by blunderDB."},
	"de": {"steps": "Schritte", "answer": "Analyse anzeigen", "best": "Bester Zug", "cube": "Würfel", "more": "… und %d weitere Stellungen der Sammlung.", "noStep": "Diese Lektion hat keinen Schritt.", "origin": "Herkunft der Datei:", "footer": "Eigenständiges Dokument, erstellt von blunderDB."},
	"el": {"steps": "Βήματα", "answer": "Εμφάνιση ανάλυσης", "best": "Καλύτερη κίνηση", "cube": "Κύβος", "more": "… και %d ακόμη θέσεις της συλλογής.", "noStep": "Αυτό το μάθημα δεν έχει βήματα.", "origin": "Προέλευση του αρχείου:", "footer": "Αυτόνομο έγγραφο από το blunderDB."},
	"es": {"steps": "Pasos", "answer": "Ver el análisis", "best": "Mejor jugada", "cube": "Cubo", "more": "… y %d posiciones más de la colección.", "noStep": "Esta lección no tiene pasos.", "origin": "Origen del archivo:", "footer": "Documento autónomo generado por blunderDB."},
	"fi": {"steps": "Vaiheet", "answer": "Näytä analyysi", "best": "Paras siirto", "cube": "Tuplauskuutio", "more": "… ja %d muuta kokoelman asemaa.", "noStep": "Tässä oppitunnissa ei ole vaiheita.", "origin": "Tiedoston alkuperä:", "footer": "blunderDB:n tuottama itsenäinen asiakirja."},
	"it": {"steps": "Passaggi", "answer": "Mostra l'analisi", "best": "Mossa migliore", "cube": "Cubo", "more": "… e altre %d posizioni della raccolta.", "noStep": "Questa lezione non ha passaggi.", "origin": "Origine del file:", "footer": "Documento autonomo prodotto da blunderDB."},
	"ja": {"steps": "ステップ", "answer": "解析を表示", "best": "最善手", "cube": "キューブ", "more": "… ほか %d 個のコレクションのポジション。", "noStep": "このレッスンにはステップがありません。", "origin": "ファイルの出所:", "footer": "blunderDB が作成した単体のドキュメント。"},
	"ru": {"steps": "Шаги", "answer": "Показать анализ", "best": "Лучший ход", "cube": "Куб", "more": "… и ещё %d позиций коллекции.", "noStep": "В этом уроке нет шагов.", "origin": "Происхождение файла:", "footer": "Автономный документ, созданный blunderDB."},
}

func lessonLabel(lang, key string) string {
	if m, ok := lessonLabels[lang]; ok {
		return m[key]
	}
	return lessonLabels[DefaultLanguage][key]
}

// lessonBoard is one board of a Step: the drawing and the stored analysis.
type lessonBoard struct {
	Diagram  string
	Analysis string // HTML of the foldable analysis, empty when none is stored
}

type lessonStepView struct {
	Title, Text string
	Boards      []lessonBoard
	Hidden      int // positions of the Collection that are not printed
}

// BuildLesson renders a Lesson as one self-contained HTML document: title,
// description, every Step with its text and board, the stored analysis of each
// position folded away like a quiz answer, and the Watermark of the database
// at the foot when it has one. diagram nil means the built-in Diagram; the
// desktop passes the boards it drew in the board palette. Nothing is written.
func BuildLesson(ctx context.Context, s storage.Storage, scope string, lessonID int64, lang string, diagram Diagrammer) (string, error) {
	if _, ok := lessonLabels[lang]; !ok {
		lang = DefaultLanguage
	}
	if diagram == nil {
		diagram = Diagram
	}
	lesson, err := s.Lessons().Get(ctx, scope, lessonID)
	if err != nil {
		return "", fmt.Errorf("report: load lesson: %w", err)
	}
	board := func(p *domain.Position) lessonBoard {
		b := lessonBoard{Diagram: diagram(p)}
		if a, err := s.Analyses().Load(ctx, scope, p.ID); err == nil && a != nil {
			b.Analysis = analysisHTML(lang, a)
		}
		return b
	}
	var views []lessonStepView
	for _, st := range lesson.Steps {
		v := lessonStepView{Title: st.Title, Text: st.Text}
		if st.CollectionID > 0 {
			total, err := s.Collections().CountPositions(ctx, scope, st.CollectionID)
			if err != nil {
				// A deleted Collection leaves the Step and its text in place.
				total = 0
			}
			for p, err := range s.Collections().Positions(ctx, scope, st.CollectionID, storage.ListOpts{Limit: MaxCollectionBoards}) {
				if err != nil {
					return "", fmt.Errorf("report: load collection %d: %w", st.CollectionID, err)
				}
				v.Boards = append(v.Boards, board(p))
			}
			if total > len(v.Boards) {
				v.Hidden = total - len(v.Boards)
			}
		}
		if st.PositionID > 0 {
			ps, err := s.Positions().LoadByIDs(ctx, scope, []int64{st.PositionID})
			if err == nil && len(ps) > 0 {
				v.Boards = append(v.Boards, board(&ps[0]))
			}
		}
		views = append(views, v)
	}
	return renderLesson(lang, lesson, views, originLine(ctx, s, scope)), nil
}

// originLine is the Watermark's origin, as its producer wrote it, or "".
func originLine(ctx context.Context, s storage.Storage, scope string) string {
	meta, err := s.Metadata().Load(ctx, scope)
	if err != nil {
		return ""
	}
	env, err := issuance.DecodeEnvelope(meta[issuance.KeyWatermark])
	if err != nil || !env.IsWatermarked() {
		return ""
	}
	w, err := env.Open()
	if err != nil {
		return ""
	}
	line := w.Origin
	if w.Note != "" {
		line += " — " + w.Note
	}
	return line
}

func analysisHTML(lang string, a *domain.PositionAnalysis) string {
	esc := html.EscapeString
	var b strings.Builder
	if a.CheckerAnalysis != nil {
		moves := a.CheckerAnalysis.Moves
		if len(moves) > maxAnalysisMoves {
			moves = moves[:maxAnalysisMoves]
		}
		for i, m := range moves {
			errCell := ""
			if m.EquityError != nil && i > 0 {
				errCell = fmt.Sprintf(" (%+.3f)", *m.EquityError)
			}
			cls := ""
			if i == 0 {
				cls = ` class="best"`
			}
			fmt.Fprintf(&b, "<li%s>%s — %+.3f%s</li>", cls, esc(m.Move), m.Equity, esc(errCell))
		}
		if b.Len() > 0 {
			return fmt.Sprintf("<p><b>%s</b></p><ol>%s</ol>", esc(lessonLabel(lang, "best")), b.String())
		}
	}
	if c := a.DoublingCubeAnalysis; c != nil && c.BestCubeAction != "" {
		return fmt.Sprintf("<p><b>%s</b> : %s</p><ul><li>ND %+.3f</li><li>D/T %+.3f</li><li>D/P %+.3f</li></ul>",
			esc(lessonLabel(lang, "cube")), esc(c.BestCubeAction), c.CubefulNoDoubleEquity, c.CubefulDoubleTakeEquity, c.CubefulDoublePassEquity)
	}
	return ""
}

func paragraphs(text string) string {
	var b strings.Builder
	for _, para := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
		if strings.TrimSpace(para) == "" {
			continue
		}
		fmt.Fprintf(&b, "<p>%s</p>\n", strings.ReplaceAll(html.EscapeString(strings.TrimSpace(para)), "\n", "<br>"))
	}
	return b.String()
}

func renderLesson(lang string, l *domain.Lesson, steps []lessonStepView, origin string) string {
	esc := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"%s\">\n<head>\n<meta charset=\"utf-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<title>%s</title>\n<style>\n%s%s</style>\n</head>\n<body>\n", esc(lang), esc(l.Name), css, lessonCSS)
	fmt.Fprintf(&b, "<h1>%s</h1>\n%s", esc(l.Name), paragraphs(l.Description))
	if len(steps) == 0 {
		fmt.Fprintf(&b, "<p>%s</p>\n", esc(lessonLabel(lang, "noStep")))
	}
	for i, st := range steps {
		fmt.Fprintf(&b, "\n<section class=\"step\">\n  <h2>%d. %s</h2>\n%s", i+1, esc(st.Title), paragraphs(st.Text))
		for _, bd := range st.Boards {
			fmt.Fprintf(&b, "  <div class=\"diagram\">%s</div>\n", bd.Diagram)
			if bd.Analysis != "" {
				fmt.Fprintf(&b, "  <details><summary>%s</summary>%s</details>\n", esc(lessonLabel(lang, "answer")), bd.Analysis)
			}
		}
		if st.Hidden > 0 {
			fmt.Fprintf(&b, "  <p class=\"meta\">%s</p>\n", esc(fmt.Sprintf(lessonLabel(lang, "more"), st.Hidden)))
		}
		b.WriteString("</section>\n")
	}
	if origin != "" {
		fmt.Fprintf(&b, "<p class=\"footer\">%s %s</p>\n", esc(lessonLabel(lang, "origin")), esc(origin))
	}
	fmt.Fprintf(&b, "<p class=\"footer\">%s</p>\n</body>\n</html>\n", esc(lessonLabel(lang, "footer")))
	return b.String()
}

const lessonCSS = `  .step { margin-bottom: 2rem; }
  .step h2 { font-size: 1.15rem; }
  details { margin: 0.4rem 0 1rem 0; }
  summary { cursor: pointer; color: #0b5; }
  li.best { font-weight: 600; }
`
