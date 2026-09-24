# Les personas et les scénarios

[README.md](README.md) dit pourquoi ; ce fichier dit **qui** dirige et **quoi**. Chaque
scénario est un tournoi vraisemblable, joué par le moteur ([outillage.md](outillage.md)
§ 3) puis parcouru à l'écran par son persona ([mesure.md](mesure.md)). Les noms de joueurs
sont **fictifs** (règle #162 : jamais de personne réelle dans une base de démonstration).

## 1. Six personas

Les quatre premiers viennent de D16 ([decisions.md](../decisions.md)) et les budgets
`ux.md` § 4 sont écrits pour eux. Les deux derniers sont ajoutés pour les scénarios que les
quatre premiers ne couvrent pas.

| Persona | Qui | Ce qui compte pour lui ou elle | Scénarios |
|---|---|---|---|
| **Marc** | TD de club, 16-24 joueurs, seul, interrompu sans cesse | 1-2 clics par geste courant ; reprendre où il en était | S1 |
| **Sophie** | TD d'un grand tournoi BMAB, 48-64 joueurs, deux jours, une assistante, transcription obligatoire, dotation, projection | rigueur, corrections tracées, imprimable, rien de perdu | S2, S5 |
| **Yanis** | remplaçant, ne connaît pas le format, prend la salle en cours | l'interface dit toujours quoi faire ensuite ; une erreur se répare | S1 (2e moitié) |
| **Léa** | TD-joueuse | tout en deux minutes entre ses matchs ; « où en étais-je » | S2 |
| **Nadia** *(nouvelle)* | responsable du championnat de club : une ronde par lundi soir pendant six semaines | une ronde annoncée à l'avance avec sa date ; des absents prévus ; des résultats reçus par message entre deux lundis ; le classement affiché au club le mardi | S4 |
| **Karim** *(nouveau)* | directeur d'un festival : trois épreuves sur un week-end, 14 tables partagées, un assistant à la saisie, page murale projetée, rondes annoncées au micro | savoir à tout moment qui est libre pour quelle épreuve ; ne jamais mettre deux matchs sur la même table ; annoncer une ronde en une phrase | S3 |

## 2. Les incidents injectés dans chaque scénario

Les mêmes six incidents, **à chaque scénario**, à un moment plausible de son déroulé ; le
rapport dit pour chacun le chemin trouvé, son coût en gestes, et ce que le persona aurait
voulu faire à la place.

| Incident | Ce qui se passe | Ce que le code prévoit aujourd'hui (à vérifier) |
|---|---|---|
| I1 joueur qui s'en va | un joueur quitte la salle, en cours de match ou entre deux | retrait immédiat (forfait de son match en cours) ou différé (« après son match ») depuis la vue Joueurs |
| I2 retardataire | un joueur arrive après le lancement | inscription avec choix d'une place d'exemption libre, sinon entrée à la phase suivante ; jamais de retirage |
| I3 résultat faux corrigé plus tard | le TD a cliqué le mauvais vainqueur, s'en aperçoit trois matchs après | `LastDecision` ne sert que pour le dernier ; sinon Historique → Corriger → fiche sur la grille ; réparation de tableau proposée dans la file |
| I4 table cassée | un plateau devient inutilisable en cours de ronde | tables indisponibles dans Réglages (en cours : Enregistrer + confirmation) ; changer un match de table par ⋯ |
| I5 coupure | l'application est fermée brutalement en cours de ronde | rejeu à l'ouverture, rien n'est stocké hors journal (F17) |
| I6 absence prévue | un joueur prévient qu'il ne sera pas là pour la ronde suivante, sans quitter le tournoi | **rien de dédié** : retrait différé puis réinscription, ou bye manuel ; à mesurer |

## 3. Les cinq scénarios

Configurations en termes du moteur (`swiss_lives`, `bracket`, `lives_bracket`,
`round_robin`, `gsl`) et des préréglages nommés de `directionStore.js` (`suisse_tableau`,
`elimination`, `consolante`, `double_elimination`, `gsl`, `poules`).

### S1 — Le soir de club (Marc, puis Yanis)

- **20 joueurs**, un vendredi de 19 h 30 à 23 h 30, 8 tables, matchs en 5 points.
- Préréglage `suisse_tableau` tel quel : suisse 2 vies continu, bascule sur tableau à
  exemptions, finale en 7.
- **Déroulé** : Marc crée le tournoi, reprend 16 inscrits du tournoi du mois dernier
  (annuaire), en tape 4 nouveaux, lance ; à 21 h il part (urgence), **Yanis** prend le
  portable sans rien connaître : il doit trouver seul comment saisir un résultat, lancer les
  propositions, retirer un joueur, clore.
- **Incidents** : I1 à 20 h 40 (un joueur part après sa deuxième défaite, sans prévenir),
  I2 à 19 h 50, I3 vu à 21 h 20 par Yanis, I5 à 22 h.
- **À prouver** : le coût d'entrée `ux.md` § 4.2 (≤ 90 s de « Nouveau » à la première
  ronde, ≤ 30 s avec l'annuaire, ≤ 8 clics hors noms) tient à 20 joueurs sur le vrai
  moteur ; Yanis clôt sans lire la doc ; le classement rejoué après I5 est identique.

### S2 — Le week-end à une épreuve (Sophie, Léa)

- **50 joueurs**, samedi 10 h → dimanche 18 h, 14 tables, matchs en 7 points, pause repas
  12 h 30-14 h et 19 h 30-20 h 30, arrêt le samedi à 23 h.
- Suisse 2 vies continu avec bascule vers un **tableau 16** (`target`) en 9 points, finale
  en 11. **La consolante n'est pas prévue au départ** : Sophie l'ajoute le samedi soir
  (F15, phase ajoutée après la courante, `config_changed` en cours) pour les éliminés.
- Sophie a une assistante : l'assistante saisit les résultats, Sophie décide. La
  simulation ne modélise pas deux postes (il n'y en a qu'un) ; elle mesure ce que cela
  coûte quand deux personnes se relaient sur un seul portable.
- **Léa** joue : entre ses matchs elle revient au portable, doit reprendre en ≤ 10 s
  (F18) : lire ce qui s'est passé depuis son dernier geste, confirmer les propositions,
  saisir ce qu'on lui apporte.
- **Incidents** : I1 (un joueur part le samedi 22 h en plein tableau 16 : que devient sa
  place ?), I2 (un joueur à 11 h 15, après 30 matchs lancés), I3 (score inversé dans le
  tableau : la réparation proposée doit apparaître), I4 (table 7 cassée à 16 h), I5 (samedi
  17 h), I6 (un joueur prévient qu'il ne revient pas dimanche matin mais dimanche 14 h).
- **À prouver** : la grille de 14 tables et la file de 20 propositions tiennent sans
  défilement à 1366×768 ; la fin estimée est plausible ; la pause repas est respectée
  ou son avertissement lisible ; la page murale suit ; « clore » un dimanche soir donne le
  classement par section et les prix.

### S3 — Le festival à trois épreuves (Karim)

- **Principal 64 joueurs** (`double_elimination` ou `consolante`, 7 points, finale 11),
  **speed 32 joueurs** (`elimination`, 3 points, samedi soir 20 h-23 h), **doubles 16
  paires** (`elimination`, 5 points, dimanche matin). 14 tables partagées entre les trois.
- Modélisation (décision Q2) : **trois Tournaments, trois Directions**, ouvertes tour à
  tour. Une paire de doubles est **un Participant dont le nom est « A / B »** : le moteur
  n'a pas de notion de paire, et c'est un constat attendu du rapport (nom du Player porté par
  les Matchs, cote d'entrée d'une paire, annuaire).
- 40 des 64 joueurs du principal jouent aussi le speed ; 20 jouent les doubles.
- **Déroulé** : samedi 10 h lancement du principal ; 20 h le speed démarre pendant que le
  principal continue (les joueurs libres du principal jouent le speed ; les tables sont
  partagées) ; dimanche 10 h les doubles ; dimanche 15 h finales.
- **Incidents** : I1 (un joueur part samedi 21 h : il est dans les deux épreuves), I2, I3,
  I4 (deux tables cassées, une par épreuve, à la même heure), I5, I6.
- **À prouver, et surtout à mesurer** : combien de gestes pour **savoir qui est libre**
  pour le speed alors que le principal tourne (deux Directions, une seule ouverte) ;
  combien de fois deux matchs se retrouvent sur la même table faute de salle partagée ;
  le coût d'**annoncer une ronde** (feuille d'appariements par épreuve, page murale par
  épreuve : trois pages murales, trois dossiers ?) ; combien de fois le TD est interrompu
  pour « je joue où ? » (Q6, l'issue de cadrage de l'écran joueur).

### S4 — Le championnat de club en rondes (Nadia)

- **24 joueurs**, une ronde chaque lundi soir pendant **six semaines**, 6 tables, matchs en
  7 points, une seule phase `swiss_lives` en mode **`rounds`** (rondes synchrones), sans
  bascule ; classement final par victoires et vies.
- **Déroulé** : lundi 1 : inscription et ronde 1. Entre deux lundis : deux matchs en
  retard joués chez l'un ou l'autre, résultats reçus par message ; Nadia les saisit le
  jeudi. Lundi 2 : ronde 2 annoncée **le vendredi précédent** (feuille d'appariements
  datée) ; trois absents prévus (I6) ; un absent non prévu (I1 ?) ; un nouveau joueur en
  semaine 2 (I2). Chaque mardi : classement affiché au club (page murale ? CSV ? capture ?).
- **Incidents** : I6 est le cœur du scénario, à chaque ronde ; I3 sur un résultat de la
  ronde 2 découvert en semaine 4 ; I5 entre deux lundis (la base est fermée six jours) ;
  I4 sans objet (noter « sans objet », pas « non mesuré »).
- **À prouver, et à mesurer** : le mode `rounds` propose-t-il la ronde N+1 tant qu'un match
  en retard de la ronde N n'est pas saisi ? Que coûte un joueur absent une ronde sans être
  retiré (retrait différé + réinscription ? bye manuel ? rien ?) ; existe-t-il un moyen
  d'**annoncer une ronde datée** trois jours avant qu'elle soit jouée (la feuille
  d'appariements n'est produite que pour des matchs lancés) ; le classement du mardi
  s'obtient-il sans le portable de Nadia (CLI `tournament standings`, page).

### S5 — Une épreuve sur cinq jours (Sophie)

- **50 joueurs**, du lundi au vendredi, sessions 9 h-23 h avec pauses 12 h 30-14 h et
  19 h-20 h 30, 10 tables, matchs en 11 points.
- `swiss_lives` continu à **3 vies** (si le moteur l'accepte sans bascule : `Target` exige
  2 vies, donc pas de bascule), **micro-rondes** `batch_minutes = 20` pour éviter le choix
  de l'adversaire, puis tableau à exemptions le vendredi décidé en cours (F15).
- **Déroulé** : cinq ouvertures et fermetures de la base ; chaque matin, « depuis hier
  soir » ; joueurs qui ne jouent que certains jours (I6, avec la variante « je reviens
  jeudi »).
- **Incidents** : les six, dont I5 chaque soir (fermeture normale, qui doit valoir I5), et
  I1 un mercredi (départ définitif à 3 victoires : que vaut son classement ?).
- **À prouver** : la **fin estimée** sur plusieurs jours (le moteur n'a pas de jour : une
  fin estimée « demain 3 h 12 » est-elle affichée ainsi ?) ; les micro-rondes à l'échelle
  (file d'attente de 20 joueurs libres, compte à rebours) ; la bande d'horloge quand
  « temps écoulé depuis le premier match » vaut quatre jours ; le journal de ~400
  événements rejoué sous le budget de 50 ms (`BenchmarkOpen` à comparer).

## 4. Ce que chaque scénario doit laisser derrière lui

- Le **journal Nicomaque** exporté (`blunderdb tournament export`) et la base SQLite de
  fin, dans le scratchpad, nommés `S<n>-<étape>.db` ; pas dans le dépôt.
- La **liste des opérations** du persona avec leur coût ([mesure.md](mesure.md) § 3) et,
  pour chaque incident, la ligne « voulu / trouvé / fait à la place / gestes ».
- Les **captures** des états où un défilement, une modale ou une confirmation apparaît,
  aux deux viewports.
- `blunderdb tournament verify` sans avertissement résiduel à la fin, ou la liste de ceux
  qui restent et pourquoi.
