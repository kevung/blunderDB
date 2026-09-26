# ADR-0048 — Le panneau de transcription est ordonné par la fréquence, non par le voisinage

Statut : acceptée.
Voir aussi : ADR-0045 (le brouillon et ses gestes), ADR-0049 à ADR-0054 (corrections en place,
curseur, coup au plateau), ADR-0018 règle 4, ADR-0021, ADR-0034.

## Contexte

Le panneau, correct geste par geste, était impraticable de bout en bout : dans le dock bas par
défaut (250 px), 354 px de contrôles précédaient la première ligne de candidats pour 168 px de
boîte. Des blocs posés chacun « sous son voisin », une chaîne de hauteur sans `min-height: 0`
(le panneau entier défilait et sautait à chaque Action), l'état du brouillon réimplémenté en
badges, et une touche chiffrée à deux sens séparés par un état invisible. Cette ADR ne touche
que la surface : rien de ce qui est enregistré ne change.

## Décision

Trois règles engendrent les treize décisions :

- **R1 — Ce qui sert à chaque tour est visible sans défiler ; ce qui sert une fois est à un
  geste.** Aucun bloc n'est placé par rapport à son voisin.
- **R2 — Un message habite là où est ce dont il parle.** Un seul domicile par message.
- **R3 — Tout geste a un chemin clavier ET un chemin souris.** Le clavier reste la voie par
  défaut ; R3 exige que chaque geste soit atteignable à la souris, pas au même coût.

**Contrat** : dans tout régime de dock utilisable, les deux cases du jet et les **cinq**
premières lignes de candidats (≈ 88 % des tours) sont visibles sans défiler, et **rien ne
s'intercale verticalement entre les cases du jet et la première ligne de candidats**.

1. **La touche chiffrée commence un jet là où le Cursor est.** En bout de document, le chiffre
   valide le candidat sélectionné et ouvre le jet suivant ; sur une Action existante
   (`entry.replacing`), il en recommence le jet sur place. `Entrée` valide aussi (dernier coup
   d'une partie) ; `Retour arrière` efface le jet en cours. Le discriminant est dessiné (la
   cellule encadrée), non un historique invisible. Meilleur coup joué : 2 K. La dernière
   Action du document : voir ADR-0051.
2. **L'état du brouillon quitte le panneau** : longueur, score, Crawford, trait, videau vers
   `MatchInfoBar` ; l'Action attendue vers la barre d'état. L'état d'enregistrement reste collé
   à son bouton (décision 12).
3. **Pas de barre de correction.** Les gestes sur l'Action au Cursor passent par le menu
   contextuel d'une cellule du Transcript, qui nomme sa cible, et par le clavier.
4. **La liste des candidats a deux projections nommées**, décidées dans `analysisRows.js`
   seulement : `judge` (neuf colonnes + provenance — Eval, Analyse) et `identify` (`move`,
   `equity`, `error` — Transcription). Un nouvel appelant justifie une nouvelle projection.
5. **La chaîne de hauteur est bornée** (`flex: 1; min-height: 0` sur `.draft` et
   `.draft-body`) : seuls la liste et le Transcript défilent. Boîte large : trois colonnes
   `palette | candidats | Transcript` ; boîte étroite : palette sous la liste. L'onglet exige
   un dock de **320 px** minimum ; en dessous, c'est la palette qui est rognée, jamais les cases
   du jet ni la liste.
6. **Le texte `.mat` s'ouvre en modale** depuis la barre du brouillon (ASCII aligné, ~409 px
   pour une colonne de 320). `TranscriptView` reste présentationnel.
7. **La palette ne contient que ce qui sert** : les cases du jet, la rangée de videau
   (Doubler, Prendre, Passer, Abandonner) sur sa propre ligne, le triangle des 21 jets. Le coup
   illégal se joue au plateau et se tape dans sa cellule (ADR-0052 règles 4 à 6).
8. **Chaque message a un domicile (R2)** et le panneau n'en porte pas en prose : ce que le
   document attend → barre d'état ; comment faire → aide en ligne et visite guidée ; où en est
   un geste → sur son objet (triangle, plateau, liste réduite par les pas joués) ; alerte
   d'Incohérence → bandeau en tête de la colonne Transcript.
9. **Un geste sans effet répond, une fois, dans la barre d'état** (~1,5 s, puis la phrase de
   l'Action attendue revient) : `Ctrl+Z` sur pile vide (toujours vide sur un brouillon
   réouvert, ADR-0045 règle 1), `Retour arrière` sans dé, `s` sans Action, `Suppr` sur un
   brouillon vide. Exception : `t` et `p` sans rien à quoi répondre (ADR-0049 règle 2) filent
   au répartiteur global sans réponse — `p` y est le compte de pips.
10. **La liste des brouillons est au clavier** : `j`/`k`, `Entrée` ouvre, première ligne
    (la plus récemment modifiée) présélectionnée, `n` crée. Surligner n'ouvre pas. Reprendre
    coûte `Ctrl+Maj+T` `Entrée`. L'entrée dans l'onglet n'ouvre aucun brouillon d'elle-même.
11. **Le chemin souris est complet** : la molette fait un pas dans la liste des candidats
    au-dessus de la liste et du plateau (`TRANSCRIBE` exclu de `handleWheel`) ; double-clic
    sur un candidat = valider ; boutons `↶ ↷` dans la barre du brouillon ; clic sur les cases
    du jet = les effacer.
12. **Le bouton et la pastille nomment le Match, non le salut du brouillon** (écrit à chaque
    geste, ADR-0045 règle 1) : « Créer le match », puis « Mettre à jour le match #N » ; pastille
    « aucun match » / « match #N à jour » / « match #N en retard sur le brouillon ». Aucune
    phrase sur la sûreté du brouillon.
13. **Le contrat est mesuré en pixels** à deux tailles : dock bas au plancher de l'onglet
    (viewport 1280×800) et dock latéral 420 px. Assertions : `.tab-content` ne défile pas ;
    cinq lignes de candidats entièrement dans le rectangle visible (`boundingBox`, pas
    `count()`) ; le haut de la première ligne est à moins de X px du bas des cases du jet —
    celle-ci tient la règle, pas le symptôme.

## Conséquences

- `MatchInfoBar` a un troisième contexte (le brouillon en cours).
- Retirer une instruction à l'écran n'est admis qu'avec, dans la même branche, ses entrées
  `raccourcis.rst` / `manuel.rst`, `make help` et les huit `.po`.
- Un « dé mal lu, vu aussitôt » en bout de document coûte 3 K au lieu de 2 ; bilan net
  ≈ −138 K par match de 250 Actions.
- Écartés : le chiffre qui valide partout (un chiffre égaré en relecture la terminerait et
  renverrait le Cursor en bout) ; replier la palette derrière une bascule (cible morte) ; une
  page plein écran (le contrat tient dans le dock — à rouvrir s'il cède) ; une ligne de message
  unique à priorités (un filtre masquerait une Incohérence) ; une prop `columns` libre (dérive
  entre panneaux) ; un balayage paramétré de tailles (un contrat devient une distribution).

## Garde

`frontend/tests/e2e/transcription-layout.spec.js` (contrat en pixels),
`frontend/tests/e2e/transcription-budgets.spec.js` (comptes de gestes),
`frontend/src/__tests__/analysisRows.test.js`, `TranscriptionPanel.mouse.test.js`.
