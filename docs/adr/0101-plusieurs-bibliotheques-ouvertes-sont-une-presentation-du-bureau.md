# Plusieurs bibliothèques ouvertes sont une présentation du bureau ; le transfert et « Ailleurs » vivent sous le contrat

Status: accepted (livrée : groupes, bascule, tâches attachées à leur bibliothèque, transfert,
panier et verrou, vue « Ailleurs » et échange sur place, écran partagé).
See also: ADR-0007 (rien n'est écrit chez le destinataire), ADR-0055 (budget de contexte),
ADR-0065 (lecture à travers les tenants), ADR-0001 (l'identité d'une position est son hachage).

## Context

Un joueur travaille souvent avec deux bases : la sienne et celle d'un club, d'un tournoi ou d'un
élève. Jusqu'ici, passer de l'une à l'autre fermait la première ; comparer une position ou
recopier un match demandait un export puis un import. Le bureau lie un seul `*database.Database`
à Wails : toutes les liaisons existantes (des centaines de méthodes) parlent à cette poignée.

## Decision

1. **Les groupes sont de la présentation, au bureau seul.** Un groupe d'onglets par bibliothèque
   ouverte (séparateur, nom, couleur), un groupe actif, l'état de chaque groupe photographié en
   mémoire. Ni la CLI ni le démon ne portent d'espace de travail.
2. **La poignée liée porte toujours la bibliothèque active.** Les autres attendent, ouvertes,
   dans des poignées à elles que tient `internal/gui` (`libraries.go`). Changer de groupe
   *échange* le contenu des deux poignées (`database.Exchange` : connexion, `Storage`, verrou,
   lecture seule, chemin, sessions de transcription et de Duel) sans ouvrir ni fermer de
   fichier : la bascule ne coûte aucune entrée-sortie. Les deux générations avancent, si bien
   qu'aucune tâche ne peut écrire à travers un échange. Un test place chaque champ de
   `Database` du côté de la bibliothèque ou de la poignée : un champ nouveau échoue tant qu'il
   n'est pas classé.
3. **La bascule est instantanée côté écran.** Les vues, la liste de positions et son cache,
   les critères de recherche (ceux du formulaire monté compris), la liste des matchs, les
   compteurs et le défilement de chaque boîte défilante du panneau sont photographiés quand on
   le quitte et rendus quand il revient, sans requête ; les compteurs sont relus après
   l'affichage, la liste des matchs seulement si elle n'a jamais été lue, si leur nombre a
   bougé ou si une tâche de fond a écrit la bibliothèque pendant l'absence. Le cache de positions n'est jamais partagé entre deux bibliothèques : un même
   identifiant y nomme deux lignes différentes, comme les ids de matchs et de tournois d'un
   critère de recherche.
4. **Un fichier (chemin canonique) ne s'ouvre pas dans deux groupes.** Le rouvrir amène le
   groupe qui le tient. L'assistant MCP et l'affichage de tournoi suivent la poignée liée,
   donc le groupe actif.
5. **Une tâche de fond reste attachée à sa bibliothèque.** L'import de fichiers (le pipeline),
   le dossier surveillé et l'analyse GammonNet par lots tournent sur une poignée *empruntée*
   (`database.Borrow`) : une poignée à elle sur la connexion et le `Storage` de la
   bibliothèque, qui partage son créneau de tâche (`jobGate`). L'échange déplace la
   bibliothèque et son créneau, pas la poignée empruntée : la tâche continue d'écrire la
   bibliothèque où elle a commencé, jamais celle qu'affiche la poignée liée. Le lot d'import
   ouvert par l'interface passe à la poignée empruntée (`HandOverImportBatch`), qui le ferme.
   La poignée empruntée ne possède rien (ni verrou de fichier, ni connexion à fermer) : qui
   ferme ou remplace la bibliothèque arrête d'abord ses tâches (`stopGroupJobs`), sinon leur
   écriture suivante échoue sur une connexion fermée — jamais dans un autre fichier.
   Le dossier surveillé appartient au groupe où il a démarré ; un groupe non affiché importe en
   fond ce que le pipeline lit seul, le reste (base à fusionner, texte de position) attend son
   retour. Chaque événement de tâche porte son groupe : hors écran, l'en-tête du groupe porte
   une pastille (en cours, terminé), et le compte rendu est dit à son retour. Restent refusés,
   parce qu'ils suivent l'écran : le rollout (son résultat s'affiche sur la position du
   plateau, et il écrit par la poignée liée), l'affichage de tournoi (il sert la bibliothèque
   active), le Duel (il tient le plateau et ses vues), l'entraînement (son chronomètre et son
   journal ne suivent aucun groupe), et l'import fichier par fichier que seule l'interface sait
   lire ou la fusion d'une base, le temps qu'ils durent.
