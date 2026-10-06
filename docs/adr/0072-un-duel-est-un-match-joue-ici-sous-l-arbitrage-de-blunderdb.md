# ADR-0072 — Un Duel est un match joué ici, sous l'arbitrage de blunderDB

Statut : acceptée. Remplace ADR-0037.
Voir aussi : ADR-0073 (la Cadence et la durée de décision), ADR-0044 (transcrire n'est pas jouer), ADR-0045 (le brouillon), ADR-0057 (l'API
offerte à un client externe), ADR-0015 (`serve` opère sur une bibliothèque), ADR-0060 (le
rollout, qui joue déjà des parties).

## Contexte

L'ADR-0037 refusait le mode de jeu pour sa surface — les règles de bout en bout, des dés
auxquels on croit, le videau, le score, l'historique, l'abandon — et non pour sa difficulté,
et demandait, pour être rouverte, une mesure de la demande plutôt qu'un argument de
faisabilité. Deux choses ont changé. Le **coût** : la Transcription a construit le document
d'Actions, le Replay qui en dérive le score, Crawford et la fin de partie, et la
matérialisation en Match ; le rollout a construit une boucle de partie où gammonNet choisit
le coup et décide du videau. La **demande** : l'auteur et des utilisateurs veulent jouer puis
revoir, sans posséder XG ni installer gnubg. Jouer ici produit en outre une donnée qu'aucun
import ne fournit complètement : la durée de chaque décision.

## Décision

1. **blunderDB joue, et ce qu'il joue sort comme un Match ordinaire.** Un Duel inverse deux
   des trois propriétés de l'ADR-0044 — quelqu'un décide, les règles refusent — et garde la
   troisième : terminé, c'est un Match comme un import en produit, analysé, compté au
   Performance Rating, exportable.
2. **Deux Côtés, chacun externe ou délégué à un Bot ; blunderDB est l'Arbitre.** L'Arbitre
   lance les dés, impose les règles, tient le score et l'horloge (ADR-0073). Il ne sait pas ce qu'il y a
   derrière un Côté externe. Humain contre Bot, deux humains à travers un client, Bot contre
   Bot : trois configurations d'un seul modèle, aucune n'est un cas particulier dans le code.
   Le bureau ne propose que la première.
3. **Un objet distinct de la Transcription, sur le même noyau de règles.** L'état de match
   devient une machine de règles exportée — un état et une Action donnent le nouvel état ou
   un refus motivé. Le Replay s'en sert pour signaler et continuer ; l'Arbitre, pour refuser.
   Le Duel écrit le même format d'Action, donc la matérialisation en Match est partagée. Il a
   en propre son brouillon, sa session et ses routes : aucun geste de Transcription (corriger
   où est le curseur, supprimer en arrière, saisir les dés) ne s'applique à un Duel.
4. **La logique vit sous le contrat et s'expose aux trois modes.** GUI, CLI et API pilotent
   le même Arbitre ; un client externe (gammonGo) se greffe sur l'API sans rien réécrire.
   L'appariement, la présence, le temps réel réseau et le choix du tenant où atterrit le
   Match restent au client.
5. **Un Duel terminé ne se rouvre pas en Transcription.** Ses Actions ont été arbitrées ; le
   Match qu'il a produit se commente et se supprime comme un autre.

6. **La politique de jeu du Bot s'écrit dans gammonNet d'abord.** Elle est une fonction sans
   état — la décision à prendre entre, une Action sort — donc elle tient dans le périmètre de
   gammonNet, qui ne connaît toujours ni partie ni appelant ; la partie reste chez l'Arbitre.
   Elle est écrite en C en amont, avec sa spécification et sa mesure, puis portée ici sous le
   contrat de l'ADR-0011, avec un corpus de référence qui tient le port. Le Bot existe ainsi en
   WebAssembly dès le départ : un client peut le faire jouer dans le navigateur, où il est
   pour l'Arbitre un Côté externe comme un autre.

