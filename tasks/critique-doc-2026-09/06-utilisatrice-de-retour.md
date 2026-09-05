# Persona 6 — Utilisatrice de retour après mise à jour

## Qui je suis (3 lignes)

Sophie, 41 ans, joueuse de tournoi. J'utilise blunderDB depuis la 0.31 (août 2026), deux soirs par semaine,
pour revoir mes matchs de club et de tournoi. Je viens d'installer la 0.36.0 et je retrouve une application
qui a bougé : le panneau « Bearoff » s'appelle « Eval », mes matchs se rangent tout seuls dans des tournois,
des touches ne font plus tout à fait la même chose. Je ne veux pas relire un manuel de 85 000 signes : je
veux la liste de ce qui a changé entre ma version et celle-ci, et une doc qui décrive l'écran que j'ai
devant moi.

## Parcours suivi (avec le nombre de clics pour atteindre « ce qui a changé »)

1. J'ouvre l'application, je tape `?`. Quatre onglets : **Manuel, Raccourcis, Commandes, À propos**.
2. Je vais droit à « À propos », l'endroit où les logiciels mettent leurs nouveautés. J'y lis
   « Version de l'application : … » et « Version de la base de données : … », puis l'auteur, la licence,
   les remerciements, les crédits. Aucune liste de changements. Les trois seuls liens de toute l'aide
   intégrée vont vers Discord, vers `github.com/kevung/gammonNet` et vers `THIRD_PARTY.md` — **aucun lien
   vers le site de documentation, aucun vers l'historique des versions**.
3. Je parcours les trois autres onglets par acquit de conscience. Rien non plus.
   → **Depuis l'application : zéro chemin. Le nombre de clics n'existe pas, il faut sortir du logiciel.**
4. J'ouvre le navigateur sur la page produit. Quatre liens en bas : « Documentation Discord Releases
   GitHub ». « Releases » = 1 clic, mais j'atterris sur GitHub, en anglais, dans un format de dépôt.
5. Je prends « Documentation » (1er clic) → l'accueil doc. Dans la barre latérale, seize entrées ; sous
   « Annexes », avant-dernière : « Historique des versions » (2e clic).
   → **Depuis le site : 2 clics**, à condition de deviner que l'historique est une annexe.
6. Bonne surprise : la barre latérale déplie toutes les versions avec leurs dates, de « 0.36.0 (2026-09-05) »
   jusqu'à « 0.1.0 (2024-12-31) ». Je vois immédiatement que j'ai quatre versions de retard.
7. Je lis 0.36 → 0.32, puis je vérifie une dizaine d'annonces dans le manuel, la page Raccourcis, la page
   Liste des commandes et la page CLI.

## Ce que j'ai trouvé en cinq minutes

- Le repérage par version est bon : la barre latérale sert de sommaire daté, chaque version tient en huit à
  dix puces courtes, et la plupart se terminent par un renvoi (« Voir Panneau Stats et Interface en ligne de
  commande (CLI). »). C'est le point fort de la page.
- Le renommage qui m'inquiétait est expliqué : « Le panneau Bearoff devient le panneau Eval : il présente
  les faits de la position … puis la seule décision que le plateau demande ».
- Le rangement automatique de mes matchs aussi : « Les tournois se remplissent d'eux-mêmes à l'import. Les
  fichiers XG, GnuBG et BGF nomment leur événement ; à l'import d'un match nouveau, blunderDB le classe dans
  le tournoi de ce nom » — et le manuel le dit avec la même précision, y compris « Un match déjà présent dans
  la base n'est jamais reclassé ».
- La page CLI est à jour de bout en bout : `search --query`, `--query-help`, `repair`, `completion`,
  `anki card`, `anki log`, `vacuum`, `analyze`, `epc`. Rien annoncé en 0.35/0.36 ne m'y a manqué.
