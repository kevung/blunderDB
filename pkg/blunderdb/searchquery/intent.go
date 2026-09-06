package searchquery

import (
	"sort"
	"strings"
	"unicode"
)

// Une grammaire d'intentions (#283, fiche I.27).
//
// « mes blunders de videau au score » se traduit en jetons VISIBLES, hors
// ligne, de façon déterministe. C'est l'option 3 de #38, et le contraire des
// deux autres : rien n'est deviné par un modèle, rien ne part sur le réseau,
// et la même phrase rend toujours la même requête.
//
// # Ce n'est pas un troisième analyseur
//
// La contrainte de la fiche est explicite : sans elle, ce fichier serait une
// troisième grammaire à tenir en phase avec les deux qui existent. Il n'en est
// pas une, parce qu'il ne rend pas des FILTRES : il rend des JETONS, que Parse
// lit ensuite. Une intention qui produit `E>80 gt:holding` produit exactement
// ce qu'un utilisateur aurait tapé, et la suite du chemin est la même.
//
// # Ce qui est rendu, et pourquoi il y a trois listes
//
// Traduire à moitié en silence est le seul vrai danger d'une couche comme
// celle-ci : l'utilisateur croit avoir demandé quelque chose que la requête ne
// demande pas. Intent porte donc, à côté des jetons, ce qui a été COMPRIS et
// ce qui a été IGNORÉ — et l'interface montre les deux avant de chercher.
//
// # Deux intentions ne sont pas des jetons, et c'est dit
//
// « de videau » et « au score » ne décrivent pas un filtre mais le PLATEAU de
// recherche : le type de décision et le score y sont posés sur le damier, pas
// écrits dans la ligne de commande. Elles sont donc rendues à part, dans
// Board, plutôt que traduites en un jeton qui ne voudrait pas dire cela.

// BoardHint is an intention that sets the SEARCH BOARD rather than a token.
type BoardHint struct {
	// Decision is "checker", "cube" or "" — the kind of decision asked for.
	Decision string `json:"decision,omitempty"`
	// Score is "match", "money" or "".
	Score string `json:"score,omitempty"`
}

// Intent is the result of reading a phrase.
type Intent struct {
	// Tokens is the query, ready for Parse. Sorted for stability so the same
	// phrase always renders the same line.
	Tokens []string `json:"tokens"`
	// Board carries the two intentions that are not tokens.
	Board BoardHint `json:"board"`
	// Matched names the phrases that were understood, in the vocabulary's own
	// wording, so the interface can show what it did with the sentence.
	Matched []string `json:"matched"`
	// Ignored is every word nothing claimed. Shown, never dropped silently.
	Ignored []string `json:"ignored"`
}

// intentRule is one entry of the vocabulary: the phrases that trigger it and
// what it produces.
type intentRule struct {
	// phrases are lower-cased, accent-folded, space-separated forms. The
	// longest matching phrase wins, so "grosse erreur" beats "erreur".
	phrases []string
	tokens  []string
	board   BoardHint
	// label names the rule for the Matched list.
	label string
}

// intentVocabulary is French and English, as the fiche asks, and nothing else.
// Every threshold it hard-codes is one the documentation already states:
// blunder at 80 millipoints and error at 20 are gnubg's own marks, which is
// what makes "mes blunders" mean the same thing here as in the statistics.
var intentVocabulary = []intentRule{
	{label: "blunder", phrases: []string{"blunder", "blunders", "bourde", "bourdes", "grosse erreur", "grosses erreurs", "big mistake", "big mistakes"}, tokens: []string{"E>80"}},
	{label: "erreur", phrases: []string{"erreur", "erreurs", "error", "errors", "mistake", "mistakes"}, tokens: []string{"E>20"}},
	{label: "videau", phrases: []string{"videau", "cube", "doublement", "doubling"}, board: BoardHint{Decision: "cube"}},
	{label: "pions", phrases: []string{"pions", "coup", "coups", "checker", "checkers", "move", "moves"}, board: BoardHint{Decision: "checker"}},
	{label: "au score", phrases: []string{"au score", "en match", "match play", "at a score"}, board: BoardHint{Score: "match"}},
	{label: "en argent", phrases: []string{"en argent", "money", "money game", "partie d argent"}, board: BoardHint{Score: "money"}},
	{label: "course", phrases: []string{"course", "en course", "race", "in a race"}, tokens: []string{"ph:race"}},
	{label: "ouverture", phrases: []string{"ouverture", "opening"}, tokens: []string{"ph:opening"}},
	{label: "milieu de partie", phrases: []string{"milieu de partie", "milieu", "middlegame"}, tokens: []string{"ph:middlegame"}},
	{label: "sortie", phrases: []string{"sortie", "sortie des pions", "bearoff", "bear off"}, tokens: []string{"ph:bearoff"}},
	{label: "holding", phrases: []string{"holding", "holding game", "jeu de tenue"}, tokens: []string{"gt:holding"}},
	{label: "backgame", phrases: []string{"backgame", "back game", "jeu arriere"}, tokens: []string{"gt:backgame"}},
	{label: "blitz", phrases: []string{"blitz", "attaque"}, tokens: []string{"gt:blitz"}},
	{label: "amorces", phrases: []string{"prime contre prime", "amorces", "prime vs prime"}, tokens: []string{"gt:primevprime"}},
	{label: "sans contact", phrases: []string{"sans contact", "no contact"}, tokens: []string{"nc"}},
	{label: "marquees", phrases: []string{"marquee", "marquees", "flaggee", "flaggees", "flagged"}, tokens: []string{"fl"}},
	{label: "commentees", phrases: []string{"commentee", "commentees", "avec commentaire", "commented", "with a comment"}, tokens: []string{"co"}},
	{label: "importees seules", phrases: []string{"importee seule", "importees seules", "individual", "individually imported"}, tokens: []string{"i"}},
}

