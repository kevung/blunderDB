# ADR-0039 — Le front web est en consultation, son périmètre est verrouillé, et il est éteint par défaut

- **Statut** : accepté
- **Date** : 2026-09-06
- **Fiche** : J.5 (#295)
- **Voir aussi** : ADR-0005 (le démon n'authentifie personne), #278 (un seul
  rendu du plateau), G.8 (#289, le contrat d'API)

## Le problème

`blunderdb serve` sert cent cinquante-sept routes et n'a aucun client hors du
bureau. Une bibliothèque consultable depuis une tablette entre deux rondes,
ou depuis un téléphone dans le train, ne demande pas une seconde application :
elle demande une page.

Le risque est nommé par la fiche elle-même : **un second front à maintenir**.
Une interface web qui grossirait jusqu'à refaire l'application de bureau
doublerait tout — les gestes, les bugs, la documentation, les traductions —
pour un public qui n'a jamais demandé cela.

## La décision

### 1. Le périmètre est fermé, pas « minimal pour l'instant »

Le front web sait faire trois choses, et cette liste est la décision :

- **consulter** une position, son analyse et son plateau ;
- **chercher**, avec la grammaire de jetons que la ligne de commande parle
  déjà ;
- **réviser** un paquet Anki, réponse et note comprises.

Il ne sait pas, et ne saura pas : éditer une position, importer, supprimer,
gérer les collections, les matchs, les tournois, la configuration, la
génération de tables, l'export. Une fonctionnalité qui manque ici n'est pas un
manque, c'est la décision.

La révision Anki est la seule ÉCRITURE, et elle est dedans parce qu'une
révision qui ne note pas n'est pas une révision — le calendrier FSRS est
l'objet même du geste. Toute autre écriture est dehors.

### 2. Il est éteint par défaut

`serve` ne sert la page que si on la lui demande (`--web`). Le démon
n'authentifie personne (ADR-0005) : livrer une interface atteignable par un
navigateur, allumée d'office, inviterait exactement le déploiement que
l'ADR-0005 interdit — un démon exposé sans mandataire devant. L'opt-in est la
seule valeur par défaut honnête.

Les fichiers statiques sont, eux, servis **sans tenant** : un navigateur doit
pouvoir charger la page avant que le mandataire ne lui attribue quoi que ce
soit, et une page ne contient aucune donnée. C'est dit ici parce que
`publicPaths` est une liste écrite en négatif — « tout ce qui n'est ni /v1/ ni
/ops/ » — et qu'une liste écrite en négatif change de sens quand l'ensemble
change. C'est exactement comme cela que `vacuum` et `purge` sont devenus
publics une fois passés sous `/ops/` (G.5, #233).

### 3. Le plateau est dessiné par LE dessinateur, pas par un second

Le front web importe `renderPositionSVG`, le même dessinateur que le rapport
HTML et l'export d'image — celui que #278 a précisément extrait pour qu'il n'y
en ait qu'un. Un second dessinateur en JavaScript nu aurait été plus léger à
écrire et impossible à tenir en phase : deux plateaux qui divergent d'un pion
sont deux plateaux faux.

Le prix est un empaquetage : le front web est bâti par Vite, comme
l'application de bureau, et son résultat est **versionné** dans
`internal/server/webui/dist` puis embarqué dans le binaire. C'est la même
discipline que les paquets d'aide (ADR-0034), `openapi.yaml` et le client
Python : engendré, versionné, régénéré par une cible de `make`.

La contrepartie est réelle et vaut d'être écrite : **rien ne détecte
automatiquement qu'un `dist` est périmé**. `make web` est à relancer quand
`frontend/src/web/` change, et l'oublier livre une page en retard sur son
code. Un test d'empreinte a été écarté — un paquet Vite n'est pas reproductible
octet pour octet d'une machine à l'autre — et le remplacer par une
vérification en intégration continue est la piste, pas la promesse.

### 4. Il parle le même contrat que tout le monde

Aucune route n'a été ajoutée pour lui. Le front web appelle `/v1/…` comme le
client Python et comme n'importe quel client : c'est ce que G.8 (#289) a rendu
possible, et c'est ce qui garantit qu'il ne peut rien faire que le contrat
n'expose déjà.

## Conséquences

- Une demande d'édition depuis le web est refusée **par cette ADR**, pas par
  un arbitrage au cas par cas.
- Le front web n'a pas de traductions propres : il parle la langue de ses
  libellés, et l'internationalisation est un chantier à décider, pas un
  oubli.
- Si un jour le périmètre doit s'ouvrir, c'est cette ADR qu'on remplace — et
  la question « faut-il une seconde application ? » se repose entière.
