# Transcription de matchs — UX et budgets de gestes

Ce document mesure ; [fonctionnel.md](fonctionnel.md) définit. Chaque flux reçoit un budget
**KLM** (Keystroke-Level Model) et le design est tenu de le respecter — les specs Playwright
du lot 3 comptent les gestes, pas seulement le résultat.

## 1. La règle de mesure

Opérateurs KLM, valeurs standard (Card, Moran & Newell) retenues pour tout le document :

| Opérateur | Valeur | Ce que c'est |
|---|---|---|
| K | 0,28 s | une touche, dactylographe moyen |
| P | 1,10 s | pointer une cible à la souris |
| B | 0,10 s | presser ou relâcher un bouton (un clic = 0,20 s) |
| H | 0,40 s | déplacer la main clavier ↔ souris |
| M | 1,35 s | préparation mentale ; **comptée une fois par décision de l'utilisateur**, identique entre designs, donc hors comparaison — *voir la réserve ci-dessous* |
| R | mesuré | attente du système : liste 0-ply d'un jet 0,8–1,2 ms (mesuré, §7), aller-retour Wails ≈ 1 ms, 2-ply 55–277 ms **hors chemin** |

**Réserve sur M, ajoutée le 2026-09-10 (ADR-0048).** M n'est hors comparaison que lorsque
les designs comparés **n'ajoutent pas de mode**. Un design qui donne deux sens à une même
touche, séparés par un état que l'utilisateur ne voit pas, lui fait payer une vérification
par tour : c'est un M, et c'est précisément celui que le modèle avait exclu. L'arbitrage du
2026-09-07 sur la touche chiffrée (§3) a été rendu sur ce compteur aveugle, et renversé pour
cette raison. Avant d'opposer un budget KLM à un design, vérifier que les deux termes de la
comparaison demandent la même chose à la mémoire.

Loi de Fitts pour affiner P quand la taille de cible varie : `T = a + b · log2(D/W + 1)`,
`a = 0`, `b = 0,15 s/bit` ; sur une fenêtre de 1024 px, D ≈ 300 px du plateau au panneau.
Une cible de 40 px : ID = 3,1 bits → 0,46 s ; de 28 px : ID = 3,6 bits → 0,54 s ; de 20 px :
ID = 4,0 bits → 0,60 s. Les P ci-dessous utilisent ces valeurs quand la cible est connue.

## 2. Le placement

Onzième onglet du panneau ancré, mode `TRANSCRIBE`, brouillon dans un store. Dock **bas**,
hauteur minimale **320 px** sur cet onglet (ADR-0048 décision 5 ; 280 px annoncés, puis 300,
puis 320 après mesure — la barre d'onglets prend 30 px du dock, et la rangée de videau porte
des mots, donc sa propre ligne) :

```
┌─────────────────────────── plateau (position du Cursor, J1 en bas) ───────────────────────────┐
│   la barre de match porte : 7 pt · 3–2 · Crawford · partie 3 · videau 1 centré · Kévin au trait│
├──────────────┬────────────────────────────────┬───────────────────────────────────────────────┤
│  ← Liste          [aucun match] [Créer le match] [Texte .mat] [Exporter] [Fermer]  ↶ ↷        │
├──────────────┼────────────────────────────────┼───────────────────────────────────────────────┤
│ palette      │ candidats (identify)           │ Transcript                                    │
│ [3][1] ✎     │  1  8/5 6/5     +0,000  ——     │ Partie 3   3–2                                │
│ [Doubler][Prendre][Passer][Abandonner]         │                                               │
│ ┌ 21 jets ─┐ │  2  13/10 6/5   −0,041 −0,041 │  Kévin              │ Alice                   │
│ │11        │ │  3  24/21 13/12 −0,103 −0,103 │  31: 8/5 6/5        │ 52: 13/8 13/11          │
│ │21 22     │ │  4  …                         │  64: 24/14          │ Doubles => 2            │
│ │31 32 33  │ │  5  …                         │  Takes              │ ▌41: 13/9 6/5 ▐ ← Cursor│
│ │41 42 43 44│ │     … 17 coups               │  …                                            │
│ │51 …      │ │                               │ Partie 2  (repliée)                           │
│ └──────────┘ │                               │                                               │
└──────────────┴───────────────────────────────┴───────────────────────────────────────────────┘
   barre d'état : « dés de Kévin »   (et, une fois, « rien à annuler »)
```

