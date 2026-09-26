# ADR-0039 — Le front web est en consultation, son périmètre est verrouillé, et il est éteint par défaut

Statut : acceptée.
Voir aussi : ADR-0005 (le démon n'authentifie personne), ADR-0034.

## Contexte

`blunderdb serve` n'avait aucun client hors du bureau. Consulter sa bibliothèque depuis une
tablette ou un téléphone demande une page, pas une seconde application. Le risque est un
second front à maintenir qui grossirait jusqu'à refaire le bureau et doublerait gestes, bugs,
documentation et traductions.

## Décision

1. **Le périmètre est fermé.** Le front web sait **consulter** une position, son analyse et
   son plateau ; **chercher** avec la grammaire de jetons de la ligne de commande ; **réviser**
   un paquet Anki, réponse et note comprises. Rien d'autre : ni édition, ni import,
   suppression, collections, matchs, tournois, configuration, tables ou export. La révision est
   la seule écriture, parce qu'une révision qui ne note pas n'en est pas une.
2. **Il est éteint par défaut** : `serve --web` seulement, parce que le démon n'authentifie
   personne (ADR-0005). Les fichiers statiques sont servis **sans tenant** (une page ne
   contient aucune donnée) ; `publicPaths` étant écrit en négatif (« ni /v1/ ni /ops/ »),
   toute nouvelle famille de routes doit être vérifiée contre lui.
3. **Le plateau est dessiné par `renderPositionSVG`**, le même dessinateur que le rapport HTML
   et l'export d'image ; jamais un second. Le front est bâti par Vite et son résultat est
   versionné dans `internal/server/webui/dist`, embarqué, régénéré par `make web`.
4. **Il parle le contrat `/v1/` de tout le monde** : aucune route n'existe pour lui seul.

## Conséquences

- Une demande d'édition depuis le web est refusée par cette ADR ; ouvrir le périmètre, c'est
  remplacer cette ADR et reposer « faut-il une seconde application ? ».
- Rien ne détecte un `dist` périmé : `make web` est à relancer après tout changement de
  `frontend/src/web/`. Un test d'empreinte est écarté (un paquet Vite n'est pas reproductible
  octet pour octet d'une machine à l'autre).
- Le front web n'a pas de traductions propres.
- Écarté : un dessinateur JavaScript nu, plus léger mais impossible à tenir en phase.

## Garde

`internal/server/webui/webui_test.go`.
