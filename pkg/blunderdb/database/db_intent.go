package database

import "github.com/kevung/blunderdb/pkg/blunderdb/searchquery"

// La grammaire d'intentions (#283, fiche I.27), côté base.
//
// Rien à lire ni à écrire : la traduction est pure. Elle passe quand même par
// `Database` pour la même raison que le reste — l'interface, la ligne de
// commande et le démon doivent traduire la même phrase de la même façon, et
// une seconde table de vocabulaire en JavaScript aurait été un second sens
// pour « blunder ».

// TranslateIntent turns a phrase into the search tokens it means, plus what
// was understood and what was not.
func (d *Database) TranslateIntent(text string) searchquery.Intent {
	return searchquery.TranslateIntent(text)
}
