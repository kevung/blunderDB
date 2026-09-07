# Diriger un tournoi — spécification fonctionnelle

Décision de référence : [ADR-0047](../../docs/adr/0047-directing-a-tournament-creates-its-matches-before-they-are-played.md)
(diriger un tournoi crée ses Matchs avant qu'ils soient joués). Vocabulaire : `CONTEXT.md`,
section « Directing a tournament » (Tournament, Direction, Participant, Directory, Slot).
Moteur : Nicomaque (`github.com/PileOfCells/backgammon-tournoi`), créé par Nicolas Harmand.
Ce document dit *ce que* le logiciel fait ; [ux.md](ux.md) dit *par quels gestes* ;
[integration.md](integration.md) dit *ce que cela touche* ; [nicomaque.md](nicomaque.md) dit
*ce que le moteur doit gagner* ; [plan.md](plan.md) dit *dans quel ordre*. Les seize décisions
du cadrage sont dans [decisions.md](decisions.md).

## 1. Les objets

| Objet | Dans blunderDB | Dans Nicomaque | Note |
|---|---|---|---|
| Tournament | ligne `tournament` (existante) | `Config.Name` | une entité, pas deux (D2) |
| Direction | table append-only, une ligne par événement | `Journal` | source de vérité ; jamais modifiée, seulement prolongée |
| état dérivé | jamais stocké, rejoué à l'ouverture | `State` (`Replay`) | classement, arbres, appariements, propositions |
| Participant | événement `player_added` | `Player{ID, Name, Club, Rating}` | `ID` = slug du nom, unique par Direction |
| Slot | événement `match_started` + colonne `match.direction_match_id` | `Match` (`M12`) | l'emplacement qu'un Match de la base remplit |
| Table | configuration + événements | `Config.Tables` + exceptions (à ajouter) | grille des tables |
| Phase, Section | dérivés | `PhaseState`, `Section` | suisse, tableau, GSL, poules, consolante… |
| proposition | dérivée, jamais stockée | `Action` | ce que le moteur suggère maintenant |

Un Tournament sans Direction est ce qu'il est aujourd'hui. Un Tournament avec Direction est
*dirigé*. Rien de ce qui existe (matchs rattachés, commentaire, badge PR, export, picker de
recherche) ne change de sens.

## 2. Le cycle de vie d'une Direction (D7)

| État | Entrée | Ce qui est permis | Ce qui ne l'est plus |
|---|---|---|---|
| **en préparation** | création | tout : la configuration se réécrit en place, les inscriptions vont et viennent | — |
| **en cours** | premier match lancé | inscriptions (§4.4), propositions, résultats, corrections, changement de configuration *par événement* (§3.5) | changer le type de la phase courante ou d'une phase passée |
| **terminée** | événement `finished` | lecture, sorties (§9), rattachement de Matchs (§7) | tout événement de direction, sauf `reopened` |
| réouverte | événement `reopened` | comme *en cours* ; le classement final est recalculé à la prochaine clôture | — |

En préparation, la configuration est un brouillon réécrit à chaque changement ; le premier
match lancé la fige dans l'événement `created` et passe à *en cours*. La Direction se
**supprime** indépendamment du Tournament : les Slots perdent leur lien, les Matchs gardent
leur tournoi, le Tournament reste. Supprimer le Tournament supprime sa Direction et délie ses
Matchs (comportement actuel).

## 3. La configuration

### 3.1 Les phases

Une suite de phases, chacune d'un des cinq types de Nicomaque, avec ses options :

| Type | Options | Défauts |
|---|---|---|
| suisse à vies | vies, mode (continu / rondes), appariement (aléatoire / par victoires), éviter les clubs, rematch autorisé, bascule (Σvies = 2^k), `batch_minutes` (§6.3) | 2 vies, continu, aléatoire, pas de bascule, pas de lot |
| tableau à vies | longueurs par tour, longueur de finale | — |
| tableau | consolante, dernière chance, réconciliation, recharge, longueurs par tour, têtes de série (§3.4) | élimination simple |
| blocs GSL | bascule | — |
| poules | taille, qualifiés par poule | 4, 2 |

Chaque phase a une longueur de match, et pour un tableau une liste de longueurs **du dernier
tour vers le premier** (`9 / 11 / 13 / 15` se lit finale 15). L'entrée d'une phase est
*survivants* (avec leurs vies), *tous* ou *les N premiers*. Des configurations nommées
(celles de `tournoi-td` : suisse + tableau, élimination, double, GSL, poules) sont proposées
comme points de départ.

### 3.2 Les tables (D13)

Un nombre de tables, puis des exceptions : *indisponible* (plateau cassé), *réservée* à une
section ou une phase (retransmission pour la finale). Le moteur n'assigne jamais une table
indisponible ni une table réservée hors de son usage. Une proposition sans table libre est
**« en attente de table »** ; le TD peut la lancer avec un numéro saisi. Un match en cours
change de table par l'événement `table_changed`.

### 3.3 Les pauses et le rythme (D11)

Des plages horaires (repas) ; une durée moyenne par point (8 min par défaut, remplacée par la
moyenne observée dès qu'il y a des matchs finis) ; un seuil de lenteur (1,5 × attendu).

### 3.4 Le tirage

Une graine, tirée à la création et affichée (un tirage est reproductible : rejouer la
Direction redonne le même tableau). **Têtes de série** : option de tirage désactivée par
défaut — l'étude de Nicomaque les écarte — qui, activée, place les joueurs par cote d'entrée
(1 contre 16, 2 contre 15…). Un tirage est un événement : une fois fait, il ne se refait
pas ; un retardataire n'en provoque jamais un nouveau (§4.4).

### 3.5 Changer la configuration en cours (D7)

Un événement `config_changed` porte la configuration entière ; le moteur la valide et refuse
ce qui changerait le type d'une phase commencée. Sont ainsi modifiables en cours : la
bascule, les longueurs des matchs à venir, les tables, les pauses, la dotation, et une phase
**ajoutée après** la phase courante (une consolante décidée le samedi soir). L'état est
recalculé et les avertissements affichés (§5.7).

### 3.6 La dotation (D14)

Droit d'entrée ; retenue d'organisation (montant ou pourcentage) ; puis **une structure par
section** (principal, consolante, dernière chance) : pourcentages du reste, ou montants
fixes. Arrondi à l'unité, reste au premier. Le pool se recalcule à chaque inscription et
retrait ; la vue l'affiche à côté du nombre d'inscrits.

## 4. Les inscriptions

### 4.1 Un Participant (D3)

Nom, club, cote d'entrée (un PR, plus bas = meilleur, 0 = inconnu). L'identifiant est un slug
du nom, unique dans la Direction ; deux homonymes se distinguent par le club ou un suffixe
que le TD voit. Le nom est **le nom du Player** que porteront les Matchs de ses Slots : à la
saisie, l'autocomplétion propose les Players de la base ; en choisir un fixe l'orthographe et
pré-remplit la cote avec le PR du joueur dans la base. Rien n'est inféré ensuite.

### 4.2 L'annuaire (D3)

La liste de tous les Participants de toutes les Directions de la base, dédoublonnée par
nom, avec le club et la cote de la dernière inscription. Une vue, jamais une table. Trois
gestes : **reprendre les inscrits** d'un tournoi précédent (tous ou une sélection), **exporter
en CSV**, **importer un CSV** (colonnes `nom, club, cote` ; séparateur détecté ; `players/csv.go`
côté Nicomaque).

### 4.3 Ordre et effectif

L'ordre d'inscription est conservé et visible ; l'effectif, le pool, le nombre de byes qu'il
impliquerait au premier tour (tableau) et la fin estimée (§6) se lisent en préparation pour
aider le TD à choisir le format.

### 4.4 Retardataires et retraits (D12)

| Cas | Règle |
|---|---|
| inscrit avant le premier match | comme tout Participant |
| suisse avant bascule | entre avec toutes ses vies (comportement actuel) |
| tableau tiré, une place BYE libre à ce tour | prend la place (`player_added` avec `slot`) |
| tableau tiré, aucune place | inscrit, entre dans la prochaine phase ou section qui l'admet ; la vue le dit ; jamais de retirage |
| forfait pour un match | résultat `forfeit` sans retrait ; le joueur reste dans le tournoi et suit le chemin d'un perdant |
| retrait immédiat | `player_withdrawn` : ses matchs en cours sont perdus par forfait, ses matchs non lancés de graphe aussi |
| retrait différé | `player_withdrawn` avec `after_current` : n'est plus apparié, finit son match en cours |
| retour d'un retiré | `player_added` à nouveau (le moteur le permet déjà) |

## 5. Diriger

### 5.1 Les propositions (D6)

À chaque événement, et à la demande, le moteur propose : lancer un match (avec longueur et
table), donner un bye, tirer un tableau ou des groupes (le tirage est joint), passer à la
phase suivante, clore, ou attendre (avec la raison). La vue les liste **en une file**, la
plus urgente en tête ; chacune se confirme en un geste, la file entière en un geste. Une
proposition confirmée est un événement ; une proposition ignorée revient à l'appel suivant.

### 5.2 L'action manuelle (D6)

Toujours disponible : lancer un match entre deux Participants libres de la phase courante,
avec la longueur et la table choisies ; donner un bye ; passer à la phase suivante ; clore.
Le moteur ne refuse que l'impossible (joueur inconnu, déjà en match, contre lui-même). Un
match manuel qui ne correspond pas au graphe (mauvais joueur en demi-finale) est accepté et
produit un avertissement permanent (§5.7).

### 5.3 Le résultat (D8)

Le **vainqueur est la seule chose exigée**. Le score est libre : les deux, ou aucun. Une
remarque facultative (« tombé au temps », « abandon : … ») est un champ texte de l'événement
de résultat, montré et saisi de la façon la plus discrète possible — elle sert une fois par
tournoi. Un forfait se déclare à la même place. Un score incohérent avec la longueur (7–4 en
5 points, deux scores ≥ longueur) est **accepté avec avertissement**. Quand le Slot porte un
Match dont le score final est connu, le résultat est pré-rempli et se confirme en un geste.
Le résultat est la parole du TD : il n'est jamais écrit par un import.

### 5.4 Corriger, annuler

Une **correction** remplace le résultat d'un match fini (événement `result_corrected`) ;
une **annulation** retire un match lancé par erreur (`match_cancelled`) ; les deux
recalculent tout et affichent ce qui n'est plus cohérent. Une correction dans un tableau
laisse le TD devant un arbre faux : le moteur **propose la réparation** (annuler les matchs
qui en dépendent, relancer les bons) comme une suite de propositions ordinaires (§5.1) — le
TD confirme ou fait autrement. Aucun événement n'est jamais effacé ; l'historique complet
est lisible (§5.8).

### 5.5 Les tables (D13)

Une grille : une case par table, avec le match, le temps écoulé et un signe de lenteur, ou
*libre*, *indisponible*, *réservée*. Changer la table d'un match en cours se fait depuis la
grille ou depuis le match.

### 5.6 Les phases

Le passage de phase est une proposition comme une autre. La vue montre la phase courante,
les précédentes (repliées, consultables), et la suivante (ce qu'elle attend : Σvies, fin
des poules). La bascule Σvies = 2^k est affichée comme un compte à rebours de vies.

### 5.7 Les avertissements

Le moteur signale ce qui n'est plus cohérent (match de tableau joué par les mauvais
joueurs, résultat hors longueur, proposition qui finirait pendant une pause, match lent,
Match rattaché qui contredit le résultat). Un avertissement est **visible en permanence,
jamais bloquant**, attaché à l'objet qu'il concerne (match, Slot, table, proposition) et
compté dans la bande d'horloge. Il disparaît quand la cause disparaît, jamais parce qu'on
l'a lu.

### 5.8 L'historique

La Direction est lisible en clair : chaque événement avec son heure, dans l'ordre, filtrable
par joueur et par match. C'est la trace que Sophie imprime et celle que Léa relit quand elle
revient à la table. Un événement `note` permet une annotation libre du TD, horodatée.

## 6. Le temps (D11)

### 6.1 La bande d'horloge

Toujours visible dans la vue tournoi : heure, temps écoulé depuis le premier match, matchs
joués / en cours / restants (estimation du moteur), min/point observé contre planifié, **fin
estimée** de la phase et du tournoi (prévision par simulation, rafraîchie à chaque événement
et chaque minute), nombre d'avertissements, prochaine pause.

### 6.2 Les matchs lents

Un match qui dépasse le seuil (1,5 × sa durée attendue) est marqué partout où il apparaît
(grille des tables, matchs en cours). Le seuil se règle dans la configuration.

### 6.3 Les micro-rondes

Par défaut, `Propose` n'est appelé qu'à la demande du TD et après chaque événement : le TD
qui clique « proposer » quand il le veut *fait* la micro-ronde. Avec `batch_minutes = X` sur
une phase suisse continue, les joueurs libres attendent l'échéance ; la vue montre la file
d'attente et le compte à rebours ; à l'échéance, le moteur apparie au hasard tous les
joueurs libres d'un même groupe de défaites, le reste attend la suivante, et les
propositions apparaissent (badge, barre d'état — §8) sans se lancer seules.

### 6.4 Les pauses

Une proposition dont la fin attendue tombe dans une pause porte l'avertissement « finirait
pendant la pause » ; le TD décide.

## 7. Les Slots et les Matchs (D1, D3, D8)

### 7.1 Remplir un Slot

Deux façons, toutes deux explicites :

- **Transcrire depuis le Slot** : crée une Transcription dont l'en-tête est hérité (les deux
  noms, la longueur, le tournoi, la ronde ou le tour, la date) et **réserve le Slot dès le
  brouillon** ; l'enregistrement rattache le Match. Le Slot montre « brouillon » puis « match ».
- **Rattacher un Match existant** : choisir un Match de la base (le picker propose d'abord
  ceux du même Tournament dont les deux noms coïncident avec les Participants, puis les
  autres). Après un import de fichiers, la vue liste les Matchs du Tournament non rattachés
  avec, pour chacun, le Slot dont les noms coïncident : un geste par ligne, jamais
  automatique.

Un Slot porte au plus un Match ; un Match remplit au plus un Slot. Rattacher un Match à un
Slot le rattache aussi au Tournament s'il ne l'était pas.

### 7.2 L'écart

Un Match rattaché dont le score final ou la longueur contredit le résultat du Slot produit
un avertissement sur le Slot ; le TD corrige le résultat (§5.4) ou laisse l'avertissement.

### 7.3 Détacher, supprimer

Détacher vide le lien, ne touche ni le Match ni le résultat. Supprimer un Match rattaché
vide le Slot. Supprimer la Direction vide tous les Slots (§2).

### 7.4 Ce que le Match y gagne

Depuis le panneau Match, un Match rattaché affiche son Slot (tournoi, phase, tour, table) ;
depuis la vue tournoi, chaque Slot rempli ouvre son Match sur le plateau (l'onglet change,
le plateau revient — §8). La recherche par tournoi existe déjà et ne change pas.

## 8. Cohabitation (D15)

La vue tournoi occupe la zone principale quand l'onglet Tournoi est actif et qu'une
Direction est ouverte ; tout autre onglet ramène le plateau sans rien fermer. Une seule
Direction ouverte à la fois ; en ouvrir une autre ferme la première (rien à sauver : tout est
déjà écrit). Ouvrir un Tournament sans Direction propose d'en créer une. La Direction vit en
arrière-plan : horloge, échéances de micro-ronde. L'onglet Tournoi porte un badge (nombre de
propositions en attente) et la barre d'état une ligne (« 3 propositions · 6 matchs en
cours · fin estimée 23 h 10 »). `Ctrl+Maj+D` et la commande `direct` font l'aller-retour
entre la vue tournoi et le plateau ; `direct <nom>` ouvre un tournoi.

