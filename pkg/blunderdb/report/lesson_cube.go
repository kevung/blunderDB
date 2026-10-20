package report

import "strings"

// cubeWords translate the pieces a cube action is written with. The stored
// label is free text from the importers ("Double, Take", "No Double",
// "Too good to double, pass"); cubeActionText splits it into these pieces and
// leaves anything it does not recognise as it was written.
var cubeWords = map[string]map[string]string{
	"fr": {"double": "Double", "nodouble": "Pas de double", "redouble": "Redouble", "noredouble": "Pas de redouble", "toogood": "Trop bon", "take": "Prise", "pass": "Passe", "beaver": "Beaver", "played": "Joué", "colon": " :"},
	"en": {"double": "Double", "nodouble": "No double", "redouble": "Redouble", "noredouble": "No redouble", "toogood": "Too good", "take": "Take", "pass": "Pass", "beaver": "Beaver", "played": "Played", "colon": ":"},
	"de": {"double": "Doppeln", "nodouble": "Nicht doppeln", "redouble": "Redoppeln", "noredouble": "Nicht redoppeln", "toogood": "Zu gut", "take": "Annehmen", "pass": "Passen", "beaver": "Beaver", "played": "Gespielt", "colon": ":"},
	"el": {"double": "Διπλασιασμός", "nodouble": "Όχι διπλασιασμός", "redouble": "Επαναδιπλασιασμός", "noredouble": "Όχι επαναδιπλασιασμός", "toogood": "Πολύ καλή θέση", "take": "Αποδοχή", "pass": "Πάσο", "beaver": "Beaver", "played": "Παίχτηκε", "colon": ":"},
	"es": {"double": "Doblar", "nodouble": "No doblar", "redouble": "Redoblar", "noredouble": "No redoblar", "toogood": "Demasiado bueno", "take": "Aceptar", "pass": "Rechazar", "beaver": "Beaver", "played": "Jugado", "colon": ":"},
	"fi": {"double": "Tuplaus", "nodouble": "Ei tuplausta", "redouble": "Uudelleentuplaus", "noredouble": "Ei uudelleentuplausta", "toogood": "Liian hyvä", "take": "Hyväksy", "pass": "Hylkää", "beaver": "Beaver", "played": "Pelattu", "colon": ":"},
	"it": {"double": "Raddoppio", "nodouble": "Nessun raddoppio", "redouble": "Ridoppio", "noredouble": "Nessun ridoppio", "toogood": "Troppo buono", "take": "Accetta", "pass": "Rifiuta", "beaver": "Beaver", "played": "Giocato", "colon": ":"},
	"ja": {"double": "ダブル", "nodouble": "ノーダブル", "redouble": "リダブル", "noredouble": "ノーリダブル", "toogood": "トゥーグッド", "take": "テイク", "pass": "パス", "beaver": "ビーバー", "played": "実際の手", "colon": ":"},
	"ru": {"double": "Удвоение", "nodouble": "Без удвоения", "redouble": "Переудвоение", "noredouble": "Без переудвоения", "toogood": "Слишком хорошо", "take": "Принять", "pass": "Отказ", "beaver": "Бивер", "played": "Сыграно", "colon": ":"},
}

func cubeWord(lang, key string) string {
	if m, ok := cubeWords[lang]; ok {
		return m[key]
	}
	return cubeWords[DefaultLanguage][key]
}

// cubeHead and cubeReply name the two halves of a label, by their lowercase
// spelling.
var cubeHead = map[string]string{
	"double": "double", "no double": "nodouble", "nodouble": "nodouble",
	"redouble": "redouble", "no redouble": "noredouble",
	"too good": "toogood", "too good to double": "toogood", "too good to redouble": "toogood",
	"tg": "toogood",
}

var cubeReply = map[string]string{
	"take": "take", "pass": "pass", "drop": "pass", "beaver": "beaver",
}

// cubeActionText writes a stored cube action in lang.
func cubeActionText(lang, label string) string {
	norm := strings.ToLower(strings.TrimSpace(label))
	norm = strings.ReplaceAll(norm, "/", ",")
	parts := strings.Split(norm, ",")
	for i := range parts {
		parts[i] = strings.Join(strings.Fields(parts[i]), " ")
	}
	if len(parts) == 0 || len(parts) > 2 {
		return label
	}
	head, ok := cubeHead[parts[0]]
	if !ok {
		// "Take", "Pass", "Beaver" alone answer a double.
		if r, isReply := cubeReply[parts[0]]; isReply && len(parts) == 1 {
			return cubeWord(lang, r)
		}
		return label
	}
	out := cubeWord(lang, head)
	if len(parts) == 2 {
		r, ok := cubeReply[parts[1]]
		if !ok {
			return label
		}
		out += ", " + cubeWord(lang, r)
	}
	return out
}
