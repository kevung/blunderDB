# ADR-0053 — Le score d'une partie peut être annoncé, et la partie est jouée à ce score

Statut : acceptée.
Voir aussi : ADR-0044, ADR-0045, ADR-0052 règle 5.

## Contexte

Le score de chaque partie était entièrement dérivé des parties précédentes. Or une erreur de
score se commet à la table, et la suite du match se joue à ce score faux : videau, Crawford
et décisions en dépendent. Recalculer le « bon » score décrirait un match que personne n'a
joué.

## Décision

1. **Le score annoncé vit sur l'ouverture** : champ facultatif `score` ([p1, p2]) de la
   première `opening` de la partie (une égalité reste dans la partie). Il est dans le document
   JSON ; le schéma SQLite ne change pas.
2. **La partie est jouée au score annoncé** : le Replay l'adopte avant d'ouvrir la partie ;
   Crawford, `initial_score`, scores away, fin du match, vainqueur et parties suivantes en
   dérivent.
3. **Ce qui diffère est marqué, jamais corrigé** : Incohérence `score_mismatch` sur
   l'ouverture, détail = score dérivé, dérivée à chaque Replay. Un score inutilisable (relance,
   argent, négatif) est marqué et ignoré. `GameInfo` expose `declared` et `derived_score`.
4. **Geste `set_score`** : `At` = index de l'ouverture (`GameInfo.first`), `Score` ou absent
   pour effacer ; passe par la pile d'annulation et tient le Cursor. Refuse seulement un index
   qui n'est pas une ouverture, un score négatif ou en argent. Un score atteignant la longueur
   est accepté (la suite est « au-delà de la fin »).
5. **Les autres gestes le gardent** : corriger le jet d'ouverture le garde ; inverser les
   joueurs l'inverse ; supprimer l'égalité qui le portait le passe à la relance.
6. **Enregistrer et exporter portent le score joué** (`initial_score`, ligne de score du
   `.mat`). `ingest` ne contrôle pas l'enchaînement. `FromMAT` relit une ligne de score non
   dérivable comme un score annoncé ; `blunderdb transcribe --check` la signale.
7. **Double-clic sur le score d'en-tête de partie** → champ ; « 3-2 », « 3–2 », « 3 2 » lus ;
   Entrée valide, vide = efface, Échap ou perte de focus ferment sans écrire. Un score annoncé
   différent s'affiche marqué, le dérivé en infobulle. Pas de champ en argent. Un clic simple
   ne plie plus la partie.

**Format** : `FormatVersion` = 2. Un document v1 se relit comme v2 sans score (`omitempty`) ;
un brouillon neuf naît en v2 et un ancien passe en v2 dès qu'un score y est écrit, pour qu'un
binaire v1 le refuse au lieu de perdre le score (`decodeTranscription`).

## Conséquences

- Le cache incrémental reste exact : `sameAction` compare le score.
- gnubg et XG liront le `.mat` tel qu'écrit : fidèle à la captation, pas au règlement.
- Écartés : un score par partie dans l'en-tête du document (à renuméroter à chaque
  insertion) ; désigner la partie par son numéro (`Apply` devrait rejouer) ; refuser un score
  au-delà de la longueur (ADR-0044) ; recalculer à l'enregistrement.

## Garde

`pkg/blunderdb/transcript/declared_score_test.go`,
`frontend/src/__tests__/TranscriptionPanel.declaredScore.test.js`.
