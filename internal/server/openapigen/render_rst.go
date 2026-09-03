package openapigen

import (
	"fmt"
	"sort"
	"strings"
)

// rstIntro/rstOutro are the only translatable prose in the generated
// annex — see the package doc comment on why the family→method table itself
// is a literal code-block instead of an RST table: Sphinx's gettext
// extraction skips a code-block's content (unlike a docutils table's cell
// text), so regenerating this file when a route is added or removed never
// touches the eight .po catalogues; only a change to this prose does.
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
lectures sont sans effet de bord, et ` + "``positions.save``" + ` (comme le reste de
` + "``positions.*``" + `) est naturellement idempotente grâce au hachage Zobrist du
contenu — enregistrer deux fois la même position renvoie la même ligne, jamais
un doublon. %d méthodes n'ont pas cette propriété (deux appels sont deux effets
distincts) et acceptent un en-tête ` + "``Idempotency-Key``" + ` optionnel : un appel
rejoué avec la même clé renvoie le résultat de la première tentative au lieu de
répéter son effet — voir la marque « (Idempotency-Key) » dans le tableau
ci-dessus. Aucune autre méthode n'a besoin ou n'accepte cet en-tête.
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
			fmt.Fprintln(&b, strings.TrimRight(line, " "))
		}
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, rstOutroTemplate, idempotentCount)
	return b.String()
}