7. **Un Bot joue à un niveau nommé de gammonNet, à pleine force.** Les niveaux sont ceux de
   l'analyse (`instant`, `normal`, `thorough`) : le Bot joue avec la Configuration qui
   analysera son match, sans quoi la revue lui compterait comme erreurs ce qu'il a joué
   exprès. Un niveau affaibli — défini par le PR qu'il joue, tiré sur la perte d'équité — est
   une piste dont l'utilité reste à vérifier, tenue en amont (gammonNet #27) et absente d'ici.
8. **Les dés sortent d'un germe scellé avant le match et révélé après.** Un germe par Duel,
   tiré du système à la création ; chaque lancer est une fonction du germe et de son rang,
   donc ni une reprise après plantage ni une reprise de coup ne relance. L'Arbitre publie
   l'empreinte du germe à la création et le germe avec le Match terminé : les lancers se
   recalculent sans faire confiance à l'Arbitre. Le Bot ne reçoit que la position et le
   lancer courant ; le flux des dés ne quitte pas l'Arbitre. Absents : les dés saisis à la
   main, une source externe, et le germe combiné de deux Côtés externes — qui s'ajoute au
   sceau sans le défaire, le jour où un client en a besoin.
9. **Un Duel se joue dans les conditions d'un match réel.** Pendant son tour le Côté arrange
   son coup librement ; la validation est un geste explicite, qui arrête la durée et
   l'horloge, et après elle rien ne se reprend. Le moteur se tait tant que le Duel est en
   cours — ni panneau Eval, ni candidats, ni indice, ni tuteur : l'analyse se lance à la fin
   et la revue s'ouvre dessus. Sans cela le Performance Rating du Match ne mesurerait rien.
   L'Arbitre joue seul ce qui n'est pas une décision : le lancer quand le videau n'est pas
   disponible, le tour passé, le coup unique.
10. **Un Duel est un brouillon jusqu'à sa fin, et ne fabrique aucun résultat.** Il s'écrit
    après chaque Action et se reprend au même point, mêmes dés à venir, horloges arrêtées ;
    plusieurs peuvent être en suspens, un seul est ouvert. Terminé, il devient un Match —
    c'est le défaut, réglé à la création ; sinon le brouillon est jeté. **Un match en points
    s'écrit entier ou pas du tout** : arrêté avant la fin, il se met en suspens, s'abandonne
    ou se *jette* (rien du Duel n'est écrit ; une Position mise sur la Pile pendant le jeu y
    reste, parce que ce geste l'a écrite sur-le-champ, comme une position apportée seule), et
    l'Arbitre refuse de le *garder* tel quel (`ErrInvalid`), quelle que soit l'interface. Le
    panneau Matchs ne montre que des matchs entiers : un Match coupé, sans vainqueur ni score
    final, ne s'y distinguerait d'un match joué qu'à son origine, et la pause garde déjà le
    match pour le reprendre. Une session en argent n'a pas de fin à elle : l'arrêter et la
    garder est sa fin ordinaire, ses parties écrites telles quelles, celle en cours sans
    vainqueur. Arrêter le Duel n'est jamais céder la partie, qui reste une Action du jeu.
    Céder le match entier en est une aussi, l'abandon du match (ADR-0074) : le Match s'écrit
    gagné par l'adversaire, comme après une perte au temps (ADR-0073 règle 1). Le Match porte
    son origine — joué ici, niveau du Bot, Cadence, germe révélé, et le cas échéant perdu au
    temps — et le Côté délégué y porte le nom de sa Configuration. Un Côté externe peut
    *déclarer* à la création le Bot qui joue derrière lui (sa Configuration, le tag gammonNet) :
    l'origine le porte à part, déclaré par le client et non attesté, puisque l'Arbitre ne voit
    pas ce qui joue derrière un Côté externe et n'authentifie personne (ADR-0005) ; le niveau
    et la version du Bot restent ce que l'Arbitre a fait jouer lui-même. Sans nom donné, ce
    Côté prend celui de la Configuration déclarée, et l'Arbitre le traite en Côté externe. La marque « arrêté avant
    la fin » reste lue sur un Match qui la porte ; aucun Duel ne l'écrit plus, et aucune
    donnée n'est migrée. Les imports ne sont pas concernés : un fichier coupé s'importe tel
    qu'il est. Son Performance Rating compte comme celui d'un autre Match (règle 1).
11. **Hors du bureau, un Duel se pilote une Action à la fois.** Une famille `/v1/duels.*` sur
    le patron de l'ADR-0057 (session en mémoire, `If-Match`, flux d'événements), activée par
    `--duel`. Une Action nomme son Côté ; le démon n'authentifie personne (ADR-0005), le
    client répond de qui joue pour qui. Quand une Action donne le trait à un Côté délégué, le
    Bot joue dans la même requête : la réponse rend l'état au prochain point où un Côté
    externe décide, avec tout ce qui s'est passé entre-temps. Le délai « naturel » d'un Bot
    est une animation du client. Le germe ne sort par aucune route avant la fin, l'export
    compris : une base exportée n'emporte aucun Duel en suspens, quelle que soit l'interface
    qui l'exporte (bureau, CLI, démon, `call`). Un export part vers un lecteur, et le
    brouillon ne vaut que sous l'Arbitre qui en tient le germe ; l'emporter sans germe ferait
    un autre Duel, aux dés à venir changés (règle 10). Le Match d'un Duel terminé part avec
    son origine, germe révélé. Seule la migration vers PostgreSQL emporte les brouillons avec
    leur germe : elle déplace la base d'un même opérateur sans la livrer à personne, et le
    germe y reste scellé. La règle tient par construction : la liste des Duels du stockage
    (`DuelStore.List`) n'a ni germe ni document, seul `Get` les rend, à qui nomme un Duel.
    Qui paie le
    calcul du Bot se décide à la création — un Côté délégué, le démon calcule ; deux Côtés
    externes, le client le fait jouer ailleurs — sans réglage de plus, et sans évaluateur nu
    (ADR-0015). La CLI a `blunderdb duel`, sans interface interactive, et un Duel de deux
    Bots joué d'un seul appel ; chaque appel étant son propre processus, un Duel piloté par
    la CLI ou par `call` n'a ni Cadence ni durée de décision.
