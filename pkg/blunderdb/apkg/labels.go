package apkg

// DefaultLanguage is used when the requested language has no catalogue.
const DefaultLanguage = "en"

// labels are the words a card is written in; %d stands for a number.
var labels = map[string]map[string]string{
	"fr": {
		"money": "Partie libre", "score": "Score", "away": "%d away", "cube": "Videau",
		"centred": "au centre", "ownOnRoll": "au joueur au trait", "ownOpp": "à l'adversaire",
		"toPlay": "%d-%d à jouer", "cubeDecision": "Décision de videau", "onRoll": "Au trait : le joueur du bas",
		"nd": "Pas de double, prise", "dt": "Double, prise", "dp": "Double, refus", "tg": "Trop bon pour doubler, refus",
		"ndEq": "Pas de double", "dtEq": "Double/prise", "dpEq": "Double/refus",
		"equity": "Équité", "played": "Joué", "error": "erreur", "noAnalysis": "Pas d'analyse",
	},
	"en": {
		"money": "Money game", "score": "Score", "away": "%d away", "cube": "Cube",
		"centred": "centred", "ownOnRoll": "owned by the player on roll", "ownOpp": "owned by the opponent",
		"toPlay": "%d-%d to play", "cubeDecision": "Cube decision", "onRoll": "On roll: the player at the bottom",
		"nd": "No double, take", "dt": "Double, take", "dp": "Double, pass", "tg": "Too good to double, pass",
		"ndEq": "No double", "dtEq": "Double/take", "dpEq": "Double/pass",
		"equity": "Equity", "played": "Played", "error": "error", "noAnalysis": "No analysis",
	},
	"de": {
		"money": "Money-Spiel", "score": "Spielstand", "away": "%d away", "cube": "Doppler",
		"centred": "in der Mitte", "ownOnRoll": "beim Spieler am Zug", "ownOpp": "beim Gegner",
		"toPlay": "%d-%d zu ziehen", "cubeDecision": "Dopplerentscheidung", "onRoll": "Am Zug: der untere Spieler",
		"nd": "Kein Doppel, annehmen", "dt": "Doppeln, annehmen", "dp": "Doppeln, aufgeben", "tg": "Zu gut zum Doppeln, aufgeben",
		"ndEq": "Kein Doppel", "dtEq": "Doppeln/annehmen", "dpEq": "Doppeln/aufgeben",
		"equity": "Equity", "played": "Gespielt", "error": "Fehler", "noAnalysis": "Keine Analyse",
	},
	"el": {
		"money": "Παιχνίδι money", "score": "Σκορ", "away": "%d away", "cube": "Κύβος",
		"centred": "στο κέντρο", "ownOnRoll": "του παίκτη που παίζει", "ownOpp": "του αντιπάλου",
		"toPlay": "%d-%d για παίξιμο", "cubeDecision": "Απόφαση κύβου", "onRoll": "Παίζει: ο παίκτης κάτω",
		"nd": "Όχι διπλό, δεκτό", "dt": "Διπλό, δεκτό", "dp": "Διπλό, πάσο", "tg": "Πολύ καλό για διπλό, πάσο",
		"ndEq": "Όχι διπλό", "dtEq": "Διπλό/δεκτό", "dpEq": "Διπλό/πάσο",
		"equity": "Equity", "played": "Παίχτηκε", "error": "σφάλμα", "noAnalysis": "Χωρίς ανάλυση",
	},
	"es": {
		"money": "Partida libre", "score": "Marcador", "away": "%d away", "cube": "Cubo",
		"centred": "en el centro", "ownOnRoll": "del jugador al turno", "ownOpp": "del rival",
		"toPlay": "%d-%d por jugar", "cubeDecision": "Decisión de cubo", "onRoll": "Al turno: el jugador de abajo",
		"nd": "No doblar, aceptar", "dt": "Doblar, aceptar", "dp": "Doblar, rechazar", "tg": "Demasiado bueno para doblar, rechazar",
		"ndEq": "No doblar", "dtEq": "Doblar/aceptar", "dpEq": "Doblar/rechazar",
		"equity": "Equity", "played": "Jugado", "error": "error", "noAnalysis": "Sin análisis",
	},
	"fi": {
		"money": "Rahapeli", "score": "Tilanne", "away": "%d away", "cube": "Kuutio",
		"centred": "keskellä", "ownOnRoll": "vuorossa olevalla", "ownOpp": "vastustajalla",
		"toPlay": "%d-%d pelattavana", "cubeDecision": "Kuutiopäätös", "onRoll": "Vuorossa: alempi pelaaja",
		"nd": "Ei tuplausta, otto", "dt": "Tuplaus, otto", "dp": "Tuplaus, luovutus", "tg": "Liian hyvä tuplattavaksi, luovutus",
		"ndEq": "Ei tuplausta", "dtEq": "Tuplaus/otto", "dpEq": "Tuplaus/luovutus",
		"equity": "Equity", "played": "Pelattu", "error": "virhe", "noAnalysis": "Ei analyysiä",
	},
	"it": {
		"money": "Partita libera", "score": "Punteggio", "away": "%d away", "cube": "Cubo",
		"centred": "al centro", "ownOnRoll": "del giocatore di turno", "ownOpp": "dell'avversario",
		"toPlay": "%d-%d da giocare", "cubeDecision": "Decisione di cubo", "onRoll": "Di turno: il giocatore in basso",
		"nd": "Non raddoppiare, accettare", "dt": "Raddoppio, accettare", "dp": "Raddoppio, rifiutare", "tg": "Troppo buono per raddoppiare, rifiutare",
		"ndEq": "Non raddoppiare", "dtEq": "Raddoppio/accettare", "dpEq": "Raddoppio/rifiutare",
		"equity": "Equity", "played": "Giocato", "error": "errore", "noAnalysis": "Nessuna analisi",
	},
	"ja": {
		"money": "マネーゲーム", "score": "スコア", "away": "%d away", "cube": "キューブ",
		"centred": "中央", "ownOnRoll": "手番のプレイヤー側", "ownOpp": "相手側",
		"toPlay": "%d-%d を動かす", "cubeDecision": "キューブの判断", "onRoll": "手番：下側のプレイヤー",
		"nd": "ノーダブル、テイク", "dt": "ダブル、テイク", "dp": "ダブル、パス", "tg": "ダブルには良すぎる、パス",
		"ndEq": "ノーダブル", "dtEq": "ダブル/テイク", "dpEq": "ダブル/パス",
		"equity": "エクイティ", "played": "実際の手", "error": "誤差", "noAnalysis": "解析なし",
	},
	"ru": {
		"money": "Игра на деньги", "score": "Счёт", "away": "%d away", "cube": "Куб",
		"centred": "в центре", "ownOnRoll": "у игрока на ходу", "ownOpp": "у соперника",
		"toPlay": "%d-%d — ход", "cubeDecision": "Решение по кубу", "onRoll": "На ходу: нижний игрок",
		"nd": "Без удвоения, принять", "dt": "Удвоение, принять", "dp": "Удвоение, сдаться", "tg": "Слишком хорошо для удвоения, сдаться",
		"ndEq": "Без удвоения", "dtEq": "Удвоение/принять", "dpEq": "Удвоение/сдаться",
		"equity": "Эквити", "played": "Сыграно", "error": "ошибка", "noAnalysis": "Нет анализа",
	},
}

// Languages lists the languages a card can be written in.
func Languages() []string {
	return []string{"fr", "en", "de", "el", "es", "fi", "it", "ja", "ru"}
}

func label(lang, key string) string {
	if m, ok := labels[lang]; ok {
		return m[key]
	}
	return labels[DefaultLanguage][key]
}
