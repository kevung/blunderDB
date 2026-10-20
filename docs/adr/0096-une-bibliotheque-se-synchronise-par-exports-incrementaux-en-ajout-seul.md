# Une bibliothèque se synchronise par exports incrémentaux, en ajout seul

Status: accepted.
See also: ADR-0007 (rien n'est écrit chez le destinataire), ADR-0004 (le dossier surveillé sonde plutôt que d'écouter)

## Context

Un joueur qui travaille sur deux machines copie sa base `.db` à la main, ou la range dans un
dossier synchronisé, ce qui casse SQLite en mode WAL : le journal et la base voyagent séparément.
Or une bibliothèque blunderDB est déjà un ensemble qui se fusionne sans conflit : positions
identifiées par leur hachage Zobrist, matchs reconnus par leur empreinte, collections et leçons
fusionnées par nom, import d'une base idempotent. Il manque de quoi n'envoyer que la différence,
et de quoi la recevoir sans geste.

## Decision

1. **Le curseur est chez le producteur.** Un export incrémental (`domain.ExportOptions.Incremental`)
   lit puis avance un repère rangé dans les métadonnées de la bibliothèque qui exporte, sous la clé
   `sync_cursor:<nom>`. La machine qui reçoit importe une base ordinaire et n'écrit rien d'autre
   que ce qu'elle importe (ADR-0007). La clé n'est pas sur `issuance.CarriedMetadataKeys` : elle ne
   quitte jamais la base.
2. **Un repère par destination.** Le nom du repère est `SyncCursor` ; le bureau et la CLI le
   prennent par défaut du dossier de destination (absolu), le démon prend `default`. Deux dossiers
   — vers le portable, vers le NAS — avancent chacun le leur.
3. **La granularité est l'identifiant de ligne, pas la date ni le lot d'import.** `ingest.SyncMark`
   tient le plus grand identifiant déjà envoyé de chaque famille : position, match, tournoi,
   collection, membre de collection, leçon. Les deux moteurs allouent des identifiants qui ne font
   que croître (`AUTOINCREMENT`, séquences) : « au-dessus du repère » veut dire « entré depuis »,
   sans horloge à croire, sans changement de schéma, et pour tout ce qui entre — import, saisie,
   fusion — pas seulement les lots d'import. Une collection repart dès qu'un membre y a été ajouté.
4. **Ajout seul.** Une suppression ou une modification en place (commentaire réécrit, analyse
   refaite sur une position déjà envoyée) ne se propage pas ; la documentation le dit. Les filtres
   enregistrés et les paquets Anki ne partent pas dans un export incrémental : ce ne sont pas des
   ajouts datables par identifiant.
5. **Le repère avance quand le fichier est complet**, dans `ingest.ExportSQLite`, la seule écriture
   des trois modes ; `PlanExport` compte la même sélection sans rien écrire. Un export qui échoue
   laisse le repère : le suivant renvoie les mêmes lignes, que l'import reconnaît. Sur le démon, le
   repère avance quand le fichier est construit, avant que le client l'ait téléchargé ; un
   téléchargement perdu se rattrape par un export entier.
6. **L'écho s'éteint en un tour.** Ce que A a reçu de B est nouveau pour le repère de A et repart
   une fois vers B, qui le reconnaît sans rien créer ; l'échange suivant est vide. Ne pas renvoyer
   ce qui a été reçu demanderait d'écrire chez le receveur d'où vient chaque ligne.
7. **La réception est le dossier surveillé.** `watch` signale aussi les `.db` (`ingest.IsLibraryFile`,
   tenu hors de `IsImportable` : un import de dossier n'avale pas la base de l'utilisateur), que le
   bureau fusionne sans aperçu et que `import --watch` passe à `import --type database`. Un export
   incrémental s'écrit sous `.<nom>.part` puis est renommé, pour qu'une surveillance ne voie jamais
   un fichier à moitié écrit ; vers un dossier, il prend un nom neuf `sync-<machine>-<date>.db`,
   car la surveillance ne signale un nom qu'une fois.

## Consequences

- Deux bibliothèques convergent après un aller-retour, sans serveur ni compte ; le transport est
  celui de l'utilisateur (Syncthing, clé USB, courriel).
- Un `.dbx` n'est pas pris par la surveillance, qui ne peut pas demander de mot de passe.
- Réinitialiser une synchronisation, c'est faire un export entier : l'import est idempotent.