6. **Le transfert vit sous le contrat, pour les trois modes** : une fonction `Storage` source →
   `Storage` destination avec une `ingest.Selection`, qui reprend la fusion de l'import natif,
   dédoublonne par hachage, copie les métadonnées par `issuance.CarriedMetadataKeys` et n'écrit
   rien qui dise d'où vient la copie (ADR-0007). Au démon, le transfert est un tirage depuis un
   tenant de `X-Read-Tenants` vers le sien, jamais l'inverse (ADR-0065).
   Au bureau, glisser une ligne sur l'en-tête d'un autre groupe, ou y verser le panier, appelle
   ce même transfert (`TransferToGroup` → `Database.TransferTo`) sur deux poignées empruntées :
   il tourne en fond, attaché au groupe d'arrivée, et compte comme un import de celui-ci (un
   vacuum l'attend, `CancelImport` l'arrête). Le panier et le verrou d'un groupe sont la
   mémoire du bureau, jamais celle d'une base.

7. **La vue « Ailleurs » cherche par hachage** avec le même service que les lectures
   `across.*`, et juge avec les seuils du groupe actif ; une analyse faite avec une autre MET
   est signalée comme aujourd'hui.

8. **L'écran partagé montre un groupe vivant et une visionneuse en lecture**, pas deux groupes
   vivants. Le front tient un seul jeu de stores, photographié par groupe (point 3), et la
   poignée liée ne porte que la bibliothèque active (point 2) : un second groupe complet
   simultané demanderait de réécrire chaque store et chaque liaison par groupe. Le second volet
   est donc une visionneuse fidèle (plateau, analyse, commentaire ; la bibliothèque, un match
   rejoué coup par coup ou une collection) qui lit son groupe par son numéro
   (`ViewerSteps`, `ViewerShow`, `ViewerCatalogOf`, sous le verrou des groupes) et n'écrit ni
   store du volet vivant ni base. L'égalité des volets tient par l'activation : cliquer la
   visionneuse active son groupe, les volets échangent leurs rôles et chaque bibliothèque garde
   son côté ; le groupe actif reste celui du volet qui reçoit le clavier. Les deux volets ont la
   même largeur ; sous 1280 px de large, l'option se refuse et propose l'échange sur place.

## Consequences

- Fermer un groupe ferme sa base ; à l'arrêt, les bases en attente sont fermées après les
  tâches. Seule la session de la bibliothèque active est enregistrée à la fermeture.
- `database` gagne deux fonctions (`Exchange`, `OpenPath`) et un champ (`path`), en fonctions
  et non en méthodes pour ne pas élargir la liaison Wails.
- Une base ouverte dans un nouveau groupe n'affiche pas la progression d'une migration : la
  poignée neuve n'a pas le rappel de progression du bureau.
- Une tâche sur une poignée empruntée et l'interface écrivent la même base sans partager le
  verrou `Database.mu` de la couche historique : c'est SQLite (WAL, `busy_timeout`) qui
  sérialise leurs écritures, comme pour le démon, dont les backends n'ont pas de verrou global.
- Une seule analyse par lots tourne à la fois, tous groupes confondus : en lancer une autre
  remplace celle en cours, quel que soit son groupe (elles prennent chacune tous les cœurs).
