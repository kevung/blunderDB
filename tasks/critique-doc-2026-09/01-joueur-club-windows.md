# Persona 1 — Joueur de club sous Windows, venant de XG

## Qui je suis (3 lignes)

Marc, 52 ans, joueur de club depuis longtemps, PR autour de 8, sous Windows 11.
J'analyse mes matchs avec eXtreme Gammon 2 depuis dix ans ; je n'ai jamais ouvert de terminal
et je ne sais pas ce qu'est GitHub. Un copain du club m'a envoyé le lien. Je lis le français.

## Parcours suivi (liste ordonnée des pages/sections lues, avec ce qui m'a fait cliquer)

1. **Page produit (racine du site)** — arrivée par le lien du copain. Tout est en anglais.
   Je reconnais « eXtreme Gammon » dans la première phrase, ça suffit à me faire rester.
   La copie d'écran ressemble à XG : plateau, liste de coups, colonnes EQUITY / ERROR. Rassurant.
2. **Bouton « Windows (.exe) »** — je clique tout de suite, le téléchargement démarre.
   Pendant ce temps je clique sur **« Documentation »**.
3. **Accueil de la documentation (en français, ouf)** — je lis le paragraphe d'introduction
   et la phrase « Nouveau venu ? Le guide utilisateur propose quatre tutoriels de bout en bout ».
   Je note « Guide utilisateur » pour plus tard.
4. **Retour au bureau** : je double-clique sur le .exe téléchargé. Windows affiche un écran bleu
   « Windows a protégé votre ordinateur ». Je referme. Peur.
5. **Recherche dans le sommaire de la doc** d'une explication. Je tombe sur
   « Annexe Windows : Détection abusive de blunderDB comme logiciel malveillant ». Je lis toute la page.
6. **Téléchargement et installation** — j'y vais après coup, pour vérifier que j'ai pris le bon fichier.
   Je découvre en bas de page la section « Avertissements Windows et Mac ».
7. **Guide utilisateur › Tutoriels de bout en bout › Mon premier import** — cinq étapes, je les suis.
8. **Glossaire** — parce que je bute sur « PR », « coût MWC », « EMG », « équité normalisée ».
9. **Foire aux questions** — pour comprendre ce que ça m'apporte par rapport à XG que j'ai déjà.

## Ce que j'ai trouvé en cinq minutes

Au bout d'environ cinq minutes de lecture (page produit, accueil de la doc, annexe Windows en entier),
j'en suis là : **le logiciel est téléchargé, mais pas encore lancé.** Je sais qu'il sert à ranger et
à retrouver des positions, qu'il lit mes fichiers XG, qu'il est gratuit. Je sais aussi que Windows va
râler et pourquoi (« Partageant blunderDB gratuitement, je ne souhaite pas m'orienter vers ces
possibilités onéreuses »). Mais je n'ai pas encore vu un seul plateau et je n'ai pas encore lu la
moindre ligne du tutoriel d'import. Mes cinq minutes sont passées à me rassurer, pas à découvrir l'outil.

Le premier import, lui, arrive vers la douzième minute : le tutoriel « Mon premier import » est court,
clair et bien fait — glisser-déposer le fichier .xg sur la fenêtre, c'est exactement ce que j'espérais.
Trois pages et six ou sept clics auraient suffi si l'épisode Windows n'avait pas mangé la moitié du temps.

## Où je me suis égaré

- **Entre le bouton de téléchargement et l'avertissement Windows.** J'ai cliqué sur « Windows (.exe) »
  sur la page produit : rien, à cet endroit, ne me prévient. Sur la page d'installation non plus, les
  liens sont en haut et l'avertissement tout en bas. Je découvre le problème par l'écran bleu de Windows,
  jamais par la documentation.
- **Dans le tableau « Quel fichier choisir ? »** : huit lignes de Linux avec des commandes
  (`sudo apt install`, `yay -S blunderdb-bin`, `flatpak install`) avant d'arriver à ma ligne, l'avant-dernière.
  J'ai cru un instant que ce logiciel n'était pas fait pour moi.
