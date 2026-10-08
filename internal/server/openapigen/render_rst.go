package openapigen

import (
	"fmt"
	"sort"
	"strings"
)

// rstIntro/rstOutro are the annex's only translatable prose: the route table
// is a code-block, which gettext skips, so adding a route never touches the
// eight .po catalogues.
const rstIntro = `.. _api_reference:

Contrat d'API
=============

Cette annexe est générée depuis le code source (` + "``go run ./cmd/openapi-gen``" + `,
voir ` + "``openapi.yaml``" + ` à la racine du dépôt pour le contrat complet, schémas
compris) : ne pas éditer directement, la prochaine génération écraserait toute
modification. Chaque famille regroupe les méthodes ` + "``POST /v1/<famille>.<méthode>``" + `
exposées par le démon (voir :ref:` + "`headless`" + `), avec leur forme de réponse
(JSON, ou NDJSON pour une liste en flux) et, quand elle existe, la mention de
l'en-tête ` + "``Idempotency-Key``" + ` optionnel.

`

const rstOutroTemplate = `
Idempotence
-----------

La plupart des méthodes n'ont besoin d'aucun mécanisme particulier : les
lectures sont sans effet de bord, et les écritures de ` + "``positions.*``" + ` sont
idempotentes dans leur effet grâce au hachage Zobrist du contenu — enregistrer
deux fois la même position renvoie la même ligne, jamais un doublon. %d méthodes
acceptent un en-tête ` + "``Idempotency-Key``" + ` optionnel : celles dont deux appels
sont deux effets distincts, et ` + "``positions.save``" + `, dont la réponse dit si
l'appel a créé la position (` + "``created``" + `) et le dirait faux si elle était
rejouée après une réponse perdue. Un appel rejoué avec la même clé renvoie le
résultat de la première tentative au lieu de répéter son effet — voir la marque
« (Idempotency-Key) » dans le tableau ci-dessus. Aucune autre méthode n'a besoin ou n'accepte cet en-tête.

Lectures conditionnelles
------------------------

Les méthodes marquées « (ETag) » rendent un en-tête ` + "``ETag``" + `. Le renvoyer dans
` + "``If-None-Match``" + ` obtient une réponse ` + "``304``" + ` sans corps tant que rien de ce que la
méthode lit n'a changé — voir :ref:` + "`headless_direction`" + `.

Repères d'une transcription
---------------------------

Un geste de ` + "``transcriptions.apply``" + ` peut porter l'instant de la vidéo où il est
fait, en millisecondes depuis le début du média : ` + "``TickMS``" + `, lu seulement quand
` + "``HasTick``" + ` vaut ` + "``true``" + ` (0 est un instant). ` + "``enter_die``" + ` en fait le repère du
jet (la première touche de dé compte), ` + "``enter_play``" + `, ` + "``validate``" + `, ` + "``dance``" + `,
` + "``double``" + `, ` + "``take``" + `, ` + "``pass``" + ` et ` + "``resign``" + ` le repère de l'action, sur une
action nouvelle seulement : une correction en place garde les repères qu'elle avait.
Le geste ` + "``set_timecode``" + ` pose ` + "``TickMS``" + ` (avec ` + "``HasTick``" + `) et ` + "``RollTickMS``" + ` (avec
` + "``HasRollTick``" + `) sur l'action du curseur ; une valeur négative efface le repère. Le
geste ` + "``set_video``" + ` attache la source vidéo nommée par ` + "``VideoSource``" + ` (URL http(s) ou
chemin local), une chaîne vide la détache ; ` + "``set_header``" + ` la pose aussi quand son
en-tête en nomme une, sans jamais l'effacer.

Le geste ` + "``seek_cursor``" + ` pose le curseur sur l'action de rang ` + "``At``" + `, la fin du
document au-delà de la dernière, sans rien écrire ni s'empiler sur la pile
d'annulation : c'est ainsi que le curseur suit une vidéo en lecture.
Une ` + "``decision_ms``" + ` accompagnée de ` + "``decision_estimated``" + ` est une estimation : un coup
sans repère d'action, mesuré du jet au premier instant de l'action suivante.

L'état rendu porte la source dans ` + "``header.video_source``" + `, les repères de chaque
action dans ` + "``roll_tick_ms``" + ` et ` + "``tick_ms``" + `, et, sur chaque action annotée, les durées
` + "``decision_ms``" + ` et ` + "``cube_decision_ms``" + ` : mesurées par l'Arbitre quand elles le sont,
sinon déduites des repères. Un repère antérieur au précédent est marqué par
l'incohérence ` + "``timecode_backwards``" + ` et laisse inconnues les durées qui en dépendent.
`

// GenerateAPIReferenceRST renders model as the Sphinx annex
// doc/source/api_reference.rst.
func GenerateAPIReferenceRST(model *Model) string {
	var b strings.Builder
	b.WriteString(rstIntro)
	b.WriteString(".. code-block:: text\n\n")

	families := map[string][]Route{}
	for _, r := range model.Routes {
		families[r.Family] = append(families[r.Family], r)
	}
	familyNames := make([]string, 0, len(families))
	for f := range families {
		familyNames = append(familyNames, f)
	}
	sort.Strings(familyNames)

	idempotentCount := 0
	for _, family := range familyNames {
		routes := families[family]
		sort.Slice(routes, func(i, j int) bool { return routes[i].Pattern < routes[j].Pattern })
		fmt.Fprintf(&b, "   %s\n", family)
		for _, r := range routes {
			shape := "JSON"
			if r.Kind == kindStream {
				shape = "NDJSON"
			} else if r.Kind == kindCustom {
				shape = "custom"
			}
			line := fmt.Sprintf("     %-4s %-40s %s", r.Method, r.Pattern, shape)
			if r.IdempotencyKeySupported {
				line += "  (Idempotency-Key)"
				idempotentCount++
			}
			if r.IfMatchRequired {
				line += "  (If-Match)"
			}
			if r.Conditional {
				line += "  (ETag)"
			}
			fmt.Fprintln(&b, strings.TrimRight(line, " "))
		}
	}
	// The event stream is no <family>.<method> call: the parser does not see it.
	b.WriteString("   events\n")
	fmt.Fprintf(&b, "     %-4s %-40s %s\n", "GET", "/v1/events", "SSE")
	b.WriteString("\n")
	fmt.Fprintf(&b, rstOutroTemplate, idempotentCount)
	return b.String()
}
