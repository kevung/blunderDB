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
| M | 1,35 s | préparation mentale ; **comptée une fois par décision de l'utilisateur**, identique entre designs, donc hors comparaison |
| R | mesuré | attente du système : 0-ply < 1 ms, aller-retour Wails ≈ 1 ms, 2-ply 55–277 ms **hors chemin** |

Loi de Fitts pour affiner P quand la taille de cible varie : `T = a + b · log2(D/W + 1)`,
`a = 0`, `b = 0,15 s/bit` ; sur une fenêtre de 1024 px, D ≈ 300 px du plateau au panneau.
Une cible de 40 px : ID = 3,1 bits → 0,46 s ; de 28 px : ID = 3,6 bits → 0,54 s ; de 20 px :
ID = 4,0 bits → 0,60 s. Les P ci-dessous utilisent ces valeurs quand la cible est connue.

## 2. Le placement

Onzième onglet du panneau ancré, mode `TRANSCRIBE`, brouillon dans un store. Dock **bas** :

```
┌─────────────────────────── plateau (position du Cursor, J1 en bas) ───────────────────────────┐
│                                                                                              │
├───────────────────────── saisie ─────────────────┬──────────────── Transcript ────────────────┤
│ [7 pt] [Crawford]  score 3–2   ● J1 au trait      │ Partie 3   3–2                             │
│ dés  [3][1]        ┌ 21 jets ┐                    │ ─────────────────────────────────────────  │
│ ┌ candidats (0-ply) ────────┐ │ 11 21 22 ..│      │  Kévin              │ Alice                │
│ │▶ 1  8/5 6/5     +0,000    │ │ 31 32 33   │      │  31: 8/5 6/5        │ 52: 13/8 13/11       │
│ │  2  13/10 6/5   −0,041    │ │ ..         │      │  64: 24/14          │ Doubles => 2         │
│ │  3  24/21 13/12 −0,103    │ └────────────┘      │  Takes              │ ▌41: 13/9 6/5 ▐   ← Cursor │
│ │  … 17 coups              │ [D] [T] [P] [R]      │  …                                          │
│ └───────────────────────────┘                     │ Partie 2  (repliée)                         │
└──────────────────────────────────────────────────┴──────────────────────────────────────────────┘
```

Dock latéral : saisie au-dessus, Transcript en dessous. Le triangle des jets et la rangée de
boutons sont le lot 2 ; le lot 1 est la moitié gauche sans eux.

## 3. La machine à états du clavier

Le panneau prend les chiffres et les lettres quand il a le focus ; les touches toujours
globales (`Ctrl+*`, Espace, `?`, `Maj+J`/`Maj+K`) restent globales.

| État | Touche | Effet | État suivant |
|---|---|---|---|
| **dés attendus** | `1`–`6` | premier dé | dés attendus (un dé) |
| dés attendus (un dé) | `1`–`6` | second dé ; liste 0-ply, premier présélectionné, flèches | **jet corrigeable** |
| jet corrigeable | `1`–`6` | recommence : premier dé | dés attendus (un dé) |
| jet corrigeable | `j`/`k`, ↓/↑ | déplace la sélection | **candidat choisi** |
| jet corrigeable · candidat choisi | Retour arrière | efface les deux dés | dés attendus |
| jet corrigeable · candidat choisi | Entrée | valide | dés attendus (camp suivant) |
| jet corrigeable · candidat choisi | `1`–`6` (depuis candidat choisi) | **valide**, puis premier dé du tour suivant | dés attendus (un dé) |
| candidat choisi | `j`/`k` | déplace | candidat choisi |
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

Ce que la règle « un chiffre valide » exclut, et ce qui le remplace : dernier coup d'une
partie → Entrée ; action de videau → sa lettre valide d'abord ; erreur découverte un tour
trop tard → Retour arrière, `h`, `j`/`k`, puis `l` ou la reprise des dés.