- En revanche je n'ai trouvé **nulle part** une page « après une mise à jour » : ce que je dois refaire
  (réimporter mes .xg pour récupérer la chance, relancer un rattrapage d'analyse), ce que la migration de
  schéma me coûte, ce qui a disparu depuis ma version.

## Dix changements confrontés au manuel

| Changement annoncé | Version | Page attendue | Verdict |
| --- | --- | --- | --- |
| « Le panneau EPC devient le panneau Bearoff » puis « Le panneau Bearoff devient le panneau Eval » | 0.32 / 0.34 | Manuel › Panneau Eval | Nommé autrement — le manuel dit « Eval » partout, sauf un résidu : « Bearoff — la base de sortie two-sided étendue utilisée par le **panneau Bearoff** » |
| « Mode défi pour l'entraînement : les résultats se masquent à chaque modification » | 0.32 | Manuel › Panneau Eval | Concorde — « Mode défi. La case Défi, dans la bande de badges, active un mode entraînement » |
| « Le panneau Stats gagne un onglet Joueurs » | 0.33 | Manuel › Onglet Joueurs | Concorde — section dédiée, colonnes détaillées, règle d'agrégation |
| « La chance de chaque lancer est désormais conservée en base … Elle n'est pas rétroactive » | 0.33 | Manuel › Onglet Joueurs | Concorde — « la colonne Chance pour tout match importé avant la version 2.15.0 du schéma … il faut réimporter les fichiers source » |
| « Nouvelle commande Compacter la base … sous-commande blunderdb vacuum » | 0.33 | Manuel › Configuration + CLI | Concorde — décrit des deux côtés, avec la contrainte d'espace disque |
| « les équités de match sont normalisées et le verdict "trop bon pour doubler" redevient visible » | 0.34 | Manuel › Panneau Eval | Concorde — « "Équité (money)" en money game … trop bon pour doubler » |
| « la bibliothèque est paginée (une base de 50 000 positions ne charge plus tout en mémoire) » | 0.35 | Manuel › Navigation dans les positions | Absent — aucune occurrence de « pagin » dans le manuel |
| « Les tournois se remplissent à l'import » | 0.36 | Manuel › Panneau Tournois | Concorde sur le site ; **absent de l'aide intégrée**, qui dit seulement « Ouvrez le panneau Tournoi avec Ctrl+Y pour gérer les tournois et leur affecter des matchs. » |
| « les tables de sortie ne sont plus embarquées ni téléchargées, mais calculées à la première ouverture » | 0.36 | Manuel › Configuration › onglet Bearoff | Concorde dans la section Configuration, **contredit** deux fois ailleurs (voir constats 2 et 4) |
| « SHIFT-J et SHIFT-K changent de vue depuis n'importe quel panneau » | 0.36 | Raccourcis › Onglets de vues | Nommé autrement et incomplet — la page dit « CTRL-PageUp, MAJ-J Vue précédente. » : ni « SHIFT », ni la précision « depuis n'importe quel panneau » |

## Aide intégrée contre site (écarts relevés)

- **Onglets Raccourcis et Commandes : identiques au site, ligne pour ligne.** J'ai comparé les deux tableaux
  entiers, rien ne diverge. C'est exactement ce que je voulais.
- **Onglet Manuel : ce n'est pas le manuel.** L'historique 0.36 annonce « L'aide intégrée est engendrée
  depuis le manuel : elle ne peut plus prendre de retard sur lui ». Or l'onglet Manuel de l'application est
  un résumé d'environ 130 paragraphes là où le manuel du site en fait dix fois plus — et il a bel et bien
  pris du retard :
  - il dit « base étendue **téléchargeable** jusqu'à 11 pions via l'onglet Bearoff de la configuration »,
    quand le manuel dit « Elles ne sont ni embarquées dans l'exécutable, ni téléchargées : blunderDB les
    calcule sur la machine qui s'en sert » ;
  - il présente le panneau Eval comme « calculer les valeurs d'EPC pour les positions de sortie (panneau
    Eval) » dans la liste des panneaux — le vocabulaire d'avant 0.34 ;
  - rien sur le remplissage automatique des tournois, rien sur la distinction équité money / équité match,
    rien sur l'évaluation progressive 0-ply puis 2-ply.
- **Contradiction interne à l'aide elle-même** : l'onglet Manuel note la touche Anki 3 « **3 (Correct)** »,
  l'onglet Raccourcis, deux clics plus loin, « 3 Évaluer : **Bien**. », et le manuel du site « 3 (Bien) ».
- **L'aide ne documente pas ses propres touches.** J'ai découvert par hasard que `h`/`l` changent d'onglet et
  `j`/`k` font défiler dans la fenêtre d'aide : aucune section « Panneau d'aide » dans Raccourcis, ni sur le
  site ni dans l'application.

## Où je me suis égarée

- Dans l'onglet « À propos » de l'aide, pendant deux bonnes minutes, à chercher un lien « Nouveautés ». Il
  n'y en a pas ; j'ai cru que je lisais mal.
- Dans la barre latérale du site : j'ai cherché « Historique » sous « Prise en main » puis « Référence »
  avant de le trouver en quinzième position sur seize, sous « Annexes ».
- Dans le manuel, en cherchant « Bearoff » : la liste des onglets de configuration parle encore du « panneau
  Bearoff », j'ai cru pendant un moment que Bearoff et Eval étaient deux panneaux différents et que j'avais
  perdu le premier.
