# Transcription de matchs — spécification fonctionnelle

Décisions de référence : [ADR-0044](../../docs/adr/0044-transcribing-a-match-is-not-playing-one.md)
(transcrire n'est pas jouer) et [ADR-0045](../../docs/adr/0045-a-transcription-is-a-draft-that-owns-its-match.md)
(le brouillon et son match). Vocabulaire : `CONTEXT.md`, section « Recording a match »
(Transcription, Transcript, Action, Cursor, Replay, Inconsistency). Ce document dit *ce que*
le logiciel fait ; [ux.md](ux.md) dit *par quels gestes* ; [integration.md](integration.md)
dit *ce que cela touche* ; [plan.md](plan.md) dit *dans quel ordre*.

## 1. Le document

Une Transcription est un document JSON, une ligne de la table `transcription`, écrit après
chaque Action. Il porte son propre `format_version` : une évolution de sa forme est une
version du document, jamais une migration de `DatabaseVersion`.

### 1.1 En-tête

| Champ | Saisi | Défaut | Note |
|---|---|---|---|
| `match_length` | oui, seul champ exigé à la création | longueur du dernier brouillon, sinon 7 | `0` = partie d'argent |
| `jacoby` | oui, si argent | vrai | drapeau de session (ADR-0028), posé sur chaque position |
| `beaver` | oui, si argent | faux | posé sur les positions, jamais une Action (ADR-0044) |
| `player1`, `player2` | plus tard | vides | joueur 1 en bas ; autocomplétés depuis les Players |
| `event`, `location`, `round`, `date` | plus tard | date = aujourd'hui | en-têtes `.mat` |
| `transcriber` | plus tard | `metadata.user` de la base | en-tête `.mat` `[Transcriber]` |
| `tournament_id` | plus tard | nul | rattachement à l'enregistrement |
| `match_id` | jamais | nul | posé au premier enregistrement, stable ensuite |
| `cursor` | dérivé du geste | fin du document | l'Action courante |
| `next_score` | jamais saisi directement | nul | une frontière de partie en attente en fin de document : le score que déclare l'Action ajoutée ensuite, qui ouvre une partie (et close, inachevée, celle en cours). Seuls la conversion d'un brouillon dont la dernière ouverture n'avait pas de coup, et un `.mat` dont la dernière partie est vide (match non terminé), en laissent un ; `set_score` en fin de document (`At` = nombre d'Actions) le corrige ou l'efface. Une insertion au milieu ne le consomme pas, une suppression ne l'efface pas |

**Format** : le document porte `format_version` = 3, sans Action d'ouverture. Un brouillon
de format 1 ou 2 est converti à la lecture (`transcript.Upgrade`) : chaque `opening` est
retirée ; le coup qui la suivait devient le premier coup de la partie, porte le score
annoncé de l'ouverture (ou de l'égalité qui la précédait) et prend les dés de l'ouverture
en ordre de joueur — joué par le perdant, il reste marqué « dés incohérents » ; joué avec
un autre jet, il garde ses dés et est marqué de même. Les relances sont ignorées. Une
ouverture qui coupait une partie en cours devient un score annoncé (le score dérivé) sur
le premier coup suivant ; une ouverture sans coup derrière laisse `next_score`. Le Cursor
garde son Action, ou la fin.

### 1.2 Les Actions

Une Action est l'acte d'un joueur. Champs communs : `side` (0 = joueur 1, 1 = joueur 2),
`kind`. Le `side` est **possédé** par l'Action (ADR-0045 §4) : proposé à la saisie, il ne
change ensuite que par le geste « changer de camp ».

| `kind` | Champs propres | Ce que la sauvegarde en fait |
|---|---|---|
| `checker` | `dice[2]`, `steps[]` (`from`, `to`, `hit`) ; `board_after` **seulement** si le coup est illégal | un Move `checker` avec sa notation et la Position d'avant le coup |
| `dance` | `dice[2]` | un Move `checker` « Cannot Move », sa Position |
| `unrecorded` | `dice[2]` | un Move `checker` de notation `???`, sa Position ; le coup a été joué, le fichier ne dit pas lequel |
| `double` | — | un Move `cube` `Double`, sa Position (décision du doubleur) |
| `take` | — | un Move `cube` `Take`, sa Position (décision du preneur, videau au niveau offert) |
| `pass` | — | un Move `cube` `Pass`, sa Position ; termine la partie |
| `resign` | `level` ∈ {1, 2, 3} | rien dans `move` ; `winner` et `points_won` de la partie |

