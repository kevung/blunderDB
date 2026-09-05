# Persona 2 — Joueuse confirmée qui compare avec XG

## Qui je suis (3 lignes)

Elena, 38 ans, PR 4,5, joueuse de tournoi anglophone. J'utilise eXtreme Gammon depuis huit ans : rollouts, position database, MET, EMG, décisions de videau au score.
J'ai déjà une base de positions dans XG et je veux savoir si blunderDB la remplace, la complète, ou ne sert à rien.
Je ne fais confiance à un chiffre que si je sais comment il est calculé, avec quelle référence, et mesuré sur quel échantillon.

## Parcours suivi (liste ordonnée des pages/sections lues, avec la question qui m'y a menée)

1. **Page produit** (racine) — « qu'est-ce que ça fait de plus que XG ? ». Trois phrases : import XG/GnuBG/BGBlitz, dédoublonnage, recherche par structure et par erreur, plus « A built-in evaluator (gammonNet) judges any position offline ». Rien sur XG comme concurrent ou complément.
2. **Documentation › accueil (index)** — je cherche une page « comparison » ou « limitations ». Il n'y en a pas ; je vais vers la FAQ.
3. **FAQ › “Do I need eXtreme Gammon to use blunderDB?”** — « puis-je garder XG ? ». Réponse claire, la meilleure de la doc sur ce point.
4. **FAQ › “What is the built-in evaluator (gammonNet) worth?”** — « le moteur vaut-il XG ? ». Réponse honnête sur l'esprit, muette sur les chiffres.
5. **Manuel › Eval Panel › “Methodology and assumptions of the Eval panel”** — « d'où sortent les équités et le verdict ? ». La page la plus sérieuse du site ; c'est là que j'ai trouvé une vraie mesure.
6. **Annex: Statistics model — XG / gnuBG / blunderDB alignment** — « le PR est-il le même chiffre que celui de XG ? ». Formules, sources gnubg, tableau d'écarts.
7. **Manuel › Stats Panel › PR / MWC toggle, Aggregation rule, MWC: limitations** — « quelle MET, quelle échelle ? ».
8. **Glossaire** — « normalised equity, referential, verdict » : bien écrit, mais il manque une entrée.
9. **Guide utilisateur › “My first import” et “Display the analysis of a position imported from XG”** — « que deviennent mes analyses XG à l'import ? ».
10. **Manuel › Configuration › onglet gammonNet** — c'est là, et seulement là, que j'ai fini par trouver la règle d'écrasement.
11. **CLI › `analyze`** — même règle, redite plus clairement ; et des commentaires d'exemples en français.
12. **À propos › Credits** — « d'où vient le réseau, d'où viennent les tables ? ».
13. **Historique 0.36.0** — pour dater ce que je viens de lire.

## Ce que j'ai trouvé en cinq minutes

Beaucoup, et de bonne qualité, ce qui rend les trous plus frustrants.

- Que blunderDB ne remplace pas XG et ne prétend pas le faire : « An XG import does, however, bring the most complete analysis (plays, cube decisions, flags, roll luck): it is the format the statistics draw on most richly. » (FAQ). C'est net.
- Que mes analyses XG ne sont **pas** écrasées : « A position that also carries an XG, GNUbg or BGBlitz analysis is never touched by this button, whatever its gammonNet content » et « the rule is “an evaluation only fills a hole”, never a replacement » (Manuel › Configuration). Et « When multiple analysis engines are present for the same position (for example XG and GNUbg), an additional column indicates the source engine of each analysis. » (Guide).
- Que le PR est défini par une formule, avec le fichier source de gnubg en référence : « cf. gnubg/formatgs.c:399–409 ». Ça, aucun autre logiciel de ma bibliothèque ne le fait.
- Que l'échelle d'équité au score est nommée : « in money game it is expressed in points, at a match score in normalised equity — the same scale as XG and GNU Backgammon, where winning the value of the current cube is worth +1 and losing it −1 ».
- Qu'il y a **une** mesure chiffrée du moteur, et qu'elle est honnête jusqu'à l'inconfort : « money verdict agreement 93.4% (3735/4000) […] 61.1% within 1% of the take point ». 61 % d'accord au point de prise, c'est exactement le chiffre qu'un vendeur cacherait.
- Que la règle d'agrégation du PR est la bonne (somme/somme), avec un contre-exemple numérique.