Trois colonnes, largeurs indicatives 220 / 280 / le reste. L'invariant qui gouverne la
verticale (ADR-0048) : **rien ne s'intercale entre les deux cases du jet et la première ligne
de candidats**, et cinq lignes de candidats sont visibles sans défiler. Les cibles souris — le
triangle, la rangée `[D][T][P][R]`, le secours de saisie à la main derrière `✎` — forment la
**palette**, à gauche quand la boîte est large, sous la liste quand elle est étroite.

Dock latéral (420 px) : une colonne, palette **sous** la liste, Transcript en dessous.

## 3. La machine à états du clavier

Le panneau prend les chiffres et les lettres quand il a le focus ; les touches toujours
globales (`Ctrl+*`, Espace, `?`, `Maj+J`/`Maj+K`) restent globales.

| État | Touche | Effet | État suivant |
|---|---|---|---|
| **dés attendus** | `1`–`6` | premier dé | dés attendus (un dé) |
| dés attendus (un dé) | `1`–`6` | second dé ; liste 0-ply, premier présélectionné, flèches | **jet saisi** |
| jet saisi | `j`/`k`, ↓/↑, **molette** | déplace la sélection | jet saisi |
| jet saisi | Retour arrière | efface les deux dés | dés attendus |
| jet saisi | Entrée, **double-clic** sur la ligne | valide | dés attendus (camp suivant) |
| jet saisi, **en bout de document** | `1`–`6` | **valide**, puis premier dé du tour suivant | dés attendus (un dé) |
| jet saisi, **sur une Action relue** | `1`–`6` | **recommence le jet, sur place** | dés attendus (un dé) |
| tout état hors saisie de dé | `d` | valide l'attente, `double` | réponse attendue |
| réponse attendue | `t` / `p` | `take` / `pass` | dés attendus / ouverture |
| tout | `r` puis `1`/`2`/`3` | résignation du camp au trait, niveau ; `Échap` annule | ouverture |
| tout | `h`/`l`, ←/→ | Cursor −1 / +1 | l'état de l'Action visée |
| tout | `i` / `a` | insertion avant / après | dés attendus (camp proposé) |
| tout | `x`, Suppr | supprime l'Action au Cursor | l'état de la suivante |
| tout | `s` | change le camp de l'Action au Cursor | inchangé |
| tout | `Ctrl+Z` / `Ctrl+Maj+Z` | annuler / rétablir | — |
| tout | `Échap` | abandonne la saisie en cours (dés, insertion, résignation) | l'état d'avant |
| ouverture | `1`–`6`, `1`–`6` | dé J1, dé J2 ; égalité → « relance », rejouer | jet corrigeable (gagnant) |

Danse : dès le second dé, si la liste est vide, l'Action `dance` est créée et l'on passe à
« dés attendus » pour l'autre camp — zéro touche de plus.

**La liste des brouillons prend le clavier elle aussi** (ADR-0048 décision 10) : `j`/`k` et
↓/↑ parcourent les lignes, `Entrée` ouvre celle qui est surlignée — la première l'est, et
`ListTranscriptions` rend les brouillons du plus récemment modifié au plus ancien —, `n`
ouvre le formulaire de création. Le panneau ne traitait jusqu'ici aucune touche tant qu'aucun
brouillon n'était ouvert, si bien que reprendre le travail de la veille n'avait aucun chemin
clavier.

