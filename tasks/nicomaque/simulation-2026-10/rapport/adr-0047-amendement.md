# Projet d'amendement de l'ADR-0047 : l'écran des joueurs est la page murale

Statut : **intégré** dans
`docs/adr/0047-directing-a-tournament-creates-its-matches-before-they-are-played.md`, avec les
chiffres de la [contre-épreuve](contre-epreuve.md). Ce fichier garde le projet tel que relu.
L'utilisateur a jugé que la simulation de #380 suffit pour trancher ; les chiffres viennent de
[README.md](README.md) § 2 et de [ecarts.md](ecarts.md).

## Ce que l'ADR-0047 dit aujourd'hui

« Hors périmètre : l'écran des joueurs. blunderDB écrit une page HTML autonome à projeter ou
imprimer ; rien dans le front web embarqué (ADR-0039 règle 1). » #457 demandait de rouvrir la
question (téléphone du joueur, route du démon) une fois les interruptions réellement comptées.

## Ce qui a été compté

Trois tournois joués par l'interface réelle (T1 16 joueurs, T2 32 joueurs avec incidents,
T3 Rencontre à deux épreuves sur 8 tables partagées). Une interruption = un joueur qui doit
demander au directeur. V0 = sans page murale ; V1 = avec la page telle que le persona l'a
configurée.

| | « Je joue où ? » | « Est-ce que je joue ? » | Total |
|---|---|---|---|
| V0, sans page | 270 | 65 | **335** |
| V1, page murale | 20 | 38 | **58** |

Les 58 qui restent ont **trois causes, toutes dans la Direction** :

| Cause | V1 | Écart | Ce qu'un écran personnel y changerait |
|---|---|---|---|
| La page n'est pas trouvée : P2 l'ouvre après 8 lancements | 16 | E8 | rien : un écran personnel aussi doit être trouvé et ouvert |
| Un match est lancé sans table, donc absent du mur | 4 | E4 | rien : le joueur n'aurait pas non plus de table à lire |
| La page ne dit ni « exempté », ni « éliminé », ni « qualifié » | 38 | E9 | rien : il faudrait écrire la même information |

Aucune des 58 n'est due à ce qu'un écran personnel est seul à pouvoir faire. Ces deux apports
sont l'annonce immédiate (le mur se rafraîchit toutes les 30 s ; le délai est compté à part et
ne crée aucune interruption) et la lecture hors de la salle.

## Amendement proposé

Remplacer le paragraphe « Hors périmètre » de la Décision par :

> **L'écran des joueurs est la page murale.** blunderDB écrit une page HTML autonome par
> Direction ou par Rencontre (ADR-0056), à projeter ou imprimer ; aucun écran personnel
> (téléphone, compte de joueur, route du démon dédiée) et rien dans le front web embarqué
> (ADR-0039 règle 1). Les routes `/v1/` servent un client externe (ADR-0057). La page doit
> suffire à répondre aux deux questions d'un joueur sans aller voir le directeur :
>
> 1. **« Je joue où ? »** Chaque match en cours y figure avec sa table. Un match sans table ne
>    se lance pas : il attend une table (E4).
> 2. **« Est-ce que je joue ? »** Exempté (avec le tour d'entrée), éliminé, qualifié y sont
>    écrits en toutes lettres à la fin de chaque phase (E9).
>
> La page porte ce nom dans l'interface (« page murale ») et se trouve depuis l'en-tête de la
> Direction et depuis la salle ; à chaque écriture, l'interface dit où est le fichier (E8).

Ajouter aux Conséquences :

> - Écarté, après mesure (#380, #457) : un écran personnel. Sur trois tournois simulés, la page
>   murale ramène les interruptions de 335 à 58, et les 58 restantes viennent de trois défauts
>   de la page (E4, E8, E9), qu'un écran personnel aurait aussi. On rouvre la question
>   seulement si, ces trois défauts corrigés, un tournoi réel compte encore des interruptions
>   que le mur ne peut pas éviter (délai de rafraîchissement, joueur hors de la salle).

Et compléter la Garde : un test du HTML de la page (`pkg/blunderdb/direction/wallpage.go`,
`service/rencontre_page.go`) qui exige la table de chaque match en cours et les mentions
« exempté », « éliminé » et « qualifié ».

## Ce que l'amendement ne décide pas

- Le délai de rafraîchissement (30 s fixes) et la page qui fait tourner deux vues : du confort
  de la page, pas une question d'écran.
- #380 reste ouverte : son critère est un vrai tournoi de club dirigé. L'amendement n'en dépend
  pas.

## Effet sur #457

Les deux critères sont remplis : les interruptions sont comptées (simulation de #380), et la
décision est écrite (cet amendement, une fois accepté). Les travaux qui en découlent sont les
issues de E4, E8 et E9.
