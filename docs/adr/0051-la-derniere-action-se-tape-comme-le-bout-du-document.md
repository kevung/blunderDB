# ADR-0051 — La dernière Action se tape comme le bout du document

- **Statut** : accepté — décidé le 2026-09-24, après usage.
- **Amende** : ADR-0048 décision 1 (la touche chiffrée recommence le jet là où le curseur
  est) et ADR-0049 (une correction rend le curseur à sa place antérieure), pour la seule
  dernière Action du document.
- **Applique** : ADR-0050 (`HoldCursor`).

## Le problème

Revenir sur la dernière Action d'une transcription — pour la relire, la corriger, ou
simplement parce que le curseur y a été posé — laissait le panneau en mode
« correction » : un chiffre y recommençait le jet sur place sans jamais valider, si bien
que retaper son jet puis le jet suivant réécrivait deux fois la même Action ; et une
validation par Entrée rendait le curseur à `Return`, qui peut désigner une cellule plus
haut (le saut vers une Incohérence en laisse une), ou le ramenait sur une marque que
l'Action porte. Or après la dernière Action il n'y a rien à protéger : l'utilisateur est
revenu là où une transcription s'écrit, et attend d'y continuer comme la première fois.

## La décision

1. **Sur la dernière Action, une fois son jet RETAPÉ, le chiffre suivant valide puis
   ouvre le jet d'après**, comme en bout de document. Le premier chiffre tapé sur la
   cellule telle qu'elle a été chargée la corrige toujours sur place — sans quoi son jet
   ne se corrigerait plus. La machine à touches le sait par un drapeau de son état
   (`retyped`) et par le contexte que le panneau lui passe (`last`) ; elle ne connaît
   toujours pas le document.
2. **Valider la dernière Action — Entrée, le chiffre ci-dessus, un geste de videau —
   envoie le curseur au bout du document**, quoi que dise `Return`, et l'y tient
   (`HoldCursor`) : le Replay ne le ramène pas sur une marque de l'Action validée. C'est
   la règle de l'ajout en bout de document, appliquée à la cellule qui le précède.
3. Ailleurs dans le document, rien ne change : le chiffre recommence le jet sur place et
   la validation rend le curseur à sa place (ADR-0048, ADR-0049).