## 9. Les sorties (D4, D10)

| Sortie | Contenu | Quand |
|---|---|---|
| page HTML autonome | arbres SVG, tableau des vies, grille des tables, matchs en cours, file d'attente, classement, dotation ; sans réseau, un seul fichier | réécrite à chaque événement dans un dossier choisi ; ouverte sur un second écran ou projetée |
| feuille d'appariements | les matchs d'une ronde, d'un bloc ou d'un tour, à imprimer ; une ligne par match, case pour le score | à la demande |
| classement CSV | rang, joueur, club, note, prix, par section | à la demande, et à la clôture |
| annuaire CSV | §4.2 | à la demande |
| Direction JSON | le journal Nicomaque brut, rejouable par `tournoi-demo -rejouer` | à la demande |
| export de base | la Direction voyage avec son Tournament (liste blanche `issuance.Carried`) | export protégé ou non |

Le crédit « moteur Nicomaque, créé par Nicolas Harmand » figure sur la page HTML et sur la
feuille d'appariements.

## 10. Le crédit

Dans la gestion d'un tournoi, un bouton info ouvre une fenêtre : « Nicomaque, moteur de
tournoi créé par Nicolas Harmand », le lien vers le dépôt et vers sa documentation
(GitHub Pages : documents introductifs, étude des formats, spécification du moteur — **dans
les neuf langues de blunderDB**, même chaîne Sphinx + gettext), la version du module embarquée. Le même crédit figure dans l'aide intégrée, sur la page
tournoi du manuel et sur la page À propos.

