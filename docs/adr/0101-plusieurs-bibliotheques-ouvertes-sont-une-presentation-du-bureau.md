# Plusieurs bibliothèques ouvertes sont une présentation du bureau ; le transfert et « Ailleurs » vivent sous le contrat

Status: accepted (étape 1 livrée : groupes et bascule ; étapes 2 à 5 en cours, issues #690 à #693).
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
   l'affichage, la liste des matchs seulement si elle n'a jamais été lue ou si leur nombre a
   bougé. Le cache de positions n'est jamais partagé entre deux bibliothèques : un même
   identifiant y nomme deux lignes différentes, comme les ids de matchs et de tournois d'un
   critère de recherche.
4. **Un fichier (chemin canonique) ne s'ouvre pas dans deux groupes.** Le rouvrir amène le
   groupe qui le tient. L'assistant MCP et l'affichage de tournoi suivent la poignée liée,
   donc le groupe actif.
5. **Tant qu'une tâche écrit la bibliothèque active, le changement de groupe est refusé**
   (`BusyError`) : ses écritures passent par la poignée liée, qui changerait de fichier sous
   elle. L'étape 2 attache les tâches à leur bibliothèque et lève ce refus.
6. **Le transfert vit sous le contrat, pour les trois modes** : une fonction `Storage` source →
   `Storage` destination avec une `ingest.Selection`, qui reprend la fusion de l'import natif,
   dédoublonne par hachage, copie les métadonnées par `issuance.CarriedMetadataKeys` et n'écrit
   rien qui dise d'où vient la copie (ADR-0007). Au démon, le transfert est un tirage depuis un
   tenant de `X-Read-Tenants` vers le sien, jamais l'inverse (ADR-0065).
7. **La vue « Ailleurs » cherche par hachage** avec le même service que les lectures
   `across.*`, et juge avec les seuils du groupe actif ; une analyse faite avec une autre MET
   est signalée comme aujourd'hui.

## Consequences

- Fermer un groupe ferme sa base ; à l'arrêt, les bases en attente sont fermées après les
  tâches. Seule la session de la bibliothèque active est enregistrée à la fermeture.
- `database` gagne deux fonctions (`Exchange`, `OpenPath`) et un champ (`path`), en fonctions
  et non en méthodes pour ne pas élargir la liaison Wails.
- Une base ouverte dans un nouveau groupe n'affiche pas la progression d'une migration : la
  poignée neuve n'a pas le rappel de progression du bureau.
