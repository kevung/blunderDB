package domain

import (
	"reflect"
	"testing"
)

func TestExtractTags(t *testing.T) {
	for _, tc := range []struct {
		text string
		want []string
		why  string
	}{
		{"#blitz #prime", []string{"#blitz", "#prime"}, "deux étiquettes"},
		{"#Blitz #blitz", []string{"#blitz"}, "la casse ne fait pas deux étiquettes"},
		{"j'ai joué #blitz.", []string{"#blitz"}, "la ponctuation de fin ne fait pas partie de l'étiquette"},
		{"#back-game et #2-away", []string{"#2-away", "#back-game"}, "tiret et chiffre appartiennent à l'étiquette"},
		{"#préparation", []string{"#préparation"}, "les accents appartiennent à l'étiquette"},
		{"rien à étiqueter", nil, "un commentaire sans dièse n'a pas d'étiquette"},
		{"##", nil, "un dièse seul n'est pas une étiquette"},
		{"#prime #blitz", []string{"#blitz", "#prime"}, "l'ordre rendu est alphabétique, pas celui de la saisie"},
	} {
		if got := ExtractTags(tc.text); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s : ExtractTags(%q) = %v, attendu %v", tc.why, tc.text, got, tc.want)
		}
	}
}