## 11. Les flux

Un flux = une intention du TD, de bout en bout. Les budgets de gestes sont dans
[ux.md](ux.md) ; ici la règle. Persona dominant entre crochets.

| # | Flux | Règle |
|---|---|---|
| F1 | Créer un tournoi dirigé [Yanis] | nom, date, lieu ; choisir une configuration nommée ou composer les phases ; tables ; la Direction est *en préparation* |
| F2 | Inscrire un joueur [Marc] | nom (autocomplété), club, cote ; Entrée inscrit et enchaîne |
| F3 | Reprendre les inscrits d'un tournoi précédent [Marc] | annuaire filtré par tournoi, sélection, un geste |
| F4 | Importer un CSV d'inscrits [Sophie] | fichier, aperçu, erreurs par ligne, confirmation |
| F5 | Lancer le tournoi [Yanis] | la première proposition (tirage ou première ronde) ; la configuration se fige |
| F6 | Confirmer les propositions [Marc, Léa] | une ou toutes ; les tables sont assignées ; feuille imprimable |
| F7 | Saisir un résultat [Marc, Léa] | choisir le match (par table ou par nom), vainqueur, score ; forfait possible |
| F8 | Corriger un résultat [Sophie] | depuis le match fini ; nouvelle saisie ; réparation proposée si tableau |
| F9 | Annuler un match lancé par erreur [Yanis] | depuis le match en cours ; recalcul |
| F10 | Apparier à la main [Marc] | deux joueurs libres, longueur, table |
| F11 | Changer la table d'un match [Marc] | depuis la grille ou le match |
| F12 | Retirer un joueur [Marc] | immédiat ou différé ; confirmation |
| F13 | Inscrire un retardataire [Marc] | comme F2 ; la vue dit où il entre |
| F14 | Passer à la phase suivante [Yanis] | proposition ; aperçu de ce que la phase reçoit |
| F15 | Changer la configuration en cours [Sophie] | bascule, longueurs, tables, pauses, dotation, phase ajoutée ; refus du reste |
| F16 | Suivre le temps [Sophie] | bande d'horloge ; matchs lents ; fin estimée |
| F17 | Reprendre après une coupure [Léa] | rouvrir la base : la Direction est là où elle était ; la file des propositions est à jour |
| F18 | Reprendre après une absence [Léa] | « depuis votre dernier geste » : événements survenus, propositions en attente |
| F19 | Transcrire un match depuis son Slot [Sophie] | en-tête hérité, Slot réservé, retour au tournoi |
| F20 | Rattacher des matchs importés [Sophie] | liste des non rattachés avec Slot suggéré ; un geste par ligne |
| F21 | Clore et classer [Sophie] | proposition de clôture ; classement par section ; prix ; CSV |
| F22 | Réouvrir pour corriger [Sophie] | événement ; correction ; nouvelle clôture |
| F23 | Afficher au mur [Marc] | dossier de la page HTML ; ouverture dans le navigateur |
| F24 | Imprimer les appariements [Sophie] | feuille d'une ronde ou d'un tour |
| F25 | Consulter l'historique [Sophie, Léa] | événements filtrés par joueur ou match |
| F26 | Supprimer la Direction [Marc] | confirmation ; les Matchs gardent leur tournoi |
| F27 | Lire le crédit [tous] | bouton info |
| F28 | Corriger une erreur de saisie [tous] | le dernier résultat reste affiché avec [Corriger] ; `Ctrl+Z` ouvre la correction du dernier événement ; un Participant se corrige (nom, club, cote) sans changer de slug |