// TranslateIntent reads a phrase and returns the query it means.
//
// It never guesses: a word that matches nothing is reported, not approximated.
// The reason is the same one that made a spell-corrector a bad idea for a
// closed vocabulary — a search that silently means something else is worse
// than a search that says it did not understand.
func TranslateIntent(text string) Intent {
	words := splitIntentWords(text)
	used := make([]bool, len(words))
	var out Intent
	seen := map[string]bool{}

	// Longest phrase first, so "grosse erreur" is claimed before "erreur" and
	// "milieu de partie" before "milieu".
	type candidate struct {
		rule   *intentRule
		phrase []string
	}
	var candidates []candidate
	for i := range intentVocabulary {
		for _, phrase := range intentVocabulary[i].phrases {
			candidates = append(candidates, candidate{&intentVocabulary[i], strings.Fields(phrase)})
		}
	}
	sort.SliceStable(candidates, func(a, b int) bool {
		return len(candidates[a].phrase) > len(candidates[b].phrase)
	})

	for _, c := range candidates {
		for start := 0; start+len(c.phrase) <= len(words); start++ {
			if !matchesAt(words, used, start, c.phrase) {
				continue
			}
			for k := range c.phrase {
				used[start+k] = true
			}
			if !seen[c.rule.label] {
				seen[c.rule.label] = true
				out.Matched = append(out.Matched, c.rule.label)
			}
			for _, tok := range c.rule.tokens {
				if !containsString(out.Tokens, tok) {
					out.Tokens = append(out.Tokens, tok)
				}
			}
			// The LAST board hint of a kind wins, the way the last word of a
			// sentence does: "au score" after "en argent" means at a score.
			if c.rule.board.Decision != "" {
				out.Board.Decision = c.rule.board.Decision
			}
			if c.rule.board.Score != "" {
				out.Board.Score = c.rule.board.Score
			}
		}
	}

	for i, w := range words {
		if !used[i] && !isFillerWord(w) {
			out.Ignored = append(out.Ignored, w)
		}
	}
	sort.Strings(out.Tokens)
	return out
}

// matchesAt reports whether phrase sits at words[start:], on words nothing has
// claimed yet.
func matchesAt(words []string, used []bool, start int, phrase []string) bool {
	for k, p := range phrase {
		if used[start+k] || words[start+k] != p {
			return false
		}
	}
	return true
}

// fillerWords are the words a sentence needs and a query does not. They are
// not "ignored" in the sense the interface reports — reporting "de" and "mes"
// as unrecognised would bury the one word that really was.
var fillerWords = map[string]bool{
	"mes": true, "mon": true, "ma": true, "les": true, "le": true, "la": true,
	"de": true, "du": true, "des": true, "d": true, "en": true, "a": true,
	"au": true, "aux": true, "et": true, "ou": true, "un": true, "une": true,
	"my": true, "the": true, "of": true, "in": true, "at": true, "and": true,
	"or": true, "a_": true, "an": true, "with": true, "on": true, "for": true,
	"show": true, "montre": true, "moi": true, "me_": true, "cherche": true,
	"find": true, "search": true, "positions": true, "position": true,
}

func isFillerWord(w string) bool { return fillerWords[w] }

// splitIntentWords lower-cases, strips accents and punctuation, and splits on
// spaces. Accent folding is what lets "marquées" and "marquees" be the same
// word — a user types what their keyboard makes easy.
func splitIntentWords(text string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(foldAccent(r))
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}

// foldAccent maps the accented Latin letters French uses onto their base. A
// table rather than a normalisation library: the alphabet is closed, and the
// dependency would be larger than the problem.
func foldAccent(r rune) rune {
	switch r {
	case 'à', 'â', 'ä', 'á', 'ã', 'å':
		return 'a'
	case 'ç':
		return 'c'
	case 'è', 'é', 'ê', 'ë':
		return 'e'
	case 'î', 'ï', 'ì', 'í':
		return 'i'
	case 'ô', 'ö', 'ò', 'ó', 'õ':
		return 'o'
	case 'ù', 'û', 'ü', 'ú':
		return 'u'
	case 'ÿ':
		return 'y'
	case 'ñ':
		return 'n'
	}
	return r
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
