package printsheet

import "strings"

// DefaultLanguage is used when the requested language has no catalogue, and
// for a key a catalogue lacks.
const DefaultLanguage = "en"

// Languages lists the languages a sheet can be written in.
func Languages() []string {
	return []string{"fr", "en", "de", "el", "es", "fi", "it", "ja", "ru"}
}

var labels = map[string]map[string]string{
	"fr": {
		"title": "Feuille de match", "length": "Match en {n} points", "money": "Partie libre",
		"games": "{n} parties", "plies": "{n} coups", "errors": "Erreurs", "blunders": "Blunders", "time": "Temps",
		"game": "Partie", "lgError": "erreur", "lgBlunder": "blunder", "lgRef": "blunder détaillé",
		"lgRemark": "position remarquable", "lgSlow": "réflexion longue",
		"lgArk":     "ArkID : A=1 … W=23, Y=24, Z=barre, x=videau ; minuscules = joueur au trait ; (portes) puis pions en plus et pions seuls",
		"positions": "Positions", "blundersOf": "{n} blunders de {p}", "blundersBoth": "{n} blunders (les deux joueurs)",
		"remarks": "{n} remarquables", "played": "joué", "best": "meilleur",
		"Double": "Double", "No Double": "Pas de double", "Take": "Prend", "Pass": "Passe", "dance": "ne joue pas",
		"reimport": "Réimport", "bytes": "{n} octets",
		"reimportNote": "Transcription, scores, temps, marques, commentaires. Analyses recalculées à l'import.",
		"noCode":       "Pas de code de réimport : {err}",
	},
	"en": {
		"title": "Match sheet", "length": "{n}-point match", "money": "Money session",
		"games": "{n} games", "plies": "{n} moves", "errors": "Errors", "blunders": "Blunders", "time": "Time",
		"game": "Game", "lgError": "error", "lgBlunder": "blunder", "lgRef": "detailed blunder",
		"lgRemark": "remarkable position", "lgSlow": "long thought",
		"lgArk":     "ArkID: A=1 … W=23, Y=24, Z=bar, x=cube; lower case = player on roll; (points) then extra checkers and blots",
		"positions": "Positions", "blundersOf": "{n} blunders by {p}", "blundersBoth": "{n} blunders (both players)",
		"remarks": "{n} remarkable", "played": "played", "best": "best",
		"Double": "Double", "No Double": "No double", "Take": "Take", "Pass": "Pass", "dance": "cannot move",
		"reimport": "Reimport", "bytes": "{n} bytes",
		"reimportNote": "Transcription, scores, times, marks, comments. Analyses are computed again on import.",
		"noCode":       "No reimport code: {err}",
	},
	"de": {
		"title": "Matchblatt", "length": "Match auf {n} Punkte", "money": "Geldspiel",
		"games": "{n} Spiele", "plies": "{n} Züge", "errors": "Fehler", "blunders": "Patzer", "time": "Zeit",
		"game": "Spiel", "lgError": "Fehler", "lgBlunder": "Patzer", "lgRef": "ausführlicher Patzer",
		"lgRemark": "bemerkenswerte Stellung", "lgSlow": "lange Bedenkzeit",
		"lgArk":     "ArkID: A=1 … W=23, Y=24, Z=Bar, x=Würfel; Kleinbuchstaben = Spieler am Zug; (Punkte), dann weitere Steine und Blots",
		"positions": "Stellungen", "blundersOf": "{n} Patzer von {p}", "blundersBoth": "{n} Patzer (beide Spieler)",
		"remarks": "{n} bemerkenswert", "played": "gespielt", "best": "beste",
		"Double": "Doppeln", "No Double": "Nicht doppeln", "Take": "Annehmen", "Pass": "Aufgeben", "dance": "kann nicht ziehen",
		"reimport": "Wiedereinlesen", "bytes": "{n} Bytes",
		"reimportNote": "Transkription, Stände, Zeiten, Markierungen, Kommentare. Analysen werden beim Import neu berechnet.",
		"noCode":       "Kein Code zum Wiedereinlesen: {err}",
	},
	"el": {
		"title": "Φύλλο αγώνα", "length": "Αγώνας {n} πόντων", "money": "Παιχνίδι money",
		"games": "{n} παρτίδες", "plies": "{n} κινήσεις", "errors": "Λάθη", "blunders": "Γκάφες", "time": "Χρόνος",
		"game": "Παρτίδα", "lgError": "λάθος", "lgBlunder": "γκάφα", "lgRef": "αναλυτική γκάφα",
		"lgRemark": "αξιοσημείωτη θέση", "lgSlow": "μακρά σκέψη",
		"lgArk":     "ArkID: A=1 … W=23, Y=24, Z=μπάρα, x=κύβος· πεζά = παίκτης που παίζει· (πόρτες) και μετά επιπλέον πούλια και μονά",
		"positions": "Θέσεις", "blundersOf": "{n} γκάφες του {p}", "blundersBoth": "{n} γκάφες (και οι δύο παίκτες)",
		"remarks": "{n} αξιοσημείωτες", "played": "παίχτηκε", "best": "καλύτερη",
		"Double": "Διπλασιασμός", "No Double": "Όχι διπλασιασμός", "Take": "Αποδοχή", "Pass": "Παραίτηση", "dance": "δεν παίζει",
		"reimport": "Επανεισαγωγή", "bytes": "{n} byte",
		"reimportNote": "Καταγραφή, σκορ, χρόνοι, σημάνσεις, σχόλια. Οι αναλύσεις υπολογίζονται ξανά στην εισαγωγή.",
		"noCode":       "Χωρίς κωδικό επανεισαγωγής: {err}",
	},
	"es": {
		"title": "Hoja de partido", "length": "Partido a {n} puntos", "money": "Partida libre",
		"games": "{n} partidas", "plies": "{n} jugadas", "errors": "Errores", "blunders": "Blunders", "time": "Tiempo",
		"game": "Partida", "lgError": "error", "lgBlunder": "blunder", "lgRef": "blunder detallado",
		"lgRemark": "posición destacada", "lgSlow": "reflexión larga",
		"lgArk":     "ArkID: A=1 … W=23, Y=24, Z=barra, x=cubo; minúsculas = jugador al turno; (puntos) y luego fichas de más y fichas sueltas",
		"positions": "Posiciones", "blundersOf": "{n} blunders de {p}", "blundersBoth": "{n} blunders (ambos jugadores)",
		"remarks": "{n} destacadas", "played": "jugado", "best": "mejor",
		"Double": "Dobla", "No Double": "No dobla", "Take": "Acepta", "Pass": "Rechaza", "dance": "no juega",
		"reimport": "Reimportación", "bytes": "{n} bytes",
		"reimportNote": "Transcripción, marcadores, tiempos, marcas, comentarios. Los análisis se recalculan al importar.",
		"noCode":       "Sin código de reimportación: {err}",
	},
	"fi": {
		"title": "Ottelulomake", "length": "{n} pisteen ottelu", "money": "Rahapeli",
		"games": "{n} peliä", "plies": "{n} siirtoa", "errors": "Virheet", "blunders": "Mokat", "time": "Aika",
		"game": "Peli", "lgError": "virhe", "lgBlunder": "moka", "lgRef": "eritelty moka",
		"lgRemark": "huomattava asema", "lgSlow": "pitkä harkinta",
		"lgArk":     "ArkID: A=1 … W=23, Y=24, Z=palkki, x=kuutio; pienet kirjaimet = vuorossa oleva pelaaja; (pisteet), sitten lisänappulat ja yksinäiset",
		"positions": "Asemat", "blundersOf": "{n} mokaa: {p}", "blundersBoth": "{n} mokaa (molemmat pelaajat)",
		"remarks": "{n} huomattavaa", "played": "pelattu", "best": "paras",
		"Double": "Tuplaus", "No Double": "Ei tuplausta", "Take": "Otto", "Pass": "Luovutus", "dance": "ei voi siirtää",
		"reimport": "Uudelleentuonti", "bytes": "{n} tavua",
		"reimportNote": "Kirjaus, tilanteet, ajat, merkinnät, kommentit. Analyysit lasketaan uudelleen tuonnissa.",
		"noCode":       "Ei uudelleentuontikoodia: {err}",
	},
	"it": {
		"title": "Foglio di partita", "length": "Partita ai {n} punti", "money": "Partita libera",
		"games": "{n} partite", "plies": "{n} mosse", "errors": "Errori", "blunders": "Blunder", "time": "Tempo",
		"game": "Partita", "lgError": "errore", "lgBlunder": "blunder", "lgRef": "blunder dettagliato",
		"lgRemark": "posizione notevole", "lgSlow": "riflessione lunga",
		"lgArk":     "ArkID: A=1 … W=23, Y=24, Z=barra, x=cubo; minuscole = giocatore di turno; (punti) poi pedine in più e pedine singole",
		"positions": "Posizioni", "blundersOf": "{n} blunder di {p}", "blundersBoth": "{n} blunder (entrambi i giocatori)",
		"remarks": "{n} notevoli", "played": "giocata", "best": "migliore",
		"Double": "Raddoppia", "No Double": "Non raddoppia", "Take": "Accetta", "Pass": "Rifiuta", "dance": "non muove",
		"reimport": "Reimportazione", "bytes": "{n} byte",
		"reimportNote": "Trascrizione, punteggi, tempi, segni, commenti. Le analisi si ricalcolano all'importazione.",
		"noCode":       "Nessun codice di reimportazione: {err}",
	},
	"ja": {
		"title": "マッチシート", "length": "{n}ポイントマッチ", "money": "マネーゲーム",
		"games": "{n}ゲーム", "plies": "{n}手", "errors": "エラー", "blunders": "ブランダー", "time": "時間",
		"game": "ゲーム", "lgError": "エラー", "lgBlunder": "ブランダー", "lgRef": "詳細なブランダー",
		"lgRemark": "注目局面", "lgSlow": "長考",
		"lgArk":     "ArkID：A=1 … W=23、Y=24、Z=バー、x=キューブ。小文字＝手番のプレイヤー。（ポイント）の後に余分な駒とブロット",
		"positions": "局面", "blundersOf": "{p}のブランダー{n}件", "blundersBoth": "ブランダー{n}件（両プレイヤー）",
		"remarks": "注目{n}件", "played": "実戦", "best": "最善",
		"Double": "ダブル", "No Double": "ノーダブル", "Take": "テイク", "Pass": "パス", "dance": "動けない",
		"reimport": "再インポート", "bytes": "{n}バイト",
		"reimportNote": "棋譜、スコア、時間、マーク、コメント。分析はインポート時に再計算されます。",
		"noCode":       "再インポート用コードなし：{err}",
	},
	"ru": {
		"title": "Лист матча", "length": "Матч до {n} очков", "money": "Игра на деньги",
		"games": "{n} партий", "plies": "{n} ходов", "errors": "Ошибки", "blunders": "Грубые ошибки", "time": "Время",
		"game": "Партия", "lgError": "ошибка", "lgBlunder": "грубая ошибка", "lgRef": "разобранная грубая ошибка",
		"lgRemark": "примечательная позиция", "lgSlow": "долгое раздумье",
		"lgArk":     "ArkID: A=1 … W=23, Y=24, Z=бар, x=куб; строчные = игрок на ходу; (пункты), затем лишние шашки и одиночные",
		"positions": "Позиции", "blundersOf": "{n} грубых ошибок: {p}", "blundersBoth": "{n} грубых ошибок (оба игрока)",
		"remarks": "{n} примечательных", "played": "сыграно", "best": "лучший",
		"Double": "Удвоение", "No Double": "Без удвоения", "Take": "Принять", "Pass": "Сдаться", "dance": "нет хода",
		"reimport": "Повторный импорт", "bytes": "{n} байт",
		"reimportNote": "Запись, счёт, время, отметки, комментарии. Анализ пересчитывается при импорте.",
		"noCode":       "Нет кода повторного импорта: {err}",
	},
}

// catalogue returns the language's labels, the default one when it has none.
func catalogue(lang string) (string, map[string]string) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if c, ok := labels[lang]; ok {
		return lang, c
	}
	return DefaultLanguage, labels[DefaultLanguage]
}

// label reads key in c, falling back to the default language, and fills its
// {name} placeholders from pairs.
func label(c map[string]string, key string, pairs ...string) string {
	s, ok := c[key]
	if !ok {
		s = labels[DefaultLanguage][key]
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		s = strings.ReplaceAll(s, "{"+pairs[i]+"}", pairs[i+1])
	}
	return s
}
