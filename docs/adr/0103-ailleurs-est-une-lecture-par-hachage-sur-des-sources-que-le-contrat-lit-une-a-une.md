# « Ailleurs » est une lecture par hachage sur des sources que le contrat lit une à une

Status: accepted.
See also: ADR-0063 (lire à travers les tenants), ADR-0065 (club et coach), ADR-0043 (positions
semblables), ADR-0068 (MET différente), `pkg/blunderdb/storage/elsewhere.go`

## Context

Le chantier « plusieurs bibliothèques ouvertes » veut savoir, pour une position, ce que les
autres bibliothèques en savent : analyse, commentaires, matchs où elle a été jouée, voisines.
Le démon lit déjà à travers les tenants d'un même `Storage` (`ReadAcross`, `StreamAcross`) ;
la CLI, elle, tient des fichiers distincts, donc des `Storage` distincts. Écrire la lecture
deux fois — une par mode — ferait diverger le jugement.

## Decision

1. Une **source** est un couple (`Storage`, tenant) nommé par l'appelant. `ReadSources` est
   l'unique éventail : il lit chaque source dans l'ordre, sous le contexte de son tenant (la
   RLS filtre chaque appel comme les autres). `ReadAcross` en est le cas « un seul `Storage` »
   (`SourcesOf`) ; il n'y a pas de second éventail.
2. `storage.Elsewhere(sources, requête, juge)` compose, par source, des lectures du contrat
   existantes : `Exists` par hachage, `Analyses().Load`, `Comments().ByPositions`,
   `MatchesByPosition` puis `MovesByMatch` (le coût du coup y est noté), `Search().Rank`
   (la copie de la source si elle la détient, sinon le plateau demandé, sans son id). Aucune
   méthode de store n'est ajoutée.
3. **Le juge est la bibliothèque qui lit** : ses seuils (`LibrarySettings`) classent les coûts
   lus ailleurs, et l'empreinte de sa table d'équité courante décide si une analyse faite à un
   score de match l'a été avec une autre table — signalée (`otherMet`), coûts affichés, jamais
   jugés (`other-met`), comme ADR-0068. L'argent n'est jamais concerné.
4. Exposition : le démon sert `across.positionElsewhere` sur le jeu de lecture privé du tenant
   qui écrit (« ici » n'est pas « ailleurs ») ; la CLI sert `blunderdb elsewhere --db … --other …`,
   les autres fichiers ouverts en lecture seule (`mode=ro`) et refusés s'ils ne sont pas au
   schéma courant, une migration étant une écriture (ADR-0007). Le bureau s'y branchera par le
   même service.

## Consequences

- Rien n'est écrit nulle part ; la lecture coûte une poignée d'appels par source, bornée par
  `MaxReadTenants` sources, `ElsewhereMaxMatches` matchs et `ElsewhereMaxNeighbours` voisines.
- Un hachage seul ne situe pas de plateau : sans position, les voisines ne sont cherchées que
  là où la source détient la position.
