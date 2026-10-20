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
   collection, membre de collection, leçon. Les identifiants ne font que croître
   (`AUTOINCREMENT`, séquences) : « au-dessus du repère » veut dire « entré depuis », sans horloge
   de la machine à croire, sans changement de schéma, et pour tout ce qui entre — import, saisie,
   fusion. Une collection repart dès qu'un membre y a été ajouté.
   **Croître n'est pas devenir visible dans l'ordre.** Sur SQLite, l'écrivain unique valide une
   transaction avant que la suivante prenne un identifiant : le plus grand identifiant lu couvre
   tout ce qui est en dessous (`sqlite.Storage.IDsVisibleInOrder`, `ingest.OrderedIDs`). Sur
   PostgreSQL, un import concurrent du même tenant tient des valeurs de séquence inférieures au
   plus grand identifiant lu et ne les rend visibles qu'à sa validation : un repère avancé
   aussitôt les perdrait sans bruit. Un moteur qui ne se déclare pas ordonné garde donc chaque
   repère *en attente* (`SyncCursor.Pending`) pendant `ingest.SyncSettleDelay` (24 h, bien
   au-delà de toute transaction d'import) ; chaque export part du dernier repère *établi*
   (`SyncCursor.Settled`) et renvoie ce qui est entré pendant ce délai. Le recouvrement coûte
   une refusion idempotente ; le trou coûterait une perte. Un repère fondé sur la visibilité
   transactionnelle (`pg_current_snapshot`) serait exact, mais demande de lire les identifiants
   dans l'instantané du test, donc une méthode de plus au contrat de stockage pour un seul
   moteur.
4. **Ajout seul.** Une suppression ou une modification en place (commentaire réécrit, analyse
   refaite sur une position déjà envoyée) ne se propage pas ; la documentation le dit. Les filtres
   enregistrés et les paquets Anki ne partent pas dans un export incrémental : ce ne sont pas des
   ajouts datables par identifiant.
5. **Le repère avance quand le fichier est complet et que rien n'a été écarté**, dans
   `ingest.ExportSQLite`, la seule écriture des trois modes ; `PlanExport` compte la même sélection
   sans rien écrire. Un export qui échoue laisse le repère. Un export qui a écarté une ligne
   (`ExportReport.Skipped`) le laisse aussi et le dit (`SyncHeld`) : le repère passerait au-dessus
   d'une ligne qui n'est pas partie. Une ligne qui échoue toujours fait donc renvoyer, à chaque
   export, tout ce qui la suit — un recouvrement, que le journal signale. Sur le démon, le repère
   avance quand le fichier est construit, avant que le client l'ait téléchargé ; un téléchargement
   perdu se rattrape par un export entier ou sous un nouveau `syncCursor`, ce que
   `mode_headless.rst` dit au client.
6. **L'écho s'éteint en un tour.** Ce que A a reçu de B est nouveau pour le repère de A et repart
   une fois vers B, qui le reconnaît sans rien créer ; l'échange suivant est vide. Ne pas renvoyer
   ce qui a été reçu demanderait d'écrire chez le receveur d'où vient chaque ligne.
7. **La réception est le dossier surveillé.** `watch` signale aussi les `.db` (`ingest.IsLibraryFile`,
   tenu hors de `IsImportable` : un import de dossier n'avale pas la base de l'utilisateur), que le
   bureau fusionne sans aperçu et que `import --watch` passe à `import --type database`.
   **Un `.db` déjà présent au démarrage de la surveillance est proposé** comme s'il venait
   d'apparaître, contrairement aux matchs : avec Syncthing, le fichier arrive le plus souvent
   pendant que blunderDB est fermé, et le receveur ne note pas ce qu'il a lu (décision 1), donc
   l'ignorer le perdrait. La fusion d'un fichier connu n'écrit rien. Le coût : chaque démarrage
   relit tous les `.db` que le dossier contient encore, en temps proportionnel à leur taille
   cumulée ; la documentation demande de supprimer de temps en temps les fichiers reçus. Retenir
   ce qui a été lu, pour s'en dispenser, serait écrire chez le receveur. Un export
   incrémental s'écrit sous `.<nom>.part` puis est renommé, pour qu'une surveillance ne voie jamais
   un fichier à moitié écrit ; vers un dossier, il prend un nom neuf `sync-<machine>-<date>.db`,
   car la surveillance ne signale un nom qu'une fois.

## Consequences

- Deux bibliothèques convergent après un aller-retour, sans serveur ni compte ; le transport est
  celui de l'utilisateur (Syncthing, clé USB, courriel).
- Un `.dbx` n'est pas pris par la surveillance, qui ne peut pas demander de mot de passe.
- Réinitialiser une synchronisation, c'est faire un export entier : l'import est idempotent.
- Sur PostgreSQL, un export incrémental n'est jamais vide dans les 24 h qui suivent un ajout.