Touches libres vérifiées le 2026-09-07 : `s`, `x`, `i`, `a`, `d`, `t`, `u` ne sont prises
nulle part ; `r`, `h`, `l`, `j`, `k`, `p` sont globales mais un panneau qui a le focus les
reçoit en premier (`panelKeyGuard`) ; `Ctrl+Z` est libre. Raccourci de l'onglet :
**`Ctrl+Maj+T`** (les seuls `Ctrl+lettre` libres sont `H`, `J`, `A`, `Z`, sans mnémonique ;
`Ctrl+Maj+I/F/S` existent déjà). Commande : `transcribe`, alias `tr`.

## 4. Les budgets

M compté une fois par flux, hors comparaison. K = 0,28 s.

### 4.1 Un tour de pions

| Cas | Gestes | KLM | Budget |
|---|---|---|---|
| meilleur coup joué | `3` `1` (le chiffre suivant valide) | 2 K = 0,56 s | ≤ 0,6 s |
| n-ième coup, n ≤ 5 | `3` `1` `j`×(n−1) | (n+1) K ; n = 3 → 1,12 s | ≤ 1,2 s |
| coup loin dans la liste (rang 12) | `3` `1` `j`×11 | 13 K = 3,6 s | → filtre du lot 2 |
| idem, lot 2, filtre par clic sur le point de départ | `3` `1` H P B B H `j`×≤2 | 2 K + 2 H + P + 0,2 + ≤ 2 K ≈ 2,9 s | ≤ 3 s |
| coup joué au plateau, dés déduits (lot 2) | H, 2 × (P B B) hops, H | 2 H + 2 P + 0,4 ≈ 3,4 s ; 4 pas (double) ≈ 5,9 s | souris seule ≤ 6 s |
| dés à la souris, triangle 21 (lot 2) | H P(28 px) B B | 0,4 + 0,54 + 0,2 = 1,14 s | vs grille 36 : P(20 px) = 0,60 s → 1,20 s, et 36 cibles à lire |
| dés au clavier | 2 K | 0,56 s | référence |

Le clavier bat la souris d'un facteur deux sur les dés ; le triangle bat la grille 6×6 de
0,06 s par jet et supprime l'ambiguïté 3-1/1-3 (36 → 21 cibles à balayer du regard). Le coup
joué au plateau est le plus lent des trois modes de choix mais le seul qui couvre le coup
illégal et qui dispense de saisir les dés.

### 4.2 Le videau

| Cas | Gestes | KLM |
|---|---|---|
| double, prise | `d` `t` | 2 K = 0,56 s |
| double, passe (partie finie, ouverture suivante) | `d` `p` | 0,56 s |
| redouble | `d` `t` | 0,56 s |
| à la souris (lot 2) | H P B B, P B B, H | 0,4 + 2 × 1,3 + 0,4 = 3,4 s |
| résignation gammon | `r` `2` | 0,56 s |

### 4.3 La correction

| Cas | Gestes | KLM |
|---|---|---|
| dé mal lu, vu aussitôt (jet corrigeable) | `4` `1` | 2 K = 0,56 s |
| candidat voisin, vu aussitôt | `j` | 0,28 s |
| erreur vue un tour plus tard, dés déjà tapés | Retour, `h`, `j`, `l` | 4 K = 1,12 s |
| erreur vue k tours plus tard | Retour, `h`×k, `j`/`k`×m, `l`×k | (1 + 2k + m) K ; k = 5, m = 1 → 3,4 s |
| coup oublié | `h`×k, `i`, `3` `1` (`j`…), `l`×k | (2k + 3 + m) K |
| coup en double | `h`×k, `x`, `l`×k | (2k + 1) K |
| camp faux | `h`×k, `s`, `l`×k | (2k + 1) K |