- En lisant l'historique de haut en bas : 0.36 m'apprend que les tables « ne sont plus … téléchargées », puis
  quatre versions plus bas 0.32 m'explique que « La base étendue TS-06-11 … se télécharge depuis le nouvel
  onglet Bearoff de la configuration ». Rien ne signale que la seconde phrase est périmée. J'ai ouvert
  l'onglet Bearoff pour chercher un bouton de téléchargement qui n'existe plus.

## Ce qui a entamé ma confiance

- Une promesse vérifiable et fausse : l'historique affirme que l'aide intégrée « ne peut plus prendre de
  retard » sur le manuel, et la première chose que j'y vérifie — les tables de sortie — est en retard.
- Deux endroits du manuel qui se contredisent sur le même sujet à mille lignes d'écart (« ni téléchargées »
  d'un côté, « TS-06-11 téléchargée » de l'autre) : je ne sais plus laquelle des deux pages décrit ma version.
- Le schéma qui change (« Schéma 2.18.0 : Jacoby et beaver quittent l'identité de la position ») sans un mot
  sur ce que ça implique pour mes fichiers. L'information existe, mais dans une annexe que rien ne me
  désigne : « une base migrée vers un schéma récent ne peut plus être ouverte par des versions plus anciennes
  de blunderDB ». J'aurais aimé le savoir **avant** d'ouvrir ma base, pas après.
- Le vocabulaire des touches qui flotte : « SHIFT-J » dans l'historique, « MAJ-J » dans les raccourcis et le
  manuel, et dans le même tableau de raccourcis « CTRL-SHIFT-I » à côté de « MAJ-J ».

## Ce qui manque

- Un chemin depuis l'application vers ce qui a changé : un lien « Nouveautés de cette version » ou seulement
  un lien vers le site dans l'onglet À propos, à côté du numéro de version.
- Une page « Après une mise à jour » qui rassemble ce qu'un retour de plusieurs versions impose : réimporter
  pour la chance (0.33) et pour les marques XG (0.30), lancer le rattrapage d'analyse (0.34), migration de
  schéma irréversible, et les renommages de panneaux avec leur ancien nom.
- Un index des renommages (EPC → Bearoff → Eval ; « Texte de recherche » → « Commentaire » ; panneau Journal
  retiré) : je cherche par l'ancien nom, c'est le seul que je connaisse.
- Des marques d'obsolescence dans l'historique : une puce périmée par une version ultérieure devrait le dire.
- Sur la page Historique, une phrase d'introduction destinée au lecteur. La seule qui s'y trouve aujourd'hui
  m'explique la maintenance des traductions.

## Constats

| # | Constat | Page › section | Gravité | Proposition |
| --- | --- | --- | --- | --- |
| 1 | Aucun chemin vers « ce qui a changé » depuis l'application : « Version de l'application : … » est suivi de l'auteur et de la licence, et les seuls liens de l'aide vont vers Discord, gammonNet et THIRD_PARTY.md | Aide intégrée › À propos | Bloquant | Ajouter dans le fragment de prose « À propos », sous le numéro de version, deux liens : la documentation en ligne et « Historique des versions » |
| 2 | L'aide intégrée est périmée sur un changement phare de 0.36 : « base étendue téléchargeable jusqu'à 11 pions via l'onglet Bearoff de la configuration » | Aide intégrée › Manuel › Panneau Eval | Bloquant | Réécrire ce passage du fragment de prose sur le modèle du manuel : les tables sont calculées sur la machine, jamais téléchargées |
| 3 | Le manuel se contredit sur le nom du panneau : « Bearoff — la base de sortie two-sided étendue utilisée par le panneau Bearoff » vs « L'onglet Bearoff gère les tables de sortie du panneau Eval » | Manuel › Configuration | Bloquant | Corriger la liste des onglets : « Bearoff — les tables de sortie utilisées par le panneau Eval » |
| 4 | Résidu de téléchargement dans le manuel : « (intégrée TS-06-06, fichier externe, ou TS-06-11 téléchargée) » | Manuel › Méthodologie et hypothèses du panneau Eval | Gênant | Remplacer « téléchargée » par « calculée localement », en cohérence avec la section Configuration |
| 5 | L'historique promet que « L'aide intégrée est engendrée depuis le manuel : elle ne peut plus prendre de retard sur lui », alors que l'onglet Manuel de l'app est un résumé indépendant | Historique › 0.36.0 | Gênant | Reformuler la puce (les raccourcis et la ligne de commande sont engendrés ; l'onglet Manuel est un résumé) et ajouter en tête de cet onglet un renvoi « manuel complet en ligne » |
| 6 | La touche Anki 3 porte deux libellés dans la même fenêtre d'aide : « 3 (Correct) » et « 3 Évaluer : Bien. » | Aide intégrée › Manuel vs Raccourcis | Gênant | Aligner le fragment de prose sur « Bien », terme employé par le manuel et la page Raccourcis |
| 7 | Le remplissage automatique des tournois, titre de la 0.36, est absent de l'aide intégrée : « Ouvrez le panneau Tournoi avec Ctrl+Y pour gérer les tournois et leur affecter des matchs. » | Aide intégrée › Manuel › Tournois | Gênant | Ajouter une phrase reprenant le manuel : l'import classe un match nouveau dans le tournoi que son fichier nomme, et ne reclasse jamais un match déjà rangé |
| 8 | L'aide intégrée décrit encore le panneau Eval comme un calculateur d'EPC : « calculer les valeurs d'EPC pour les positions de sortie (panneau Eval) » | Aide intégrée › Manuel › Description de l'interface | Gênant | Reprendre la formule du manuel : le panneau évalue n'importe quelle position, et se spécialise en EPC sur une position de sortie |
| 9 | L'historique écrit « SHIFT-J et SHIFT-K » là où toute la doc écrit « MAJ-J / MAJ-K » | Historique › 0.36.0 › corrections notables | Gênant | Uniformiser sur « MAJ », la notation des pages Raccourcis et Manuel |
| 10 | Le correctif « SHIFT-J et SHIFT-K changent de vue depuis n'importe quel panneau » n'apparaît pas dans la page Raccourcis, qui dit seulement « CTRL-PageUp, MAJ-J Vue précédente. » | Raccourcis › Onglets de vues | Gênant | Préciser dans le tableau que ces touches changent de vue quel que soit le panneau actif |
| 11 | Aucune marque d'obsolescence : « La base étendue TS-06-11 … se télécharge depuis le nouvel onglet Bearoff » (0.32) reste lisible sans avertissement après la 0.36 | Historique › 0.32.0 | Gênant | Ajouter aux puces périmées une mention brève « (remplacé en 0.36.0) » |
| 12 | Le changement de schéma est annoncé sans sa conséquence : « Schéma 2.18.0 : Jacoby et beaver quittent l'identité de la position » ; l'irréversibilité n'est dite que dans une annexe | Historique › 0.36.0 | Gênant | Renvoyer chaque puce de schéma vers « Annexe: Schéma de la base de données » et y rappeler que la migration est automatique et sans retour |
| 13 | Rien ne dit à une revenante ce qu'elle doit refaire : réimporter pour la chance, réimporter pour les marques, lancer le rattrapage d'analyse — chaque version le dit isolément | Site (page inexistante) | Gênant | Créer une courte page « Après une mise à jour », liée depuis l'accueil doc et depuis la tête de l'historique |
| 14 | « Historique des versions » est rangé en Annexes, avant-dernière entrée sur seize | Accueil doc › barre latérale | Gênant | Remonter l'entrée sous « Prise en main » ou la citer dans le corps de l'accueil |
| 15 | La seule phrase d'introduction de la page Historique parle de traduction : « pour qu'une correction ultérieure … n'invalide qu'une puce et non la traduction de toute la version » | Historique › introduction | Mineur | Remplacer par un mode d'emploi du lecteur (versions les plus récentes en tête, renvois en fin de version), et déplacer la note de maintenance hors de la page publiée |
| 16 | La fenêtre d'aide se pilote au clavier (h/l pour les onglets, j/k pour défiler, ESPACE pour la page suivante) sans que ce soit écrit nulle part | Raccourcis (section absente) | Mineur | Ajouter une section « Panneau d'aide » au tableau des raccourcis ; elle sera reprise telle quelle dans l'aide intégrée |
| 17 | La commande d'ouverture du panneau s'appelle toujours `epc` : « epc Ouvre le panneau Eval » — aucune trace de l'ancien nom ni d'un synonyme | Liste des commandes › Opérations globales | Mineur | Mentionner dans la ligne que `epc` est l'ancien nom conservé, et ajouter une note d'index « Bearoff → voir Panneau Eval » dans le manuel |
| 18 | Le fait 0.35 « la bibliothèque est paginée » n'a aucune contrepartie dans le manuel | Manuel › Navigation dans les positions | Mineur | Une phrase sur le chargement par pages sur les grosses bases, ou retirer la promesse de l'historique si elle est sans surface visible |
| 19 | Notations mélangées dans un même tableau : « CTRL-SHIFT-I Importer une base de données. » et « CTRL-PageUp, MAJ-J Vue précédente. » | Raccourcis › Base de données / Onglets de vues | Mineur | Choisir MAJ partout dans la version française |
| 20 | La page produit propose « Documentation Discord Releases GitHub » : rien ne se nomme « Nouveautés », et « Releases » sort du site vers un dépôt en anglais | Page produit › pied de page | Mineur | Ajouter un lien « Historique des versions » pointant la page française, à côté de « Documentation » |