## Où je me suis égarée

- **Trente secondes perdues à chercher une page « blunderDB vs XG »**. Elle n'existe pas ; la réponse est éclatée entre deux entrées de FAQ, l'annexe statistique et deux paragraphes du manuel. Le sommaire ne m'y conduit pas.
- **La règle d'import est rangée sous « Configuration »**. Ma question — « mes analyses XG survivent-elles ? » — est une question d'import, pas de réglages. Le manuel n'a aucune section « Import » ; j'ai lu tout le guide utilisateur avant d'y arriver.
- **Le mot « rollout »**. Je l'ai cherché sur toutes les pages anglaises. Zéro occurrence. Je ne sais toujours pas si un rollout XG stocké dans mon `.xg` est importé comme analyse, ni s'il est distingué d'une évaluation 4-ply.
- **Deux définitions du millipoint**. J'ai lu `E>40` dans les filtres, `0.100 EMG (100 millipoints)` dans l'annexe, et `× 500` dans la formule du PR. J'ai dû faire l'arithmétique moi-même pour comprendre que les filtres sont en millièmes d'équité et le PR en demi-millièmes.

## Ce qui a entamé ma confiance

Une phrase, surtout : **« blunderDB never computes luck: it has no evaluation engine. »** (annexe statistique). Or la page produit m'a vendu un moteur intégré, la FAQ lui consacre une entrée, et le manuel décrit une recherche 2-ply avec modèle Janowski. Les deux affirmations sont vraies dans leur contexte (la chance est lue dans le fichier source, jamais recalculée), mais je les ai lues comme une contradiction frontale. Quand une doc se contredit sur « ai-je un moteur ou non », je relis tout le reste en me demandant ce qui est à jour.

Deuxième coup : **la seule mesure publiée du moteur est money-only**. « TestEvalMeasure, 4000 sampled money decisions » — et le même paragraphe m'annonce que le régime évalué est le seul, avec l'exact, à « allow […] a verdict at the match score ». Donc la fonctionnalité que je voulais tester (verdict de videau à 2-away/4-away) est précisément celle qui n'est chiffrée nulle part.

Troisième : **les crédits contredisent le manuel** sur l'origine des tables de bearoff.

Quatrième, plus vénielle mais visible : les exemples de la page CLI anglaise sont commentés en français (`# Décisions de videau, 30 pips de retard, 50 millipoints d'erreur`). Sur une page qui me demande de faire confiance à des chiffres, une page à moitié traduite donne l'impression d'une doc en chantier.

## Ce qui manque

Je distingue ce que **je n'ai pas trouvé** (ça existe peut-être, mal rangé) de ce qui **n'existe pas dans la doc**.

Je n'ai pas trouvé : une page de comparaison fonctionnelle avec la position database de XG ; le sort des rollouts importés ; le domaine de la MET du moteur (jusqu'à combien de points ?) ; les seuils d'erreur de XG en regard du seuil 0,100 de blunderDB.

N'existe pas, à ma lecture : toute mesure du moteur **hors course** (positions de contact) ; toute mesure du moteur **au score** ; toute mention du mot rollout ; toute entrée de glossaire pour EMG ; l'identité des trois matchs de référence de l'annexe statistique.

## Constats

