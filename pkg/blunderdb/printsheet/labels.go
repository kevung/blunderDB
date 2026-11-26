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
		"reimportNote":  "Transcription, scores, temps, marques, commentaires. Analyses recalculées à l'import.",
		"noCode":        "Pas de code de réimport : {err}",
		"reimportRange": "Réimport des positions {a} à {b}", "reimportListNote": "Positions seules (ArkID) : ni analyses, ni commentaires. Chaque bloc se relit seul ; un code perdu se reconstruit par les autres.",
		"listTitle": "Liste de positions", "nPositions": "{n} positions", "printedOn": "imprimée le {d}", "versoNote": "réponses au verso, même numéro", "answers": "Réponses", "lgPips": "pips trait–adversaire", "lgAnalysis": "<b>meilleur coup</b> · autre choix et sa perte en millièmes · gain/gammon en %",
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
		"reimportNote":  "Transcription, scores, times, marks, comments. Analyses are computed again on import.",
		"noCode":        "No reimport code: {err}",
		"reimportRange": "Reimport of positions {a} to {b}", "reimportListNote": "Positions only (ArkID): no analyses, no comments. Each block reads back alone; a lost code is rebuilt from the others.",
		"listTitle": "Position list", "nPositions": "{n} positions", "printedOn": "printed {d}", "versoNote": "answers overleaf, same number", "answers": "Answers", "lgPips": "pips on roll–opponent", "lgAnalysis": "<b>best play</b> · other choice and its loss in thousandths · win/gammon in %",
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
		"reimportNote":  "Transkription, Stände, Zeiten, Markierungen, Kommentare. Analysen werden beim Import neu berechnet.",
		"noCode":        "Kein Code zum Wiedereinlesen: {err}",
		"reimportRange": "Wiedereinlesen der Positionen {a} bis {b}", "reimportListNote": "Nur Positionen (ArkID): keine Analysen, keine Kommentare. Jeder Block wird allein gelesen; ein verlorener Code wird aus den anderen wiederhergestellt.",
		"listTitle": "Stellungsliste", "nPositions": "{n} Stellungen", "printedOn": "gedruckt am {d}", "versoNote": "Antworten auf der Rückseite, gleiche Nummer", "answers": "Antworten", "lgPips": "Pips am Zug–Gegner", "lgAnalysis": "<b>bester Zug</b> · andere Wahl und ihr Verlust in Tausendsteln · Sieg/Gammon in %",
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
		"reimportNote":  "Καταγραφή, σκορ, χρόνοι, σημάνσεις, σχόλια. Οι αναλύσεις υπολογίζονται ξανά στην εισαγωγή.",
		"noCode":        "Χωρίς κωδικό επανεισαγωγής: {err}",
		"reimportRange": "Επανεισαγωγή θέσεων {a} έως {b}", "reimportListNote": "Μόνο θέσεις (ArkID): χωρίς αναλύσεις, χωρίς σχόλια. Κάθε μπλοκ διαβάζεται μόνο του· ένας χαμένος κωδικός ανακτάται από τους άλλους.",
		"listTitle": "Λίστα θέσεων", "nPositions": "{n} θέσεις", "printedOn": "τυπώθηκε {d}", "versoNote": "απαντήσεις στο πίσω μέρος, ίδιος αριθμός", "answers": "Απαντήσεις", "lgPips": "pips παίκτη που παίζει–αντιπάλου", "lgAnalysis": "<b>καλύτερη κίνηση</b> · άλλη επιλογή και η απώλειά της σε χιλιοστά · νίκη/γκάμον σε %",
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
		"reimportNote":  "Transcripción, marcadores, tiempos, marcas, comentarios. Los análisis se recalculan al importar.",
		"noCode":        "Sin código de reimportación: {err}",
		"reimportRange": "Reimportación de las posiciones {a} a {b}", "reimportListNote": "Solo posiciones (ArkID): sin análisis ni comentarios. Cada bloque se lee solo; un código perdido se reconstruye con los demás.",
		"listTitle": "Lista de posiciones", "nPositions": "{n} posiciones", "printedOn": "impresa el {d}", "versoNote": "respuestas al dorso, mismo número", "answers": "Respuestas", "lgPips": "pips en turno–rival", "lgAnalysis": "<b>mejor jugada</b> · otra opción y su pérdida en milésimas · victoria/gammon en %",
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
		"reimportNote":  "Kirjaus, tilanteet, ajat, merkinnät, kommentit. Analyysit lasketaan uudelleen tuonnissa.",
		"noCode":        "Ei uudelleentuontikoodia: {err}",
		"reimportRange": "Asemien {a}–{b} uudelleentuonti", "reimportListNote": "Vain asemat (ArkID): ei analyyseja eikä kommentteja. Jokainen lohko luetaan yksinään; kadonnut koodi rakennetaan muista.",
		"listTitle": "Asemaluettelo", "nPositions": "{n} asemaa", "printedOn": "tulostettu {d}", "versoNote": "vastaukset kääntöpuolella, sama numero", "answers": "Vastaukset", "lgPips": "pipit vuorossa–vastustaja", "lgAnalysis": "<b>paras siirto</b> · toinen vaihtoehto ja sen menetys tuhannesosina · voitto/gammon %",
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
		"reimportNote":  "Trascrizione, punteggi, tempi, segni, commenti. Le analisi si ricalcolano all'importazione.",
		"noCode":        "Nessun codice di reimportazione: {err}",
		"reimportRange": "Reimportazione delle posizioni da {a} a {b}", "reimportListNote": "Solo posizioni (ArkID): niente analisi né commenti. Ogni blocco si rilegge da solo; un codice perso si ricostruisce dagli altri.",
		"listTitle": "Elenco di posizioni", "nPositions": "{n} posizioni", "printedOn": "stampato il {d}", "versoNote": "risposte sul retro, stesso numero", "answers": "Risposte", "lgPips": "pip di turno–avversario", "lgAnalysis": "<b>mossa migliore</b> · altra scelta e la sua perdita in millesimi · vittoria/gammon in %",
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
		"reimportNote":  "棋譜、スコア、時間、マーク、コメント。分析はインポート時に再計算されます。",
		"noCode":        "再インポート用コードなし：{err}",
		"reimportRange": "局面 {a}〜{b} の再インポート", "reimportListNote": "局面のみ（ArkID）：分析・コメントは含みません。各ブロックは単独で読み込めます。失われたコードは他のコードから復元されます。",
		"listTitle": "局面リスト", "nPositions": "{n} 局面", "printedOn": "{d} 印刷", "versoNote": "解答は裏面、同じ番号", "answers": "解答", "lgPips": "ピップ 手番側–相手", "lgAnalysis": "<b>最善手</b> · 別の選択とその損失（千分率） · 勝率/ギャモン率 %",
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
		"reimportNote":  "Запись, счёт, время, отметки, комментарии. Анализ пересчитывается при импорте.",
		"noCode":        "Нет кода повторного импорта: {err}",
		"reimportRange": "Повторный импорт позиций {a}–{b}", "reimportListNote": "Только позиции (ArkID): без анализа и комментариев. Каждый блок читается отдельно; потерянный код восстанавливается по остальным.",
		"listTitle": "Список позиций", "nPositions": "{n} позиций", "printedOn": "напечатано {d}", "versoNote": "ответы на обороте, тот же номер", "answers": "Ответы", "lgPips": "пипы на ходу–соперник", "lgAnalysis": "<b>лучший ход</b> · другой выбор и его потеря в тысячных · победа/гаммон в %",
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