- **À la dernière étape du tutoriel d'import**, celle qui m'intéressait le plus : « la commande bl
  (ou blunders, ESPACE pour ouvrir la ligne de commande) ». Une ligne de commande dans le logiciel :
  c'est là que j'ai failli refermer. J'ai appris plus tard, dans le manuel, qu'un tableau
  « Top blunders » cliquable existe dans le panneau Stats — le tutoriel ne le dit pas.
- **Dans le glossaire**, cherché à la main dans le sommaire : aucune page du guide ne m'y renvoie.

## Ce qui a entamé ma confiance

- L'écran **« Windows a protégé votre ordinateur »** rencontré *avant* toute explication. Après coup,
  l'annexe est honnête et me rassure à moitié — mais l'ordre est le mauvais.
- Le titre de cette annexe, tel qu'il apparaît dans le sommaire de gauche :
  « Annexe Windows : Détection abusive de blunderDB comme logiciel malveillant ». En balayant, je ne
  lis que « logiciel malveillant ».
- La procédure proposée ne s'arrête pas au simple « Exécuter quand même » : elle enchaîne sur
  « Cliquez sur Ajouter une exclusion et sélectionnez Fichier », c'est-à-dire désactiver mon antivirus
  sur un fichier. **Ça, je ne le ferai pas**, et rien ne me dit que c'est un dernier recours rare.
- Les copies d'écran de cette page sont en anglais (le fichier s'appelle `smartscreen_en.png`) alors
  que mon Windows est en français : je ne suis pas sûr de reconnaître les boutons.
- Les vérifications proposées (« Get-FileHash », « gh attestation verify ») sont des lignes à taper
  dans une fenêtre noire. Donc, pour moi, la garantie annoncée n'existe pas.

## Ce qui manque

**Ce que je n'ai pas trouvé** (mais qui existe ailleurs dans la doc) : la case « analyser
automatiquement après import », qui est dans le manuel ; le tableau cliquable « Top blunders » du
panneau Stats, qui est dans le manuel ; le fait que le logiciel calcule des tables au premier
lancement (« environ six secondes sur un cœur »), qui est dans le manuel ; les définitions de PR et
de MWC, qui sont dans le glossaire.

**Ce qui n'existe nulle part**, à ma connaissance après lecture :
- une page produit en français ;
- une phrase disant que mon match .xg doit **déjà avoir été analysé dans XG** pour que les erreurs
  apparaissent (et ce qu'il faut faire sinon) ;
- une réponse à ma vraie question : « j'ai déjà XG, qu'est-ce que blunderDB m'apporte en plus ? »
  La FAQ pose la question à l'envers (« Ai-je besoin d'eXtreme Gammon pour utiliser blunderDB? ») ;
- une définition d'« EMG », mot que le manuel emploie pour définir un blunder ;
- une indication de la configuration requise (Windows 10/11 n'est dit que dans l'annexe) et de la
  taille du téléchargement.

## Constats