12. **Un Duel part d'un Départ, qui est une Position.** Par défaut la position initiale au
    début du match ; sinon la Position au plateau, ou la position initiale à un score
    choisi. La Position donne le plateau, le videau, le trait, le score — Crawford et argent
    compris — et, si elle en porte un, le premier lancer, que l'Arbitre ne tire donc pas.
    Seule la première partie commence au Départ ; les suivantes commencent à la position
    initiale, au score atteint. La machine de règles reçoit donc un état de départ, et le
    brouillon le porte. Un Départ que les règles n'admettent pas est refusé avec son motif,
    jamais corrigé. Le Match le porte dans son origine, et son export `.mat` est refusé : le
    format ne sait pas commencer une partie ailleurs qu'à la position initiale. À la
    création se règlent aussi la longueur (1 à 25 points, la table d'équité n'allant pas
    au-delà) ou une session en argent — Jacoby au choix, pas de beaver, videau plafonné à
    64, close par l'abandon du match ou, hors de l'interface graphique, par « arrêter et garder » —, le niveau du Bot, la Cadence (ADR-0073), le Côté joué et
    le nom du joueur.

## Conséquences

- gammonGo ne porte pas de moteur de jeu : il est un client de l'Arbitre.
- Le Duel attend une version de gammonNet : rien du Bot ne se livre ici avant que sa
  politique soit publiée en amont. Rejeté : l'écrire en Go d'abord et la remonter ensuite
  (comme le rollout) — le Bot du navigateur aurait attendu, et les niveaux de force auraient
  été mesurés deux fois.
- Les lieux qui disent « ce n'est pas un mode de jeu » (les questions d'entraînement jouées
  depuis une graine, ADR-0041) restent vrais par ce qu'ils n'ont pas : ni dés montrés, ni
  score, ni rien de conservé.
- Rejeté : une Transcription avec un drapeau « jouée » (chaque geste de correction à
  interdire un par un, deux concepts dans un objet) ; le jeu dans gammonGo seul (le bureau
  n'aurait rien) ; câbler « le côté 2 est le moteur » (à défaire le jour où deux humains
  jouent).
