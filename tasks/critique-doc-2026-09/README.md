# Critique de la documentation par personas — septembre 2026

Sept lecteurs fictifs, aux attentes qui ne se recouvrent pas, ont lu le site de
documentation de blunderDB **tel qu'il est publié après la refonte de
l'accueil** (commit `792503bca`, 2026-09-06) et l'ont critiqué chacun selon son
critère. Ce dossier est un rapport : **aucune modification de la documentation
n'a été faite dans cette passe**. La synthèse classe les changements proposés
par impact puis par coût ; une branche ultérieure exécute ceux qui sont
retenus.

## Fichiers

| Fichier | Persona | Point de départ | Critère |
|---|---|---|---|
| [01-joueur-club-windows.md](01-joueur-club-windows.md) | Joueur de club sous Windows venant de XG, première visite, non technicien | Racine du site | Temps jusqu'au premier import, alertes Windows |
| [02-joueuse-confirmee-xg.md](02-joueuse-confirmee-xg.md) | Joueuse confirmée qui compare avec sa base XG | Page produit, version anglaise | Confiance : ce que l'outil ne fait pas, parité du PR |
| [03-utilisateur-clavier-cli.md](03-utilisateur-clavier-cli.md) | Utilisateur clavier et CLI qui scripte ses imports | Page CLI | Complétude et cohérence mode commande / raccourcis / CLI |
| [04-administratrice-club-serveur.md](04-administratrice-club-serveur.md) | Administratrice de club qui déploie le mode serveur | Page headless | Avertissement d'absence d'authentification impossible à manquer, Docker suffisant |
| [05-lecteur-japonais.md](05-lecteur-japonais.md) | Lecteur non francophone, navigateur en japonais, puis allemand | Racine du site | Négociation de langue, français résiduel, balisage CJK |
| [06-utilisatrice-de-retour.md](06-utilisatrice-de-retour.md) | Utilisatrice de retour après mise à jour 0.31 → 0.36 | Aide intégrée, puis site | « Ce qui a changé » en un clic, doc conforme à l'écran |
| [07-mainteneur.md](07-mainteneur.md) | Le mainteneur | Sources, catalogues, scripts | Coût par page en neuf langues, engendré contre manuel, dérive à la prochaine release |
| [SYNTHESE.md](SYNTHESE.md) | — | — | Changements consolidés, classés impact puis coût, avec fichiers et effets de bord |

## Méthode

- Le site a été construit localement depuis `main` (`cd doc && python build.py`,
  HTML et PDF, neuf langues, sans avertissement Sphinx), puis chaque page HTML
  a été convertie en texte (`w3m -dump`) : les personas lisent **ce que le
  visiteur voit** — barre latérale, titres, encarts, liens — et jamais les
  `.rst`. Les images sont absentes du texte ; les personas 1 et 5 ont reçu en
  plus des captures Chromium de la page produit, de l'accueil de la doc et de
  la page d'installation (fr, en, ja).
- La page produit (`doc/site/index.html`, racine gh-pages, anglais seul) a été
  rendue avec `__VERSION__` remplacé par `0.36.0`.
- Chaque persona est un agent indépendant avec sa fiche, son point de départ
  imposé et sa trame de rapport commune : parcours suivi, trouvé en cinq
  minutes, où il s'est égaré, ce qui a entamé sa confiance, ce qui manque,
  puis un tableau de constats (page › section, gravité bloquant / gênant /
  mineur, proposition). Sept lectures séparées gardent les contradictions ;
  la synthèse les laisse visibles au lieu de les arbitrer en silence.
- Le persona 6 a lu en plus l'aide intégrée de l'application
  (`frontend/src/i18n/help/fr.js`), engendrée depuis `raccourcis.rst` et
  `cmd_mode.rst` (ADR-0034). Le persona 7 est le seul à lire les sources, les
  catalogues, les scripts et le workflow CI.
- Contributeur potentiel écarté : sa documentation est `ARCHITECTURE.md`,
  `CLAUDE.md` et les ADR, pas le site.

## Ce qui était déjà tranché avant la lecture

Ces décisions ont été prises et exécutées avant que les personas ne lisent ;
elles ne sont pas remises en cause par ce dossier (session de grilling du
2026-09-06, commit `792503bca`) :

1. L'historique des versions a sa page (`historique.rst`, fin du sommaire) ;
   l'accueil de la doc ne porte plus ni changelog ni « dernière version ».
2. La feuille de route (`roadmap.rst`, H.11) est supprimée ; deux phrases au
   présent dans la page À propos renvoient aux jalons et Discussions GitHub.
3. La documentation décrit la version publiée, au présent, et rien d'autre
   (`CLAUDE.md`, section Documentation) ; la skill de release le vérifie par
   motifs en phase 1.
4. Le site en deux temps (page produit, puis clic vers la doc) est conservé ;
   l'accueil Sphinx est une présentation et trois sommaires à légende (Prise
   en main, Référence, Annexes) ; Contacts, Don, Remerciements et Crédits
   forment la page À propos.

## Suite

Relire `SYNTHESE.md`, cocher les changements retenus, ouvrir une branche
d'exécution. Tout `.rst` modifié embarque ses huit `.po` dans le même commit
(`scripts/doc-po-update.sh`, puis `scripts/doc-i18n-check.sh`) ; un changement
à `raccourcis.rst` ou `cmd_mode.rst` passe par `make help`.
