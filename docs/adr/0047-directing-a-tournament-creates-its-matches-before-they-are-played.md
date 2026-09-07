# ADR-0047 — Diriger un tournoi, c'est créer ses Matchs avant qu'ils soient joués

- **Statut** : accepté — décidé le 2026-09-07, après un entretien de conception le même jour
  (dix-neuf décisions, `tasks/nicomaque/decisions.md`).
- **Amende** : ADR-0037 (blunderDB ne joue pas au backgammon), comme 0044 l'a fait pour la
  transcription.
- **Voir aussi** : ADR-0039 (le front web est verrouillé), ADR-0044 et ADR-0045 (la
  transcription et son brouillon), ADR-0005 (le démon n'authentifie personne).
- **Moteur** : Nicomaque, créé par **Nicolas Harmand**
  (`github.com/PileOfCells/backgammon-tournoi`).

## Le problème

blunderDB range les matchs d'un joueur par tournoi : le `Tournament` est une étiquette
posée *après* l'import, sur des matchs qui existent déjà. Diriger un tournoi est l'inverse —
il faut inscrire, apparier, lancer, saisir des résultats, tenir un tableau, classer, et
tout cela se passe avant qu'un seul fichier existe.

Un moteur complet fait déjà ce travail : Nicomaque, une bibliothèque Go sans dépendance,
fondée sur un journal d'événements, qui propose au directeur de tournoi quoi faire et
reconstruit son état par rejeu. Il ne persiste rien et n'a pas d'interface : son propre
`RESTE_A_FAIRE.md` place en tête « interface TD dans le logiciel hôte » et « persistance du
journal côté hôte ».

La question est donc posée à blunderDB, et ADR-0037 y avait déjà répondu par avance pour
une question voisine : *« un mode de jeu serait un second produit partageant un binaire
avec le premier, en concurrence pour la même maintenance »*. Diriger un tournoi est plus
loin encore du cœur du logiciel — « j'apporte mes matchs joués, je demande ce que j'ai
raté » — que jouer une partie ne l'était.

## La décision

**Un tournoi dirigé est une source de Matchs, et c'est ce qui le rattache à blunderDB.**

Le `Tournament` existant ne change pas d'identité : il gagne la possibilité d'être créé
*avant* ses Matchs. Chaque match lancé par le directeur est un **Slot** — deux
Participants, une longueur, une table, un résultat — que remplit ensuite une Transcription
(ADR-0044) ou un fichier importé. Le tournoi de club où rien n'est enregistré laisse ses
Slots vides ; le tournoi à transcription obligatoire, comme ceux du BMAB, les remplit tous,
et le directeur obtient dans une seule base ce que personne n'a aujourd'hui : les matchs
rangés par ronde et par joueur, analysés, cherchables.

Trois lignes tiennent cette décision et la séparent du second produit qu'ADR-0037 refuse :

1. **Une seule entité.** Il n'y a pas un « tournoi de blunderDB » et un « tournoi de
   Nicomaque ». Un `Tournament` porte optionnellement une **Direction** : tout ce que le
   directeur a décidé, dans l'ordre, jamais modifié, seulement prolongé. Un tournoi importé
   n'en a pas, un tournoi dirigé en a une, un tournoi du BMAB a les deux faces du même
   objet.

2. **Le moteur propose, le directeur décide, la Direction enregistre.** Rien n'est refusé
   sauf l'impossible. Un appariement manuel, un résultat incohérent, un match lancé au
   mauvais tour : tout est accepté, tracé, et signalé sans bloquer. C'est la posture de
   0044 — « les règles vérifient et n'imposent jamais » — transposée d'un match à une salle.

3. **L'état n'est jamais stocké.** Le classement, les arbres, les appariements, la
   proposition suivante sont rejoués depuis la Direction à chaque ouverture. Une coupure de
   courant ne coûte rien ; une correction ne laisse pas d'état intermédiaire faux.

**Ce que la décision n'ouvre pas** : l'écran des joueurs. Un tournoi se regarde aussi sur un
mur et sur un téléphone, et Nicomaque sait rendre ces pages. blunderDB écrit une page HTML
autonome que le directeur projette ou imprime ; il n'ouvre **aucune route** de consultation
et ne touche pas au front web, dont ADR-0039 a fermé le périmètre en écrivant qu'il ne
gérerait jamais les tournois. Si le besoin dépasse la page, il fera l'objet d'un ADR à lui,
écrit après un vrai tournoi et pas avant.

## Les conséquences

- La Direction est une table **append-only**, une ligne par événement, écrite dans la
  transaction du geste (schéma 2.23.0). L'invariant de parité vaut : la logique est sur
  `Database` et le contrat `Storage`, exposée au frontend et à une sous-commande CLI non
  interactive. Le démon ne reçoit rien.
- La zone principale de l'application affiche autre chose que le plateau pour la première
  fois, et seulement là : onglet Tournoi actif *et* Direction ouverte. Tout autre onglet
  ramène le plateau sans rien fermer.
- Le vocabulaire entre dans `CONTEXT.md` (section « Directing a tournament ») :
  **Direction**, **Participant**, **Directory**, **Slot**. Le mot *journal* est déjà pris
  par l'Entraînement, le mot *replay* par la Transcription : ils ne sortent pas de
  Nicomaque.
- Nicomaque cesse d'émettre des phrases françaises et émet des **codes** (libellés, notes de
  classement, avertissements, raisons d'attente), parce que blunderDB parle neuf langues et
  que ces chaînes iraient dans la base. Le journal est versionné. C'est fait avant le
  premier tournoi réel, pendant qu'aucun journal n'existe.
- Le moteur reste **la propriété de son auteur** et évolue chez lui : ce que ce chantier
  lui demande est écrit dans `tasks/nicomaque/nicomaque.md` et devient des issues du dépôt
  Nicomaque, consommées par tag. blunderDB ne bifurque pas le moteur.
- **Le crédit est visible** : « Nicomaque, moteur de tournoi créé par Nicolas Harmand », le
  lien vers le dépôt et vers sa documentation, dans l'aide intégrée, sur la page À propos,
  sur la page tournoi du manuel, sur un bouton info de la gestion d'un tournoi, et en pied
  des pages produites. La documentation du moteur est publiée dans les neuf langues de
  blunderDB.

## Ce qui a été écarté

- **Un exécutable séparé** ou une place dans gammonGo : c'est ce que ADR-0037 imposerait
  si un tournoi dirigé ne produisait pas de Matchs. Il en produit, et le regroupement par
  ronde et par joueur n'a de valeur que là où les matchs sont analysés.
- **Deux entités** (un tournoi « organisation » distinct du `Tournament`) : deux « Open de
  Lyon 2026 » à réconcilier au premier import.
- **Une fiche « personne » réutilisable** entre tournois : ce serait l'identité que
  `CONTEXT.md` refuse depuis toujours (*« blunderDB n'a aucune notion de personne derrière
  le nom »*). L'annuaire est une **vue** dérivée des Directions, exportable en CSV, et deux
  orthographes y restent deux lignes.
- **Un résultat déduit du Match rattaché** : pendant un tournoi, c'est la parole du
  directeur qui fait foi — forfaits, abandons, feuille papier. Le Match pré-remplit la
  saisie et, s'il la contredit plus tard, le désaccord est montré, jamais résolu tout seul.
