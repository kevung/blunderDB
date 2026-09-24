# ADR-0053 — Le score d'une partie peut être annoncé, et la partie est jouée à ce score

- **Statut** : accepté — décidé le 2026-09-24, après usage.
- **Amende** : fonctionnel.md §1.2 (un champ facultatif sur l'`opening`), §1.3 (le score
  d'une partie est dérivé *sauf annonce*), §1.4 (une septième Incohérence), §2 (un geste)
  et §5 (la ligne de score du `.mat`).
- **Applique** : ADR-0044 (transcrire n'est pas jouer : rien n'est refusé, tout est marqué)
  et ADR-0045 (le brouillon est un document JSON versionné ; les Incohérences sont
  dérivées, jamais stockées).

## Le problème

Le score de chaque partie était entièrement **dérivé** : le Replay l'obtient en ajoutant le
résultat des parties précédentes, et rien dans le document ne pouvait le contredire. Or une
erreur de score se commet à la table — un point oublié sur la feuille, une partie comptée
double, une partie jamais notée — et la suite du match se joue alors à ce score faux : le
videau, le Crawford, les décisions de chacun en dépendent. Une transcription qui recalcule
le « bon » score décrit un match que personne n'a joué, et les positions enregistrées
portent des scores away que les joueurs n'avaient pas sous les yeux.

## La décision

1. **Le score annoncé vit sur l'ouverture.** L'Action `opening` qui commence une partie
   porte un champ facultatif `score` (points du joueur 1, du joueur 2). Une égalité
   d'ouverture reste dans la même partie : c'est la première ouverture, celle qui ouvre la
   partie, qui porte le score. Le champ est dans le document JSON du brouillon ; rien ne
   change dans le schéma SQLite.
2. **La partie est jouée au score annoncé.** Au début d'une partie qui en porte un, le
   score de jeu du Replay devient le score annoncé, **avant** que la partie s'ouvre : la
   mention Crawford est décidée sur lui, et tout ce qui en dérive le suit — l'`initial_score`
   de la partie, les scores away de ses positions, la fin du match, le vainqueur, et les
   parties suivantes.
3. **Ce qui diffère est marqué, jamais corrigé.** Quand le score annoncé n'est pas celui
   que donnent les parties précédentes, l'ouverture porte une nouvelle Incohérence,
   `score_mismatch` (« score annoncé incohérent »), dont le détail cite le score dérivé.
   Elle est dérivée à chaque Replay et jamais stockée. Un score annoncé qui ne peut servir —
   sur une relance, en session d'argent, négatif — est marqué de même et ignoré.
   `GameInfo` expose `declared` et `derived_score`, pour que la vue montre l'écart sans le
   recalculer.
4. **Un geste, `set_score`.** Il désigne la partie par l'index de son ouverture (`At`) —
   c'est `GameInfo.first`, que la vue a déjà —, et porte le score (`Score`), ou l'efface
   quand il est absent. Il passe par la pile d'annulation de l'`Editor` comme tout geste,
   et tient le Cursor : annoncer un score n'édite pas la décision en cours. Il ne refuse
   que ce qui n'a pas de sens : un index qui n'est pas l'ouverture d'une partie, un score
   négatif, un score en argent. Un score qui atteint la longueur du match est accepté ; ce
   qui le suit est alors « au-delà de la fin », et le Replay le dit.
5. **Les autres gestes le gardent.** Corriger le jet d'une ouverture garde son score ;
   inverser les joueurs inverse le score ; supprimer l'égalité qui le portait le passe à
   la relance, qui ouvre désormais la partie.
6. **Enregistrer et exporter portent le score joué.** Le `Game` enregistré et la ligne de
   score du `.mat` sont l'`initial_score` de la partie, donc le score annoncé. `ingest` ne
   contrôle pas l'enchaînement des scores et ne répare rien. À la lecture d'un `.mat`
   (`FromMAT`), une ligne de score que les parties précédentes ne donnent pas redevient un
   score annoncé : l'aller-retour garde l'erreur, et `blunderdb transcribe --check` la
   signale.
7. **Dans le Transcript, un double-clic sur le score de l'en-tête d'une partie** l'ouvre
   en champ, pré-rempli du score affiché. « 3-2 », « 3–2 » et « 3 2 » sont lus ; Entrée
   valide, un champ vidé puis validé efface le score annoncé, Échap (par `escapeService`)
   et la perte du focus ferment sans rien écrire — la mécanique du coup tapé dans sa
   cellule (ADR-0052). Un score qui diffère du score dérivé s'affiche marqué, le score
   dérivé en infobulle. Pas de champ en session d'argent. Un clic simple sur ce score ne
   plie plus la partie : le premier clic du double-clic la replierait.

## Le format du document

`FormatVersion` passe à **2**. Un document de version 1 est un document de version 2 sans
score annoncé : il se relit à l'identique, et le champ absent (`omitempty`) ne s'écrit pas.
La version est relevée sur un ancien brouillon dès qu'un score y est écrit, et un
brouillon neuf naît en version 2 : un binaire qui ne lit que la version 1 refuse alors le
brouillon au lieu de perdre le score en silence au prochain enregistrement — ce pour quoi
la version existe (`decodeTranscription`).

## Les options écartées

- **Un score par partie dans l'en-tête du document.** Il faudrait le renuméroter à chaque
  insertion ou suppression d'une partie ; porté par l'ouverture, il la suit.
- **Désigner la partie par son numéro.** `Apply` devrait rejouer le document pour trouver
  l'ouverture ; l'index de l'ouverture est déjà dans la vue et se vérifie localement.
- **Refuser un score au-delà de la longueur.** C'est ce qui a été noté ; ADR-0044.
- **Recalculer le « bon » score à l'enregistrement.** Le Match décrirait un autre match.

## Les conséquences

- Le cache incrémental reste exact : l'Action porte son score, `sameAction` le compare, et
  la dérivation d'une Action ne dépend toujours que de l'état laissé par la précédente.
- Aucun moteur d'analyse n'est tenu d'accepter ce score : gnubg et XG liront le `.mat` tel
  qu'il est écrit. La transcription est fidèle à la captation, pas au règlement.