Il n'y a pas d'Action d'ouverture. La première Action d'une partie est son premier coup,
joué par le gagnant du jet d'ouverture avec ce jet : ses `dice` sont **dé du J1, dé du J2**,
et le plus haut désigne le camp qui joue. Un premier coup en double est marqué « dés
incohérents » (aucun jet d'ouverture n'est un double), comme un premier coup joué par le
camp que l'ordre des dés ne désigne pas ; un `double` avant le premier coup est un « videau
impossible ». Les égalités (relances) ne se transcrivent pas. Toute Action peut porter
`score[2]` **facultatif**, le score annoncé de la partie qu'elle ouvre (ADR-0053) : une
Action qui en porte un ouvre toujours une partie, et close inachevée celle en cours.

### 1.3 Ce que le Replay dérive

Le Replay rejoue les Actions dans l'ordre à partir d'une Action donnée et attache à chacune,
sans les stocker dans le Match : la Position d'avant (plateau, videau, dés, score away,
Crawford, trait), le plateau d'après, le numéro de partie, le score de la partie, et ses
Incohérences. Il dérive aussi, par partie : `initial_score`, `winner`, `points_won`, la
mention Crawford ; et pour le match : la fin (un joueur atteint la longueur), le vainqueur.

**Plateau d'après une Action `checker` légale** : la position résultante du coup légal dont
les `steps` correspondent (comparaison par plateau résultant, jamais par notation —
`LegalMoves` déduplique par plateau). **Illégale** : `board_after` tel que saisi.

**Fin de partie** : détectée quand un camp a ses quinze pions sortis (points = 1, 2 ou 3
selon que l'adversaire a sorti au moins un pion, aucun, ou aucun avec un pion dans le jan
adverse ou à la barre, × valeur du videau), après un `pass` (valeur du videau avant le
double), après un `resign` (`level` × valeur du videau). Le score de la partie suivante
en découle ; le premier coup de la partie suivante est attendu.

**Score annoncé** (ADR-0053) : **sauf** quand la première Action d'une partie porte un
`score`. La partie est alors jouée à ce score — `initial_score`, scores away de ses
positions, mention Crawford, et tout ce qui suit en découle —, et s'il n'est pas celui que
donnent les parties précédentes, cette Action porte l'Incohérence « score annoncé
incohérent ». Un score annoncé en argent (autre que 0-0) ou négatif est marqué de même et
ignoré.

**Crawford** : la première partie où un joueur atteint `match_length − 1` est la partie
Crawford (positions à sentinelle away `1`) ; les parties suivantes sont post-Crawford
(sentinelle `0`). Dans la partie Crawford, un `double` est une Incohérence « action de
videau impossible ». En argent, pas de Crawford et le score est `[-1, -1]`.

**Fin de match** : dès qu'un score atteint `match_length`. Toute Action au-delà porte
l'Incohérence « au-delà de la fin ».

### 1.4 Les Incohérences

| Incohérence | Détection | Conservée |
|---|---|---|
| coup illégal | `board_after` n'est le résultat d'aucun coup de `LegalMoves(position d'avant)` ; ou `steps` ne correspondent à aucun coup légal après une correction en amont | oui, `board_after` fait foi |
| double trait | deux Actions consécutives de même `side` d'une même partie, hors `take`/`pass`/`resign` (après un `take`, le doubleur rejoue : ce n'est pas un double trait) | oui |
| videau impossible | `double` par un camp qui ne possède pas le videau (ni centré) ; `double` en partie Crawford ; `take`/`pass` sans `double` juste avant ; `double` avec un videau déjà au plafond `max_cube` s'il est défini | oui |
| au-delà de la fin | Action après que le match est gagné | oui |
| dés incohérents | `checker` dont les `steps` n'utilisent pas les dés de l'Action (cas produit par une correction de jet) | oui, requalifie le coup en illégal |
| coup non consigné | Action `unrecorded` : le jet est connu, le coup ne l'est pas (`???` dans un `.mat` de gnubg) | oui, le plateau d'avant est reconduit et tout ce qui suit est invérifiable |
| score annoncé incohérent | la première Action d'une partie annonce un score que les parties précédentes ne donnent pas (ADR-0053) ; ou un score qui ne peut servir : en argent, négatif | oui, la partie est jouée au score annoncé, le détail cite le score dérivé |

