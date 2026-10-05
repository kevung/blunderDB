# ADR-0053 — Le score d'une partie peut être annoncé, et la partie est jouée à ce score

Statut : acceptée.
Voir aussi : ADR-0044, ADR-0045, ADR-0052 règle 5.

## Contexte

Le score de chaque partie était entièrement dérivé des parties précédentes. Or une erreur de
score se commet à la table, et la suite du match se joue à ce score faux : videau, Crawford
et décisions en dépendent. Recalculer le « bon » score décrirait un match que personne n'a
joué.

## Décision

1. **Le score annoncé vit sur la première Action de la partie** : champ facultatif `score`
   ([p1, p2]) ; le Replay expose `opens_game` sur cette Action. Il est dans le document JSON ;
   le schéma SQLite ne change pas. Une Action qui porte un score ouvre toujours une partie :
   si une partie court encore, elle s'arrête là, inachevée. C'est ainsi qu'un `.mat` dont une
   partie s'interrompt se relit, et une partie d'argent qui s'interrompt porte un 0-0 qui
   n'annonce rien.
2. **La partie est jouée au score annoncé** : le Replay l'adopte avant d'ouvrir la partie ;
   Crawford, `initial_score`, scores away, fin du match, vainqueur et parties suivantes en
   dérivent.
3. **Ce qui diffère est marqué, jamais corrigé** : Incohérence `score_mismatch` sur la
   première Action, détail = score dérivé, dérivée à chaque Replay. Un score inutilisable
   (argent, négatif) est marqué et ignoré. `GameInfo` expose `declared` et `derived_score`.
4. **Geste `set_score`** : `At` = index de la première Action (`GameInfo.first`), `Score` ou
   absent pour effacer ; passe par la pile d'annulation et tient le Cursor. Refuse seulement
   un index qui n'ouvre pas une partie, un score négatif ou en argent. Un score atteignant la longueur
   est accepté (la suite est « au-delà de la fin »).
5. **Les autres gestes le gardent** : corriger l'Action qui le porte le garde ; inverser les
   joueurs l'inverse ; la supprimer le passe à l'Action suivante de la même partie ; insérer
   devant elle le passe à l'Action insérée, qui ouvre alors la partie.
6. **Enregistrer et exporter portent le score joué** (`initial_score`, ligne de score du
   `.mat`). `ingest` ne contrôle pas l'enchaînement. `FromMAT` relit une ligne de score non
   dérivable comme un score annoncé ; `blunderdb transcribe --check` la signale.
7. **Double-clic sur le score d'en-tête de partie** → champ ; « 3-2 », « 3–2 », « 3 2 » lus ;
   Entrée valide, vide = efface, Échap ou perte de focus ferment sans écrire. Un score annoncé
   différent s'affiche marqué, le dérivé en infobulle. Pas de champ en argent. Un clic simple
   ne plie pas la partie.

**Format** : `FormatVersion` = 3, sans Action d'ouverture (ADR-0049 règle 6). Tout brouillon
écrit l'est en v3, qu'un binaire antérieur refuse au lieu de le mal lire. Un document v1 ou
v2 est converti à la lecture (`transcript.Upgrade`, appelé par `decodeTranscription`) :
l'ouverture disparaît et le coup qui la suivait devient la première Action, portant le score
annoncé sur elle ; les relances sont ignorées ; une ouverture qui coupait une partie en cours
devient un score annoncé (le score dérivé) sur la première Action suivante. Une ouverture
sans coup derrière — brouillon fermé juste après elle — laisse une frontière en attente en
fin de document (`next_score`), que reprend l'Action ajoutée ensuite ; un `.mat` dont la
dernière partie est vide en laisse une aussi, sauf après la fin du match. `set_score` sur
la case de fin (`At` = nombre d'Actions) la corrige ou l'efface.

## Conséquences

- Le cache incrémental reste exact : `sameAction` compare le score.
- gnubg et XG liront le `.mat` tel qu'écrit : fidèle à la captation, pas au règlement.
- Écartés : un score par partie dans l'en-tête du document (à renuméroter à chaque
  insertion) ; désigner la partie par son numéro (`Apply` devrait rejouer) ; refuser un score
  au-delà de la longueur (ADR-0044) ; recalculer à l'enregistrement.

## Garde

`pkg/blunderdb/transcript/declared_score_test.go`,
`frontend/src/__tests__/TranscriptionPanel.declaredScore.test.js`.