## 12. Hors périmètre

- L'écran des joueurs (téléphone, route serveur) et toute écriture depuis le front web (D4,
  ADR-0039) : à décider après un vrai tournoi, dans un ADR propre.
- Une fiche « personne » réutilisable entre tournois : l'annuaire dérivé y pourvoit (D3).
- La validation des règles de classement par une fédération (D14).
- Un départage : Nicomaque n'en a pas par principe (ex æquo ou barrage).
- La console interactive en ligne de commande : `tournoi-td` existe dans Nicomaque.
- Le paiement, la comptabilité, l'inscription en ligne.

## 13. Hypothèses prises seul, à confirmer

1. **`config_changed` porte la configuration entière** (pas un diff) : le moteur compare et
   refuse ce qui change une phase commencée.
2. **Deux façons de désigner un match à la saisie du résultat** : par sa table (le plus
   court) ou par un nom de joueur.
3. **Une proposition ignorée revient** à l'appel suivant, identique (déterminisme) ; il n'y
   a pas de « refuser une proposition » — on apparie à la main à la place.
4. **Le slug du Participant** est dérivé du nom ; deux homonymes reçoivent un suffixe
   numérique visible et le club les distingue à l'écran.
5. **Ouvrir une autre Direction ferme la première** sans confirmation (rien n'est en
   attente d'écriture).
6. **La page HTML est réécrite à chaque événement** dans un dossier choisi une fois par
   Direction, et le chemin est mémorisé avec le Tournament.
7. **La feuille d'appariements** et la page HTML sont produites par `render/` de Nicomaque,
   habillé par la charte de blunderDB (CSS embarqué), traduit via les codes (D9).
8. **Le rattachement suggéré** exige l'égalité des deux noms *et* le même Tournament ; une
   égalité partielle n'est pas suggérée.
9. **La cote d'entrée pré-remplie** est le PR du Player sur la base entière (badge existant),
   arrondi à une décimale.