| # | Constat | Page › section | Gravité | Proposition |
|---|---|---|---|---|
| 1 | « blunderDB never computes luck: it has no evaluation engine. » contredit frontalement la page produit (« A built-in evaluator (gammonNet) judges any position offline ») et la FAQ. | Annex: Statistics model › Luck (encart Important) | bloquant | Reformuler : « blunderDB never *recomputes* luck: the embedded evaluator is not used for it. It takes verbatim… », et renvoyer à la FAQ du moteur. |
| 2 | Le moteur n'est comparé à XG nulle part sur des positions de contact. La FAQ botte en touche : « It is neither the only nor necessarily the best engine on the market: it is the one that works offline ». La seule mesure du site porte sur le bearoff. | FAQ › What is the built-in evaluator worth? ; Manuel › Methodology | bloquant | Ajouter à la FAQ un paragraphe « what has been measured, and what has not », avec renvoi explicite : mesuré en course money, non mesuré hors course et au score. |
| 3 | Aucune mesure du verdict de videau **au score**, alors que le manuel présente cette capacité comme distinctive : « available even at a match score, which the estimated regime could never offer ». Les 93,4 % cités sont « 4000 sampled money decisions ». | Manuel › Methodology › Winning probability and verdict, evaluated regime | bloquant | Dire en une phrase que la mesure est money-only et que le verdict au score n'a pas de mesure publiée ; sinon un lecteur transporte les 93,4 % au score. |
| 4 | Le mot « rollout » n'apparaît sur aucune page anglaise. Un utilisateur XG ne sait ni si blunderDB en fait, ni ce que devient un rollout stocké dans son `.xg`. | Toute la doc (FAQ, Manuel › Analysis Panel) | bloquant | Une entrée FAQ « Are XG rollouts imported? Does blunderDB roll out? » ; et préciser dans Analysis Panel ce que couvre « the level of analysis ». |
| 5 | Deux échelles portent le même nom. Annexe : « The factor 500 converts equity to millipoints » ; puis « 0.100 EMG (100 millipoints) » ; filtres : « Too good to double … s d e>1000 ». | Annex › PR, Errors and blunders ; Annexe filtres | bloquant | Nommer les deux échelles distinctement (mpt PR ×500 vs millièmes d'équité ×1000) et le rappeler dans la ligne des filtres. |
| 6 | « PR (Performance Rate) — Measures money-game play quality: sum of errors in milli-points, divided by the number of decisions. Independent of match score. » contredit l'annexe (erreurs EMG *cubeful* au score, conversion MWC par MET). | Manuel › Stats Panel › PR / MWC toggle | bloquant | Supprimer « money-game » et « Independent of match score », renvoyer à la formule de l'annexe. |
| 7 | Crédits périmés : « the one-sided … and two-sided … bearoff databases were generated with GNU Backgammon » alors que le manuel dit « blunderDB computes them itself, identically to GNUbg's makebearoff tool — byte for byte » et que 0.36.0 annonce qu'elles ne sont plus embarquées. | À propos › Credits | gênant | Réécrire le point : tables générées par blunderDB, algorithme porté de makebearoff, empreinte vérifiée contre gnubg. |
| 8 | La parité XG repose sur un échantillon anonyme : « Metrics are aligned within the following bounds (measured on 3 reference matches) », et « Typical gap » n'est pas défini (écart moyen ? maximum observé ?). | Annex › blunderDB ↔ XG ↔ gnuBG correspondence | gênant | Dire combien de décisions au total, quelle version de XG, et si « typical gap » est un maximum observé. |
| 9 | Le glossaire n'a pas d'entrée **EMG**, terme employé six fois dans l'annexe et dans le manuel (« at least 0.100 EMG »). Il a « Normalised equity », qui est la même chose sans le dire. | Glossaire | gênant | Ajouter EMG comme alias de « normalised equity » (et signaler l'usage XG du sigle). |
| 10 | La FAQ dit le PR « aligned with eXtreme Gammon's and GNUbg's conventions », l'annexe dit du seuil de blunder « This threshold is blunderDB's own ». Les deux sont vraies mais se lisent comme un démenti. | FAQ › PR vs Snowie ; Annex › Errors and blunders | gênant | Préciser dans la FAQ : conventions de comptage alignées, seuil de gravité propre. |
| 11 | Trois désignations pour la MET du moteur : « blunderDB's match equity table » (FAQ), « the Kazaross-XG2 match equity table » (cmd_mode, Stats), « the search, cube model and match equity table around it are gammonNet's own configuration » (Credits). | FAQ ; Liste des commandes › met ; Manuel › MWC ; À propos | gênant | Une phrase unique : quelle MET, utilisée par quoi (stats, moteur, ou les deux). |
| 12 | Le domaine de la MET n'est jamais chiffré, alors que deux pages parlent de son bord : « a match score beyond its table's range » / « a score beyond the horizon of the match equity table ». | Manuel › Configuration ; Manuel › Eval Panel | gênant | Donner la longueur maximale couverte (jusqu'à N-away) ; sinon on ne sait pas si un 25 points sort du domaine. |
| 13 | La règle d'écrasement à l'import — la question numéro un d'un utilisateur XG — n'est énoncée que sous Configuration › onglet gammonNet et dans la page CLI. Le manuel n'a aucune section Import. | Manuel (sommaire) ; Manuel › Configuration | gênant | Créer une courte section « Import : ce qui est écrit, ce qui ne l'est jamais » dans le manuel, ou l'ajouter au guide, tutoriel « My first import ». |
| 14 | Fusion de bases décrite en termes vagues, en rupture avec la précision du reste : « blunderDB will automatically merge the positions » / « intelligently merges analyses and comments ». | FAQ › How to merge multiple databases? | gênant | Remplacer « intelligently » par la règle réelle (une analyse par moteur, jamais d'écrasement, commentaires concaténés). |
| 15 | Les exemples de la page CLI anglaise sont commentés en français : « # Vérifier une table reçue d'ailleurs », « # Décisions de videau, 30 pips de retard, 50 millipoints d'erreur », « # Sur un seul cœur, pour laisser la machine à autre chose », et les métavariables restent « --db <chemin> », « <fichier> ». | CLI (nombreuses sections) ; Headless mode | gênant | Traduire les commentaires et métavariables des blocs de code, ou les rendre neutres (`--db <path>`). |
| 16 | La page produit promet « judges **any** position offline », que le manuel dément par deux verdicts d'abstention : « not evaluable at this score » et « the engine refuses the position ». | Page produit ; Manuel › Eval Panel | mineur | Nuancer en « judges any position offline, and says so when it cannot ». |
| 17 | Orthographe incohérente : « normalised » (Glossaire, Manuel, FAQ) vs « the normalized equity » (Guide). | Guide › Display the analysis of a position imported from XG | mineur | Uniformiser sur l'orthographe britannique employée partout ailleurs. |
| 18 | Le barème de PR (« World-class < 3 / Expert 3 – 5 / … ») est donné sans source ni référentiel (longueur de match, échantillon). Un PR 4,5 « Expert » n'est pas une information vérifiable. | Manuel › Stats Panel › PR / MWC toggle | mineur | Citer l'origine du barème, ou l'annoncer explicitement comme un repère indicatif propre à blunderDB. |
| 19 | « byte for byte » et « the SHA-256 fingerprint is checked » sont étayés — bon point — mais la version de gnubg qui sert de référence n'est pas nommée ; une empreinte sans version de référence ne se rejoue pas. | Manuel › Configuration › Bearoff ; CLI › bearoff verify | mineur | Nommer la version de gnubg/makebearoff dont l'empreinte est enregistrée. |
| 20 | L'unité du PR change de nom d'une page à l'autre : « millipoints, mpt » (annexe), « milli-points » (manuel), « mp » (exemple d'agrégation), « thousandths of an equity point » (glossaire). | Annex ; Manuel › Stats ; Glossaire | mineur | Choisir un terme et une abréviation, et les employer partout. |