Le **coup non consigné** n'est pas une danse. Une cellule qui ne porte que ses dés dit que
le joueur n'a **pas pu** jouer ; une cellule `???` dit que gnubg n'a **pas consigné** ce
qu'il a joué. Le décodage du parseur est vide dans les deux cas, seule la marque les
distingue, et la rendre en danse écrirait dans le fichier une affirmation que personne n'a
faite. Le constat porte sur le *dossier*, jamais sur les joueurs.

Aucune Incohérence n'est refusée, aucune n'est supprimée par le logiciel, aucune n'est une
donnée du Match enregistré. Après un Replay, le Cursor saute à la première.

## 2. Les gestes et leur sémantique

Chaque geste est une fonction pure du paquet `transcript` : `(document, geste) → document
annoté`. Les touches sont dans [ux.md](ux.md) ; ici, l'effet.

| Geste | Précondition | Effet | Replay depuis |
|---|---|---|---|
| créer un brouillon | longueur donnée | document vide, le premier coup d'une partie attendu | — |
| saisir un dé | `checker` attendue | remplit le premier dé libre ; si les deux sont pleins, recommence ; au premier coup d'une partie, le second dé désigne le camp (le plus haut joue), sauf si le jet ressaisi est celui déjà écrit | — |
| effacer les dés | dés saisis, Action non validée | vide les deux dés | — |
| sélectionner un candidat | jet saisi, coups légaux non vides | `steps` = ceux du candidat | — |
| valider | candidat sélectionné | l'Action `checker` est **créée** au Cursor (ou remplace l'Action corrigée) ; le Cursor avance | l'Action créée |
| danser | jet saisi, coups légaux vides | Action `dance` créée aussitôt | l'Action |
| doubler | trait au camp courant, videau centré ou possédé | valide l'Action en attente s'il y en a une, puis crée `double` | l'Action |
| prendre / passer | Action précédente `double` | valide l'attente, crée `take` / `pass` | l'Action |
| résigner (camp, niveau) | partie en cours | valide l'attente, crée `resign` ; partie terminée | l'Action |
| reculer / avancer le Cursor | — | le Cursor change ; les dés et le candidat de l'Action visée sont chargés | — |
| corriger en place | Cursor sur une Action | la nouvelle saisie remplace l'Action ; le Cursor **revient** à sa place antérieure après validation | l'Action |
| insérer avant / après | Cursor sur une Action | une Action nouvelle est attendue à cet endroit ; `side` proposé = celui qui rend la suite cohérente | l'Action insérée |
| supprimer | Cursor sur une Action, ou une saisie en cours | l'Action (ou la saisie non écrite) disparaît ; les suivantes gardent leur `side` ; le Cursor recule sur la précédente, chargée pour correction (ADR-0050) | la suivante (Cursor tenu) |
| changer de camp | Cursor sur une Action | `side` inversé | l'Action |
| changer la longueur | — | `match_length` ; en argent ↔ match, drapeaux de session réévalués | la première Action |
| inverser les joueurs | — | noms échangés ; tous les `side` inversés ; plateau retourné ; scores annoncés inversés | la première Action |
| annoncer le score d'une partie | `At` = la première Action d'une partie, match (pas d'argent), score ≥ 0 ; un score ≥ `match_length` est accepté et marqué | `score` posé sur la première Action de la partie (geste `set_score`) ; sans score, il est effacé et la partie revient au score dérivé ; le Cursor ne bouge pas (ADR-0053) | cette Action (Cursor tenu) |
| annuler / rétablir | pile non vide | document précédent / suivant (pile en mémoire) | tout |
| enregistrer | au moins une Action | Match créé ou remplacé (§4) | — |
| exporter `.mat` | — | fichier rendu depuis le document ; avertissement si coup illégal | — |
| fermer le brouillon | — | ligne supprimée ; si jamais enregistré, confirmation ; le Match éventuel reste et devient définitif | — |

**Corriger un jet** : si les `steps` de l'Action restent un coup légal du nouveau jet, ils
sont gardés ; sinon le premier candidat du nouveau jet est présélectionné et l'Action est
marquée « à revoir » jusqu'à validation.

**Saisie manuelle d'un coup** (ADR-0052) : une notation (`13/7 8/7*`, `bar/22`, `6/off`)
tapée dans la cellule du Transcript ouverte par un double-clic — les dés sont ceux de la
cellule —, ou, le jet saisi, un glissé au plateau qu'aucun coup légal n'offre, qui rend le
coup libre jusqu'à Entrée. Si le plateau résultant est celui d'un coup légal, l'Action est
un coup ordinaire ; sinon un coup illégal avec `board_after`.

## 3. Écriture et reprise

Le document est écrit dans sa ligne **après chaque geste qui change une Action** (création,
validation, correction, insertion, suppression, changement de camp, de longueur, de
métadonnées), dans une transaction propre. Les gestes qui ne changent que le Cursor ou la
sélection n'écrivent pas. Après un plantage, la réouverture de la base retrouve les
brouillons ; ouvrir l'un d'eux place le Cursor à la fin et rejoue tout. La pile d'annulation
est perdue.

## 4. Enregistrement, remplacement, analyse

1. Le document est rejoué en entier ; s'il contient des Incohérences, l'utilisateur en est
   averti et peut enregistrer quand même (ADR-0044 : rien n'est refusé).
2. Le paquet produit le match, ses parties et ses coups (types du domaine ; l'appelant en
   fait un `ingest.MatchGraph`) : un `Game` par partie avec `InitialScore` (le score joué,
   l'annoncé s'il y en a un), `Winner`, `PointsWon` ; un `Move` par Action
   `checker`/`dance`/`double`/`take`/`pass`, numéroté, avec sa `Position` (sentinelle Crawford écrite, drapeaux de session posés) ; aucune
   analyse, aucun commentaire. `MatchHash` et `CanonicalHash` sont calculés sur ce graphe
   et changent à chaque enregistrement — ils servent la déduplication des *imports*, pas
   l'identité du Match transcrit, qui est son `id`.
3. `ingest.WriteMatch` en **mode remplacement** : dans une transaction, les `game`/`move` du
   `match_id` existant sont supprimés (cascade), la ligne `match` est mise à jour en place,
   les nouveaux `game`/`move`/`position` sont écrits ; les positions orphelines sont purgées
   par `positionIsHeldSQL` ; **rien ne passe par le Trash** (ADR-0045 §3). Premier
   enregistrement : création ordinaire, `match_id` posé sur le document.
4. Le lot gammonNet **ciblé** démarre : `AnalyzeMissingWithGammonNet` restreint aux positions
   du match sans analyse ; profondeur canonique 2-ply, `k = 12` ; progression dans la barre
   d'état ; annulable ; `shutdown` l'annule avant de fermer la base.
5. Le brouillon reste ouvert. Un ré-enregistrement refait 1–4 ; seules les positions nouvelles
   sont analysées (ADR-0013).

**Reprise de l'analyse** : à l'ouverture d'une base, si le Match du dernier brouillon
enregistré a des positions sans analyse, la barre d'état propose de terminer ; le lot ciblé
repart. Aucun état n'est stocké pour cela.

## 5. Export et import `.mat`

- **Rendu** : `RenderMAT` sur le graphe du document, sans passer par la base. En-têtes :
  `[Site]`, `[Event]`, `[Round]`, `[Player 1]`, `[Player 2]`, `[EventDate]`, `[Transcriber]`
  quand ils sont non vides ; `N point match` ou `0 point match`. Une partie terminée par
  résignation ou passe se termine par « Wins N points » sans dernier coup ; une partie
  inachevée n'a pas de ligne « Wins ».
- **Ce que le `.mat` ne porte pas** : Crawford (déduit par le lecteur), Jacoby/beaver,
  la résignation comme telle, les coups illégaux comme tels, l'analyse, les commentaires.
- **Coup illégal** : exporté tel que joué ; le dialogue avertit que gnubg et XG signaleront
  « Invalid move » et divergeront ensuite ; l'export n'est jamais refusé.
- **Score annoncé** : la ligne de score d'une partie est son `InitialScore`, donc le score
  annoncé quand il y en a un (ADR-0053). À la lecture, une ligne de score que les parties
  précédentes ne donnent pas redevient un score annoncé sur la première Action de la
  partie ; le premier coup prend l'ordre de l'ouverture (dé J1, dé J2), le `.mat` écrivant
  le gros dé d'abord.
- **Coup non consigné** : lu `???`, rendu `???`. L'aller-retour d'un `.mat` de gnubg qui en
  porte redonne le même nombre de cellules `???`, jamais des danses.
- **Aller-retour** : `gnubgparser.ParseMAT(ingest.RenderMAT(transcript.MatchParts(doc)))`
  redonne le même graphe (test de
  table sur des `.mat` réels du dépôt et sur des documents synthétiques couvrant chaque
  `kind`) ; un `.mat` importé peut être **rejoué** par le paquet pour en signaler les
  Incohérences (`blunderdb transcribe --check`, lot 3).

## 6. Les flux

État avant → gestes → état après. Les touches sont dans ux.md.

1. **Nouveau match** : aucun brouillon ouvert → « nouvelle transcription », longueur → document
   vide, premier coup attendu, plateau initial, score `[N, N]`.
2. **Premier coup** : premier coup attendu → dé J1, dé J2 → le plus haut a le trait, ses
   candidats pour ce jet sont listés → choix → validation. Les égalités ne se saisissent pas.
3. **Tour de pions** : `checker` attendue → deux dés → liste, premier présélectionné → choix
   → validation → Action créée, plateau avancé, trait à l'autre camp.
4. **Danse** : deux dés → aucun coup → `dance` créée, trait à l'autre camp.
5. **Double / prise** : trait au camp A, videau disponible → `double` → `take` par B → videau
   à B au niveau doublé, trait à A, dés attendus.
6. **Double / passe** : `double` → `pass` → partie gagnée par A de la valeur d'avant le
   double ; score avancé ; premier coup de la partie suivante attendu.
7. **Redouble** : videau possédé par A → `double` par A → §5/§6.
8. **Résignation** : partie en cours → camp, niveau → partie gagnée par l'autre ; score ;
   premier coup de la partie suivante attendu.
9. **Fin de partie par sortie** : `checker` qui sort le quinzième pion → points calculés ;
   score ; premier coup de la partie suivante attendu.
10. **Partie suivante** : score affiché, Crawford dérivé et affiché si c'est la partie.
11. **Fin de match** : score atteint `match_length` → le document est « terminé » ; toute
    Action de plus est marquée.
12. **Correction à mi-match** : Cursor reculé de k Actions → dés ou candidat changés →
    validation → Replay depuis là ; Incohérences marquées ; Cursor revenu.
13. **Insertion** : Cursor sur l'Action n → insérer avant → Action attendue de camp proposé →
    validation → Replay ; un double trait apparaît si le camp est le même que le voisin.
14. **Suppression** : Cursor sur n → supprimer → Replay depuis n ; double trait probable à n,
    marqué ; le Cursor reste sur n−1 (ADR-0050).
15. **Changement de camp** : Cursor sur n → changer → Replay ; les coups suivants peuvent
    devenir illégaux (pions de l'autre camp) : marqués, jamais supprimés.
16. **Coup illégal** : jet saisi, coup glissé au plateau hors des règles puis Entrée, ou tapé
    dans sa cellule → pas de candidat correspondant → Action `checker` avec `board_after`,
    marquée.
17. **Changement de longueur** : `match_length` modifié → Replay complet ; score away, Crawford,
    référentiel recalculés ; Actions au-delà de la fin marquées.
18. **Enregistrement** : « Enregistrer » → §4 ; le Match apparaît dans le panneau Match ; le
    lot d'analyse tourne.
19. **Ré-enregistrement** : correction puis « Enregistrer » → Match remplacé, même `id` ; seules
    les positions nouvelles analysées.
20. **Export** : « Exporter `.mat` » → fichier `Player1_Player2_YYYY-MM-DD_Np.mat` ;
    avertissement si coup illégal.
21. **Plantage et reprise** : fermeture brutale → réouverture → le brouillon est là, à sa
    dernière Action écrite ; la pile d'annulation est vide.
22. **Abandon** : « Fermer le brouillon » → confirmation si jamais enregistré → ligne
    supprimée ; un Match déjà enregistré reste, définitif.

## 7. Hors périmètre

Double automatique, beaver/raccoon comme Action, match sans règle de Crawford (backlog),
rouvrir un Match enregistré sans brouillon comme nouveau brouillon (backlog : « rejouer un
`.mat` importé » y répond en partie), édition des coups d'un Match hors brouillon, le mode
`serve` et le front web (ADR-0039).