| # | Constat | Page › section | Gravité | Proposition |
|---|---|---|---|---|
| 1 | Le premier écran du site est en anglais seulement : « A backgammon blunder analysis tool: import your matches from eXtreme Gammon, GnuBG and BGBlitz ». Aucun sélecteur de langue visible, rien ne dit que la doc existe en neuf langues. | Page produit › en-tête | bloquant | Traduire la page produit en français (au minimum le paragraphe d'accroche et les boutons), et y placer le même sélecteur de langue que la doc. |
| 2 | Aucun avertissement Windows avant le téléchargement : ni sur la page produit sous « Windows (.exe) », ni à côté des liens « pour Windows: …blunderDB-windows-0.36.0.exe ». La section « Avertissements Windows et Mac » est la dernière de la page d'installation. | Page produit › boutons ; Téléchargement et installation › en-tête vs. § final | bloquant | Mettre une phrase courte immédiatement sous le bouton et sous le lien Windows : « Windows affichera un avertissement au premier lancement : c'est normal, voici pourquoi et quoi faire. » |
| 3 | « Mon premier import » ne dit nulle part que le fichier .xg doit avoir été analysé dans XG au préalable. Si j'importe un match brut, je ne vois aucune erreur et je conclus que le logiciel ne marche pas. | Guide utilisateur › Mon premier import (étapes 2 à 5) | bloquant | Ajouter une note à l'étape 2 : ce que blunderDB reprend du fichier XG (analyse existante), ce qu'il fait s'il n'y en a pas, et où activer « analyser automatiquement après import ». |
| 4 | La dernière étape du tutoriel, la plus attendue, n'existe qu'en ligne de commande : « la commande bl (ou blunders, ESPACE pour ouvrir la ligne de commande) ». Le manuel décrit pourtant un « Top blunders » cliquable : « Cliquer sur une ligne charge la position concernée dans le panneau d'analyse. » | Guide utilisateur › Mon premier import (étape 5) vs. Manuel › Panneau Stats › Top blunders | bloquant | Donner d'abord le geste à la souris (panneau Stats → Top blunders) et la commande ensuite, en raccourci pour habitués. |
| 5 | La procédure Windows enchaîne SmartScreen et l'exclusion Windows Defender sans hiérarchie : « Cliquez sur Ajouter une exclusion et sélectionnez Fichier ». Désactiver l'antivirus sur un fichier est un geste que je refuse. | Annexe Windows › Blocage Windows Defender | bloquant | Dire explicitement que l'étape 1 (« Informations supplémentaires » → « Exécuter quand même ») suffit dans la quasi-totalité des cas, et présenter l'exclusion Defender comme un dernier recours, avec la mise en garde qui va avec. |
| 6 | Le titre de l'annexe, tel qu'il s'affiche dans le sommaire, met « logiciel malveillant » en évidence : « Annexe Windows : Détection abusive de blunderDB comme logiciel malveillant ». | Sommaire (toutes pages) › Annexes | gênant | Retitrer côté sommaire en formulation rassurante et orientée action, par exemple « Windows bloque le lancement : que faire ». |
| 7 | La documentation française cite les boutons en anglais : « bouton « Save Position » », « bouton « New Deck » », « cliquer sur « Add » », « les onglets At least et Except », « cliquer sur « Save » ». Mon interface est en français. | Guide utilisateur › Créer une base, Ajouter une position, Une session Anki, Rechercher des positions | gênant | Citer les libellés français de l'interface, et n'indiquer l'anglais entre parenthèses que si le bouton n'est pas traduit. |
| 8 | Contradiction sur la suppression d'une position : le guide dit « La suppression de la position est définitive et ne nécessite aucune confirmation de la part de l'utilisateur », les deux pages de référence disent « Supprimer la position courante (confirmation demandée) » et « Supprime la position courante (confirmation demandée) ». | Guide utilisateur › Supprimer une position vs. Raccourcis clavier › Position vs. Liste des commandes › Positions et navigation | gênant | Aligner les trois formulations sur le comportement réel. |
| 9 | « PR » et « coût MWC » arrivent sans un mot d'explication : « Le panneau des matchs : liste triable, PR et coût MWC par match. » Le glossaire les définit, mais aucun texte du guide ni du manuel ne renvoie vers lui : on n'y arrive que par le sommaire. | Guide utilisateur › Étudier un match ; Glossaire | gênant | Lier chaque sigle vers l'entrée du glossaire à sa première apparition dans le guide, et citer le glossaire dans le paragraphe d'accueil « Nouveau venu ? ». |
| 10 | « EMG » sert à définir ce qu'est une erreur grave — « Blunders  Nombre d'erreurs graves (au moins 0,100 EMG) » — mais le mot n'est expliqué nulle part, ni sur place ni au glossaire. | Manuel › Panneau Stats › colonnes | gênant | Ajouter une entrée « EMG » au glossaire et développer le sigle à sa première occurrence dans le manuel. |
| 11 | Le tableau « Quel fichier choisir ? » range Windows en avant-dernière ligne, après huit lignes Linux pleines de commandes. Un utilisateur Windows croit d'abord que l'outil n'est pas pour lui. | Téléchargement et installation › Quel fichier choisir ? | gênant | Placer Windows et macOS en tête du tableau, ou grouper les lignes Linux sous une seule ligne dépliable. |
| 12 | La page produit et l'accueil de la doc ne décrivent pas le même logiciel. Produit : « import your matches …, store every position once, and search them by structure and by mistake » plus un évaluateur intégré. Accueil doc : « constituer des bases de données de positions de backgammon … catalogues de positions de référence » — pas un mot sur l'import de matchs, ni sur l'évaluateur, ni sur les statistiques. | Page produit › accroche vs. Accueil de la documentation › paragraphe d'ouverture | gênant | Reprendre en accueil de doc la même accroche que la page produit, puis développer. |
| 13 | Aucune page ne répond à la question que se pose un utilisateur d'XG : « qu'est-ce que ça m'apporte de plus qu'XG ? » La FAQ pose l'inverse (« Ai-je besoin d'eXtreme Gammon pour utiliser blunderDB? »). | Foire aux questions | gênant | Ajouter une entrée « J'utilise déjà eXtreme Gammon : à quoi me sert blunderDB ? » avec trois différences concrètes (agrégation multi-matchs, recherche par structure, révision espacée). |
| 14 | Les copies d'écran de l'annexe Windows sont en anglais (`smartscreen_en.png`) alors que la page est en français et que mon Windows l'est aussi. | Annexe Windows › Avertissement Windows SmartScreen | gênant | Refaire les captures sur un Windows en français, ou à défaut citer entre guillemets les libellés français à chercher. |
| 15 | Les moyens de vérifier le fichier téléchargé passent tous par une fenêtre de commandes : « Get-FileHash .\blunderDB-windows-x.y.z.exe -Algorithm SHA256 », « gh attestation verify ». Pour moi, la garantie annoncée sur la page produit (« Every asset is checksummed and attestable ») n'est pas utilisable. | Téléchargement et installation › Vérifier un téléchargement | mineur | Indiquer en une phrase que cette vérification est facultative et destinée aux utilisateurs avancés, pour ne pas laisser croire qu'elle est une étape obligatoire de l'installation. |
| 16 | Contradiction sur les tables de bearoff : la FAQ dit « utilise la base de données de bearoff à 6 points de GNUbg » et « base téléchargeable jusqu'à 11 », le manuel dit « Elles ne sont ni embarquées dans l'exécutable, ni téléchargées : blunderDB les calcule sur la machine qui s'en sert ». | Foire aux questions › Qu'est-ce que l'EPC ? vs. Manuel › Configuration › onglet Bearoff | mineur | Corriger la FAQ ; ne parler de téléchargement nulle part si rien n'est téléchargé. |
| 17 | Le sigle PR n'a pas le même nom d'une page à l'autre : « PR (Performance Rate) » au glossaire, « Le PR (Performance Rating) » à la FAQ. | Glossaire vs. Foire aux questions › différence PR / Snowie | mineur | Choisir une seule dénomination et l'employer partout. |
| 18 | La page produit annonce « one binary, five modes: desktop app, command line, headless server » : cinq modes annoncés, trois nommés, et « headless » n'est ni traduit ni expliqué à cet endroit. Elle annonce aussi trois formats d'import quand la doc en liste quatre (Jellyfish en plus). | Page produit › accroche | mineur | Sur la page produit, ne parler que de ce qui concerne un joueur (l'application de bureau) et lister les quatre formats ; renvoyer le reste à la doc. |
| 19 | Rien avant le premier lancement ne prévient qu'un calcul démarre tout seul en arrière-plan : « Les deux tables ordinaires … sont calculées au premier lancement, en arrière-plan et sans rien demander : environ six secondes sur un cœur ». C'est dans le manuel, jamais dans la page d'installation. | Manuel › Configuration › onglet Bearoff ; absent de Téléchargement et installation | mineur | Ajouter une phrase « au premier lancement » sur la page d'installation : ce qui se passe, combien de temps, et qu'il n'y a rien à faire. |
| 20 | Le guide dit « Les touches GAUCHE/DROITE (ou j/k) parcourent les coups », la page des raccourcis dit l'inverse dans le détail (« GAUCHE, k » / « DROITE, j »). | Guide utilisateur › Mon premier import vs. Raccourcis clavier › Navigation | mineur | Écrire « (ou k/j) » dans le guide, dans le même ordre que la page des raccourcis. |
