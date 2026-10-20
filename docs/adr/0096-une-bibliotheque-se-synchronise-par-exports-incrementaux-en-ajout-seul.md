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
   aussitôt les perdrait sans bruit. Aucune durée ne borne une transaction : le repère n'est donc
   pas cru au bout d'un délai, mais quand on sait que ces transactions sont finies. Un moteur non
   ordonné qui sait le dire (`ingest.SyncFence`) garde chaque repère *en attente*
   (`SyncCursor.Pending`) avec une *barrière* prise juste après la lecture des identifiants, qui
   nomme les transactions alors en cours ; chaque export part du dernier repère *établi*
   (`SyncCursor.Settled`). Avant de lire, l'export vérifie les barrières des repères en attente,
   du plus ancien au plus récent, et s'arrête à la première qui tient : une fois le fichier
   complet, le plus récent des repères dont la barrière était levée devient établi, car cet
   export, parti d'en dessous, a vu tout ce que ces transactions ont validé. Un repère plus
   récent n'est jamais établi devant un plus ancien dont la barrière tient. Sur PostgreSQL
   (`postgres.Storage.SyncFence`), la barrière retient `pg_snapshot_xmax(pg_current_snapshot())`
   — levée quand le `xmin` de l'instantané courant l'atteint, ce qui couvre toute transaction
   ayant écrit, quel que soit son rôle — et les couples `(pid, xact_start)` de
   `pg_stat_activity` de la base, ce qui couvre une transaction qui a tiré une valeur de
   séquence sans avoir encore d'identifiant de transaction. Reste un trou théorique : une
   transaction d'un autre rôle, sans droit de lecture sur sa ligne de `pg_stat_activity`
   (`xact_start` y est nul), qui a tiré une valeur de séquence mais n'a pas encore de `xid` au
   moment de la barrière. Ni l'une ni l'autre moitié ne la couvre : son `xid`, pris après,
   dépasse le `xmax` retenu. Le démon écrit sous un rôle unique, que `pg_stat_activity` lui
   montre toujours ; une écriture concurrente sous un autre rôle sort de ce modèle. Une
   transaction longue retarde le
   repère, sans limite et sans perte ; une barrière illisible tient. Un moteur ni ordonné ni
   doté de barrière (`storage/remote`) n'établit jamais de repère : chaque export incrémental y
   emporte tout depuis le repère zéro. Le recouvrement coûte une refusion idempotente ; le trou
   coûterait une perte. Un délai fixe (24 h) a été écarté : il suppose une borne que rien
   n'impose, et un repère cru au bout du délai sans qu'un export ait relu entre-temps passait
   au-dessus des lignes validées tard.
4. **Ajout seul.** Une suppression ou une modification en place (commentaire réécrit, analyse
   refaite sur une position déjà envoyée) ne se propage pas ; la documentation le dit. Un
   commentaire **ajouté** à une position déjà envoyée est un ajout : le repère suit aussi les
   identifiants de `comment`, et la position repart avec tous ses commentaires, que l'import
   ajoute par texte. Un commentaire réécrit ne repart pas : l'identifiant ne voit pas une
   écriture en place, et l'import n'écrase jamais une ligne existante, si bien que le nouveau
   texte arriverait à côté de l'ancien au lieu de le remplacer. Pour qu'un identifiant de
   commentaire ne resserve jamais, la table `comment` est en `AUTOINCREMENT` sous SQLite
   (schéma 2.46.0) ; sous PostgreSQL c'est déjà un `BIGSERIAL`. Un repère écrit avant ce suivi
   lit un repère de commentaire nul : le premier export incrémental après la mise à jour renvoie
   donc une fois toutes les positions commentées, que l'import reconnaît sans doublon ; le
   manuel le dit. Les filtres
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
   un fichier à moitié écrit ; vers un dossier, il prend un nom neuf `sync-<machine>-<date>.db`.
   **Un export incrémental n'écrase jamais un fichier** : vers un nom de fichier déjà pris, il
   est refusé. Le fichier en place porte le delta précédent, que le repère compte déjà comme
   envoyé ; si l'autre machine ne l'a pas encore fusionné, l'écraser le perdrait, et celui qui
   exporte ne peut pas savoir s'il l'a été (décision 1). La surveillance reconnaît un fichier à
   son nom, sa taille et sa date de modification : un fichier réécrit sous un nom déjà vu
   (copie à la main, outil de synchronisation) est proposé de nouveau. La bibliothèque ouverte n'est jamais
   proposée à elle-même (`watch.Watcher.Skip`, `Database.IsOpenDatabase`, qui compare les
   fichiers et non les chemins écrits), et sa fusion à la main est refusée
   (`database.ErrImportIsOpenDatabase`).

## Consequences

- Deux bibliothèques convergent après un aller-retour, sans serveur ni compte ; le transport est
  celui de l'utilisateur (Syncthing, clé USB, courriel).
- Un `.dbx` n'est pas pris par la surveillance, qui ne peut pas demander de mot de passe.
- Réinitialiser une synchronisation, c'est faire un export entier : l'import est idempotent.
- Sur PostgreSQL, chaque export incrémental renvoie au moins ce qu'a porté le précédent ; une
  transaction restée ouverte (`idle in transaction`) empêche tout repère de s'établir, et chaque
  export renvoie tout depuis le dernier établi tant qu'elle dure.
- Sur le backend `remote`, un export incrémental est un export entier.