**Arbitrage du 2026-09-07, RENVERSÉ le 2026-09-10 (ADR-0048 décision 1).** Il opposait deux
lectures de la touche chiffrée depuis « jet corrigeable » — recommencer le jet (correction à
deux touches, meilleur coup à trois) ou valider (l'inverse) — et retenait la première, en
distinguant les deux états par une phrase à l'écran.

Ce que ce raisonnement ne pouvait pas voir : les deux états étaient séparés par un
**historique invisible** (« avez-vous touché la liste ? »), donc par un mode, donc par un M
par tour — la seule grandeur que §1 déclarait hors comparaison. Le compteur qui a servi à
décider était aveugle au coût de sa propre décision (voir la réserve de §1).

La règle retenue énonce la même chose un cran plus haut : **la touche chiffrée commence un jet
là où le Cursor est.** En bout de document il n'y a rien sous le Cursor, donc elle valide le
candidat sélectionné et ouvre le jet suivant ; sur une Action relue il y a quelque chose, donc
elle en recommence le jet, sur place. C'est **un seul sens**, et la différence est un objet
dessiné — la cellule encadrée du Transcript, où l'on s'est rendu délibérément une frappe plus
tôt. Un état que l'on voit n'est pas un mode.

Le discriminant est `entry.replacing`, que le moteur expose déjà. Le meilleur coup coûte donc
**deux** touches. « Le chiffre valide partout » aurait coûté une touche de plus à chacune des
trois lignes de §4.3, et pire — `validate` n'est pas gardé par `entryDiffers`, il réécrit
l'Action et rend le Cursor à `doc.Return`, si bien qu'un chiffre égaré en relecture aurait mis
fin à la relecture.

**Une ligne de §4.3 bouge quand même**, mesurée à l'implémentation : « dé mal lu, vu aussitôt »
en **bout de document** passe de 2 K à 3 K, parce que le chiffre y valide et qu'il faut donc
effacer le jet avant de le reprendre. Sur une Action relue elle reste à 2 K. Les deux lectures
du chiffre depuis un jet complet s'excluent ; à 150 tours gagnés contre 12 perdus, c'est le bon
sens de l'échange.

Ce que la règle « un chiffre valide depuis un candidat choisi » exclut, et ce qui le remplace : dernier coup d'une
partie → Entrée ; action de videau → sa lettre valide d'abord ; erreur découverte un tour
trop tard → Retour arrière, `h`, `j`/`k`, puis `l` ou la reprise des dés.

Touches libres vérifiées le 2026-09-07 : `s`, `x`, `i`, `a`, `d`, `t`, `u` ne sont prises
nulle part ; **`n` vérifié libre le 2026-09-10** (seul `Ctrl+N` est lié,
`keyboardService.js`), pour « nouveau brouillon » depuis la liste ; `r`, `h`, `l`, `j`, `k`, `p` sont globales mais un panneau qui a le focus les
reçoit en premier (`panelKeyGuard`) ; `Ctrl+Z` est libre. Raccourci de l'onglet :
**`Ctrl+Maj+T`** (les seuls `Ctrl+lettre` libres sont `H`, `J`, `A`, `Z`, sans mnémonique ;
`Ctrl+Maj+I/F/S` existent déjà). Commande : `transcribe`, alias `tr`.

## 4. Les budgets

M compté une fois par flux, hors comparaison. K = 0,28 s.

### 4.1 Un tour de pions

| Cas | Gestes | KLM | Budget |
|---|---|---|---|
| meilleur coup joué | `3` `1` puis le jet suivant | **2 K = 0,56 s** | ≤ 0,9 s |
| meilleur coup, dernier de la partie | `3` `1` Entrée | 3 K = 0,84 s | ≤ 0,9 s |
| n-ième coup, n ≤ 5 | `3` `1` `j`×(n−1) puis le jet suivant | (n+1) K ; n = 3 → 1,12 s | ≤ 1,2 s |
| n-ième coup à la molette | `3` `1` puis n−1 crans, double-clic ou jet suivant | 2 K + (n−1) crans | souris de plein droit (R3) |
| coup loin dans la liste (rang 12) | `3` `1` `j`×11 | 13 K = 3,6 s | → filtre du lot 2 |
| idem, lot 2, filtre par clic sur le point de départ | `3` `1` H P B B H `j`×≤2 | 2 K + 2 H + P + 0,2 + ≤ 2 K ≈ 2,9 s | ≤ 3 s |
| coup joué au plateau, dés déduits (lot 2) | H, 2 × (P B B) hops, H | 2 H + 2 P + 0,4 ≈ 3,4 s ; 4 pas (double) ≈ 5,9 s | souris seule ≤ 6 s |
| dés à la souris, triangle 21 (lot 2) | H P(28 px) B B | 0,4 + 0,61 + 0,2 = **1,21 s** (mesuré) | vs grille 36 : P(20 px) = 0,66 s → 1,26 s, et 36 cibles à lire |
| dés au clavier | 2 K | 0,56 s | référence |
| effacer le jet en cours | Retour arrière, ou **clic sur les cases du jet** | 1 K / P B B | R3 |

**Mesuré le 2026-09-10 (ADR-0048).** Le meilleur coup joué revient à **deux** touches, la
validation étant portée par la première touche du tour d'après — l'annonce d'origine de ce
document, que l'arbitrage du 2026-09-07 avait fait passer à trois et que son renversement
rétablit. `Entrée` reste la sortie du dernier coup d'une partie, qui n'a pas de tour suivant.

Le clavier bat la souris d'un facteur deux sur les dés ; le triangle bat la grille 6×6 de
0,05 s par jet et supprime l'ambiguïté 3-1/1-3 (36 → 21 cibles à balayer du regard). Le coup
joué au plateau est le plus lent des trois modes de choix mais le seul qui couvre le coup
illégal et qui dispense de saisir les dés.

**Mesuré le 2026-09-07** (prototype jetable, trois variantes posées dans une fenêtre de
1024 px, D pris du centre du plateau au centre de la case). La case de 28 px donne un ID de
3,65 à 4,08 bits — et non 3,6 : la distance réelle dépasse les 300 px supposés — soit un clic
de 1,15 à 1,21 s. La case élastique (37,6 px, bloc de 236 px) gagne 0,03 s, sous la résolution
du modèle, contre 60 px de largeur et une cible dont la taille change avec le panneau : la
case de 28 px est retenue. La grille de 36 à 20 px coûte 1,17 à 1,26 s. Le classement des
trois designs tient donc, seule la valeur absolue était optimiste de 0,07 s.

### 4.2 Le videau

| Cas | Gestes | KLM |
|---|---|---|
| double, prise | `d` `t` | 2 K = 0,56 s |
| double, passe (partie finie, ouverture suivante) | `d` `p` | 0,56 s |
| redouble | `d` `t` | 0,56 s |
| à la souris (rangée `[D][T][P][R]`) | H P B B, P B B, H | 0,4 + 2 × 1,3 + 0,4 = 3,4 s |
| résignation gammon | `r` `2` | 0,56 s |

### 4.3 La correction

| Cas | Gestes | KLM |
|---|---|---|
| dé mal lu, vu aussitôt, en bout de document | Retour arrière, `4` `1` | 3 K = 0,84 s |
| dé mal lu, vu aussitôt, sur une Action relue | `4` `1` | 2 K = 0,56 s |
| candidat voisin, vu aussitôt | `j` | 0,28 s |
| erreur vue un tour plus tard, dés déjà tapés | Retour, `h`, `j`, `l` | 4 K = 1,12 s |
| erreur vue k tours plus tard | Retour, `h`×k, `j`/`k`×m, `l`×k | (1 + 2k + m) K ; k = 5, m = 1 → 3,4 s |
| coup oublié | `h`×k, `i`, `3` `1` (`j`…), Entrée, `l`×k | (2k + 4 + m) K |
| coup en double | `h`×k, `x`, `l`×k | (2k + 1) K |
| camp faux | `h`×k, `s`, `l`×k | (2k + 1) K |

Le coût d'une correction croît avec la distance k, jamais avec la longueur du match : le
Replay est de l'ordre de la milliseconde et n'entre pas dans le budget. À la souris, un clic
sur la cellule du Transcript remplace `h`×k (P B B = 1,3 s).

**Mesuré le 2026-09-07, ligne « coup oublié ».** Elle coûte une touche de plus que ce
document annonçait : `2k + 4 + m` et non `2k + 3 + m`. Un déplacement du Cursor valide une
correction EN PLACE — c'est ce qui garde la ligne « erreur vue un tour plus tard » à quatre
touches — mais il ne valide pas une INSERTION : `commitCorrection` (`transcript/apply.go`) ne
commet qu'une Entry de mode `EntryReplace`, et le moteur interrogé laisse le document à sept
Actions sur un `cursor_forward` là où `validate` le porte à huit. La touche qui manquait est
Entrée. Les deux autres lignes en `2k + 1` sont inchangées : `x` et `s` agissent sur l'Action
au Cursor et n'ouvrent aucune saisie.

### 4.4 Le match entier

Un match en 7 points, ~250 Actions, ~60 % de meilleurs coups joués, ~15 % de rangs 2–3,
~10 % au-delà, 15 doubles/prises/passes, 12 corrections. Avec le meilleur coup à deux touches
(§4.1, 2026-09-10) : ≈ 250 × 0,75 s + 12 × 2 s ≈ **3,5 minutes de gestes**, hors lecture de la
vidéo, contre 4,4 sous l'arbitrage renversé. Le temps de l'outil est sous le temps de la
source ; c'est le critère.

### 4.5 La création et l'enregistrement

| Cas | Gestes | KLM |
|---|---|---|
| **reprendre le brouillon en cours** | `Ctrl+Maj+T`, Entrée | **2 K = 0,56 s** |
| nouveau brouillon | `Ctrl+Maj+T`, `n`, `7`, Entrée | 4 K = 1,12 s |
| idem à la souris | `Ctrl+Maj+T`, clic sur « Nouveau », `7`, Entrée | H + P + 2B + 2 K ≈ 2,3 s |
| créer le match | `Ctrl+Entrée` dans le panneau | 1 K |
| voir le texte `.mat` | bouton, modale | P B B |
| exporter `.mat` | bouton, dialogue système | P B B + dialogue |
| fermer | bouton, confirmation si aucun match n'a été créé | P B B (+ 1 K) |

**Corrigé le 2026-09-10 (ADR-0048 décision 10).** Ce tableau comptait « nouvelle » comme une
frappe : le trajet souris seul coûte H + P + 2B = 1,7 s, plus que le total de 1,5 s annoncé.
Et il ne comptait **pas** la reprise d'un brouillon existant — le geste de chaque session
après la première, puisqu'un match en 7 points ne se transcrit pas d'une traite —, qui n'avait
alors aucun chemin clavier du tout.

`Ctrl+S` a été essayé et abandonné : `isAlwaysGlobal` rend vrai pour tout combo `Ctrl`, et le
dispatcher global le tient pour « sauver la position ». C'est donc `Ctrl+Entrée`, pris avant
la garde et arrêté net.

## 5. Le retour visuel, et le domicile de chaque message

**R2 (ADR-0048) : un message habite là où est ce dont il parle.** Un seul domicile par
message ; le panneau ne porte plus de prose.

- **Plateau** : position du Cursor, J1 en bas ; flèches du candidat sélectionné
  (`selectedMoveStore`), que la **molette** fait défiler sans quitter le plateau des yeux ;
  points de départ possibles en surbrillance (`quizPlaySourcesStore`), destinations de la
  source choisie (`quizPlayTargetsStore`).
- **Barre de match** (`MatchInfoBar`) : longueur, score, « Crawford » dérivé, camp au trait,
  valeur et propriétaire du videau — l'état du brouillon, là où l'œil est déjà quand il
  regarde le plateau.
- **Barre du brouillon** : les cinq gestes qui font sortir le brouillon de lui-même, plus
  `↶ ↷`, plus **l'état du match** — « aucun match », « match #123 à jour », « match #123 en
  retard sur le brouillon ». Elle ne dit **rien** de la sûreté du brouillon : il est écrit
  après chaque Action, il n'y a rien à signaler, et l'absence d'alarme est le message juste.
- **Liste des candidats** : projection `identify` — rang, notation, équité, écart au meilleur
  (Évaluation 0-ply, jamais stockée). Une **puce** sur l'en-tête quand un filtre par point est
  posé, qui donne aussi le moyen de le lever.
- **Transcript** : cellule du Cursor encadrée ; Incohérences en décoration de cellule avec
  info-bulle nommant laquelle, plus un **bandeau** en tête de la colonne quand le document en
  porte ; partie courante ouverte, autres repliées ; en-tête de partie = score initial +
  Crawford. Le texte `.mat` exact est une **modale** (bouton dans la barre du brouillon, à côté
  d'« Exporter ») et non un volet sous le tableau : c'est de l'ASCII aligné en colonnes, sa
  ligne la plus longue fait 62 caractères ≈ 409 px, et une colonne de 320 px en détruit
  l'alignement, c'est-à-dire la seule raison de le regarder.
- **Barre d'état** : mode `TRANSCRIBE` ; l'Action attendue en un mot (« dés de Kévin »,
  « réponse d'Alice au double », « relance », « danse », « correction en place », « coup à
  revoir ») ; la progression du lot d'analyse après création du match ; et la **réponse
  transitoire d'un geste sans effet** (~1,5 s) — « rien à annuler », « aucune Action sous le
  curseur ». `t` et `p` en sont exceptées : sans offre en face elles filent au dispatcher
  global, où `p` vaut le compte de pions, légitimement voulu pendant une transcription.
- **Ce qui a disparu** : les six lignes d'instruction permanentes (« Jouez le coup sur le
  plateau », « Jet corrigeable : … »). Une instruction permanente est l'aveu qu'un geste ne se
  devine pas ; elle vit dans `raccourcis.rst`, dans l'aide en ligne qui en est engendrée
  (ADR-0034) et dans la visite guidée, pas sous les dés lus 250 fois par tour de match.

## 6. Tolérance aux fautes, résumé

Tout geste est annulable (`Ctrl+Z`) ; rien n'est refusé ni supprimé par le logiciel ; toute
Incohérence est visible à l'endroit exact ; un dé peut être retapé tant que le coup n'est pas
validé ; le Cursor revient où il était après une correction ; le brouillon est écrit après
chaque Action ; un plantage ne perd que la pile d'annulation.

## 7. Prototype et vérification

- **Prototype jetable** (skill `prototype`) avant le lot 2 : le triangle des 21 jets et la
  liste filtrable, pour confirmer les P mesurés ci-dessus sur une vraie fenêtre.
- **Specs Playwright** — ÉCRITES le 2026-09-07 :
  `frontend/tests/e2e/transcription-budgets.spec.js`, dix-sept specs qui comptent les gestes
  par `helpers/gestureCount.js` et échouent au-delà du budget (vérifié : deux touches
  superflues font rougir « meilleur coup joué »). Le moteur Go y est remplacé par un
  document figé (`helpers/transcriptionDraft.js`) qui ne dérive que deux faits, le Cursor et
  la réponse attendue après un double ; ce que la suite de gestes produit comme DOCUMENT est
  tenu en Go, Action par Action. Deux lignes de §4.1 ne sont pas là et ne le seront pas
  ainsi : « filtré par clic sur le point de départ » et « coup joué au plateau » passent par
  un point du damier, dessiné par two.js et sans cible DOM — les viser à la coordonnée
  donnerait une spec instable pour une mesure que `transcriptionFilter.test.js` fait déjà en
  comptant clics et touches. La ligne « videau à la souris » de §4.2 est mesurée par
  `transcriptionKeys.cubeMouse.test.js`, livré avec le videau à la souris (#353) : deux
  clics, `H + 2×(P+2B) + H = 3,4 s`. Le conflit
  de port avec gammonGo est paré par `BLUNDERDB_E2E_PORT`
  (`frontend/playwright.config.js`).
- **Latence** — MESURÉE le 2026-09-07, le seuil de 20 ms annoncé ici était faux d'un facteur
  cinq. Un rejeu complet de 300 Actions coûte **39 ms** (101 ms sur un document plus dense),
  entièrement dans `domain.LegalMoves` (175 µs un jet ordinaire, 3,6 ms un double). Le rejeu
  est donc **incrémental** : ajouter une Action à la fin, le geste de chaque tour, coûte
  **104 µs**, soit 380 fois moins. Une correction ne coûte que la queue du document qu'elle
  invalide. Les tests figent les deux : un seuil de 5 ms pour l'ajout, et l'égalité stricte
  entre le rejeu incrémental et le rejeu complet sur un document couvrant tous les `kind` et
  toutes les incohérences.
- **Latence de la liste 0-ply** — MESURÉE le 2026-09-07
  (`internal/gui/transcription_latency_test.go`, position de contact, 892 coups sur les
  21 jets). Le geste de chaque tour — `domain.LegalMoves` d'un jet puis le classement 0-ply
  de ses coups, les deux appels que fait `computeCandidates` — coûte **0,8 à 1,2 ms**, soit
  un trentième de son seuil de 30 ms. Le coup joué au plateau, qui balaie les 21 jets pour
  déduire lequel a été lancé, coûte **5 à 6,6 ms** ; son seuil est posé à 60 ms, le double
  de celui annoncé, parce que le même travail a pris 19 ms sur une machine chargée et qu'un
  garde-fou qui rougit chez le voisin ne garde rien. Le seuil de 30 ms proposé pour « les
  21 jets triés au 0-ply » est en revanche **démenti** : ce balayage coûte 35 à 46 ms, et
  jusqu'à 177 ms sous charge. Il ne l'est que pour un geste qui n'existe pas — rien dans
  l'application ne classe vingt et un jets — et il reste mesuré, sans seuil serré, comme
  prix de la question que §4.1 laisse ouverte.