Le coût d'une correction croît avec la distance k, jamais avec la longueur du match : le
Replay est de l'ordre de la milliseconde et n'entre pas dans le budget. À la souris, un clic
sur la cellule du Transcript remplace `h`×k (P B B = 1,3 s).

### 4.4 Le match entier

Un match en 7 points, ~250 Actions, ~60 % de meilleurs coups joués, ~15 % de rangs 2–3,
~10 % au-delà, 15 doubles/prises/passes, 12 corrections : ≈ 250 × 0,9 s + 12 × 2 s ≈ **4,4
minutes de gestes**, hors lecture de la vidéo. Le temps de l'outil est sous le temps de la
source ; c'est le critère.

### 4.5 La création et l'enregistrement

| Cas | Gestes | KLM |
|---|---|---|
| nouveau brouillon | `Ctrl+Maj+T`, « nouvelle », `7`, Entrée | ≈ 1,5 s |
| enregistrer | `Ctrl+S` dans le panneau (sauvegarde du brouillon en Match) | 1 K |
| exporter `.mat` | bouton, dialogue système | P B B + dialogue |
| fermer | bouton, confirmation si jamais enregistré | P B B (+ 1 K) |

`Ctrl+S` est « sauver la position » ailleurs ; dans le panneau il enregistre le brouillon,
c'est la même intention. À vérifier au lot 1 que `isAlwaysGlobal` laisse le panneau la
prendre — sinon `Ctrl+Entrée`.

## 5. Le retour visuel

- **Plateau** : position du Cursor, J1 en bas ; flèches du candidat sélectionné
  (`selectedMoveStore`) ; en lot 2, points de départ possibles en surbrillance
  (`quizPlaySourcesStore`), destinations de la source choisie (`quizPlayTargetsStore`).
- **Barre de match** (`MatchInfoBar`) : longueur, score, « Crawford » dérivé, camp au trait,
  valeur et propriétaire du videau, « brouillon non enregistré » / « enregistré il y a … ».
- **Liste des candidats** : rang, notation, équité et écart au meilleur (Évaluation 0-ply,
  jamais stockée) ; la ligne validée garde son écart visible dans le Transcript au survol.
- **Transcript** : cellule du Cursor encadrée ; Incohérences en décoration de cellule avec
  info-bulle nommant laquelle ; partie courante ouverte, autres repliées ; en-tête de partie
  = score initial + Crawford ; texte `.mat` exact dans un volet dépliable, bouton « copier ».
- **Barre d'état** : mode `TRANSCRIBE`, Action attendue en un mot (« dés de Kévin »,
  « réponse d'Alice au double »), progression du lot d'analyse après enregistrement.

## 6. Tolérance aux fautes, résumé

Tout geste est annulable (`Ctrl+Z`) ; rien n'est refusé ni supprimé par le logiciel ; toute
Incohérence est visible à l'endroit exact ; un dé peut être retapé tant que le coup n'est pas
validé ; le Cursor revient où il était après une correction ; le brouillon est écrit après
chaque Action ; un plantage ne perd que la pile d'annulation.

## 7. Prototype et vérification

- **Prototype jetable** (skill `prototype`) avant le lot 2 : le triangle des 21 jets et la
  liste filtrable, pour confirmer les P mesurés ci-dessus sur une vraie fenêtre.
- **Specs Playwright** (lot 3) : une spec par ligne des tableaux 4.1–4.3, qui compte les
  `keyboard.press`/`mouse.click` émis et échoue si le nombre dépasse le budget, sous
  `frontend/tests/e2e/`. Le conflit de port avec gammonGo est déjà paré dans le dépôt par
  `BLUNDERDB_E2E_PORT` (`frontend/playwright.config.js`).
- **Latence** : un test Go mesure `transcript.Replay` sur un match de 300 Actions (seuil
  proposé : 20 ms) et `LegalMoves` + tri 0-ply sur les 21 jets d'une position de contact
  (seuil : 30 ms) ; au-delà, rouge.
