# ADR-0051 — La dernière Action se tape comme le bout du document

Statut : acceptée.
Voir aussi : ADR-0048 décision 1, ADR-0049, ADR-0050 règle 4.

## Contexte

Sur la dernière Action, le chiffre recommençait le jet sans jamais valider (retaper son jet
puis le suivant réécrivait deux fois la même Action), et la validation rendait le curseur à
`Return`, parfois plus haut. Après la dernière Action il n'y a rien à protéger.

## Décision

1. **Sur la dernière Action, une fois son jet retapé, le chiffre suivant valide puis ouvre le
   jet d'après**, comme en bout de document. Le premier chiffre sur la cellule telle que chargée
   la corrige toujours sur place. La machine à touches le sait par son état (`retyped`) et le
   contexte passé par le panneau (`last`) ; elle ne connaît pas le document.
2. **Valider la dernière Action (Entrée, le chiffre ci-dessus, un geste de videau) envoie le
   curseur au bout du document** quoi que dise `Return`, et l'y tient (`HoldCursor`).
3. Ailleurs, ADR-0048 décision 1 et ADR-0049 s'appliquent inchangées.

## Garde

`frontend/src/__tests__/transcriptionKeys.correction.test.js`,
`pkg/blunderdb/transcript/inplace_test.go`.
