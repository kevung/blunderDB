# ADR-0057 — La transcription et la direction s'exposent par l'API à un client externe

Statut : acceptée.
Amende : ADR-0045 règle 9 (« `serve` exposes nothing »), ADR-0047 (« Le démon ne reçoit
rien »), ADR-0056 (CLI en lecture seule, démon muet).
Laisse intacte : ADR-0039 (le front web embarqué reste en consultation).
Voir aussi : ADR-0005, ADR-0044, ADR-0045, ADR-0047, ADR-0056.

## Contexte

Transcrire un match et diriger un tournoi ne se font qu'au bureau : la logique vit sur
`Database`, le démon `serve` n'en expose rien et `call` non plus. Un client externe —
gammonGo, qui embarque `pkg/blunderdb/server` — veut transcrire depuis une tablette et
diriger une salle depuis plusieurs postes. Il lui faut une API, pas un second front embarqué :
l'ADR-0039 a fermé ce périmètre et la raison tient toujours (un second front à maintenir).
Exposer des gestes d'écriture sur un démon qui n'authentifie personne (ADR-0005), à plusieurs
clients à la fois, pose trois questions : où vit l'état non durable d'une transcription,
comment deux écritures concurrentes se détectent, et comment un client apprend qu'un autre a
écrit.

## Décision

1. **L'API, pas le front web.** Transcription, Direction et Rencontre sont exposées par les
   routes `/v1/` et donc par `call`, sur le contrat de tout le monde (ADR-0039 règle 4). Le
   client visé est externe (gammonGo, scripts). Le front web embarqué n'en reçoit rien :
   ADR-0039 règle 1 est inchangée, et l'ouvrir reste la remplacer.
2. **La logique descend sous le contrat.** Direction, Rencontre et Transcription vivent sur le
   contrat `Storage` et un service commun ; `Database` en devient la façade pour le bureau et
   la CLI. Une seule implémentation sert le GUI, la CLI et le démon (parité, `CLAUDE.md`).
3. **La session de transcription est un cache en mémoire.** Une session par (tenant,
   brouillon), fermée après un délai d'inactivité ; un identifiant de session inconnu ou
   expiré rend **410**, et le client rouvre le brouillon, curseur en fin de document. Le
   document reste écrit après chaque geste (ADR-0045 règle 1) : une session perdue
   (inactivité, redémarrage, autre instance) ne perd que la pile d'annulation, jamais un
   geste. Sous `call`, chaque appel est sa propre session : pas d'annulation d'un appel à
   l'autre.
4. **Tout geste d'écriture porte sa version, obligatoirement.** `If-Match` absent → **428** ;
   version périmée → **409** avec l'état frais, que le client relit avant de rejouer son
   geste s'il reste valide. La comparaison se fait dans la transaction du geste. Direction :
   la version est le numéro du dernier événement du journal ; un geste de salle compare
   celles de toutes les épreuves membres. Transcription : une **colonne de révision** sur la
   table des brouillons, incrémentée à chaque geste — un changement de schéma (bump de
   `DatabaseVersion`, migration SQLite et PostgreSQL), parce que la version doit se lire et
   se comparer sans désérialiser le document. Les lectures rendent un `ETag` ; `If-None-Match`
   → 304. Le GUI de bureau passe par le même contrôle.
5. **L'écriture est éteinte par défaut, la lecture non.** `serve --direction` ouvre les gestes
   de Direction et de Rencontre, `serve --transcription` ceux de la Transcription ; sans ces
   drapeaux, les routes d'écriture répondent comme absentes. Les routes de lecture sont
   toujours servies, sous tenant, comme le reste de `/v1/`.
6. **Un client apprend qu'un autre a écrit par SSE.** `GET /v1/events` diffuse un message
   court par geste validé (sorte, identifiant, version), publié après `commit` ; le client
   relit ce qui l'intéresse, il n'y a qu'un format de lecture. Le bus est en mémoire par
   processus et par tenant ; en PostgreSQL à plusieurs instances, il passe par
   `LISTEN/NOTIFY`. SQLite reste une instance par construction.
7. **Aucune authentification dans le moteur** (ADR-0005) : ni compte, ni rôle de directeur ou
   d'arbitre. Le démon fait confiance à `X-Tenant-ID` derrière un proxy authentifiant ; un
   rôle est une règle du proxy sur un préfixe de route. Les avertissements de
   `mode_headless.rst` couvrent les nouveaux drapeaux.

## Conséquences

- La CLI garde ses commandes actuelles ; les gestes d'écriture en ligne de commande passent
  par `call`, pas par de nouvelles sous-commandes.
- `openapi.yaml` régénéré ; `mode_headless.rst`, `cmd_mode.rst` et leurs huit `.po` décrivent
  les drapeaux, les codes 409/410/428 et le flux SSE.
- La route SSE rejoint les routes sans échéance ; la compression la vide à chaque message ;
  la limitation de débit compte une connexion, pas ses messages.
- Un `Idempotency-Key` sur un geste évite qu'un double envoi saisisse deux résultats.
- Écartés : remplacer l'ADR-0039 pour un front web éditeur (un second front, que rien ne
  demande tant que le client est externe) ; une session persistée en base (un bump pour une
  pile d'annulation) ; une session tenue par le client (le moteur devrait accepter une pile
  fournie) ; la version facultative, « le dernier qui écrit gagne » (deux directeurs saisiraient
  deux résultats sur la même table sans le savoir) ; la révision dans le document JSON (se lire
  exige de le désérialiser) ; les websockets (le sens serveur → client suffit) ; une instance
  par tenant imposée pour la direction.

## Garde

À écrire avec les lots H4–H7 de `tasks/plan-headless-transcription-direction-2026-10.md` : un
test de course (deux gestes concurrents, un 409), un geste sans `If-Match` → 428, une session
expirée → 410 puis réouverture sans geste perdu, un geste → un message SSE, les routes
d'écriture absentes sans leur drapeau.
