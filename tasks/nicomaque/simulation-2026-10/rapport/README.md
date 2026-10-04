# Simulation 2026-10 — rapport (index)

Protocole de #380 exécuté par simulation : trois tournois joués **par l'interface réelle** (front
Svelte servi par Vite, vrai `*database.Database`, vrai moteur Nicomaque, vraie SQLite, par le
shim `outils/_shim/`), mesurés geste par geste (`outils/e2e/meter.js`), et dont le classement
est recalculé sans code de l'application par la feuille papier (`outils/oracle/oracle.py`).
Personas, incidents et modèle d'interruptions : [../scenarios.md](../scenarios.md). Outillage :
[../outils/README.md](../outils/README.md).

Toutes les mesures sont à l'échelle 100 % : T1 avait contourné le défaut de harnais UIScale,
T2 et T3 ont été rejoués après sa correction (voir le piège UIScale dans l'outillage).

| Rapport | Contenu |
|---|---|
| [T1.md](T1.md) | 16 joueurs, Sophie (P1) puis Hélène (P4, clavier seul, 150 %) |
| [T2.md](T2.md) | 32 joueurs, Yanis (P2), incidents dont SIGKILL et reprise |
| [T3.md](T3.md) | Rencontre à deux épreuves, Léa (P3), N26 et sorties de page |
| [ecarts.md](ecarts.md) | 25 écarts consolidés (6 bloquants, 14 d'ergonomie, 5 de confort), avec critère de correction, synthèse par persona et défauts de harnais |
| [adr-0047-amendement.md](adr-0047-amendement.md) | projet d'amendement de l'ADR-0047 qui tranche #457 (écran des joueurs) |

## 1. Les trois tournois

| | Format | Persona | Fini ? | Oracle = app | Particularité |
|---|---|---|---|---|---|
| **T1** | 16 joueurs, suisse 2 vies continu → tableau à exemptions (`suisse_tableau`) | Sophie (P1) ; Hélène (P4) rejoue l'entrée, l'inscription, des lancements et des résultats | oui, par l'interface | **oui, 16/16** (écran, API, CSV) | chemin nominal ; 27 matchs, 4 exemptés |
| **T2** | 32 joueurs, même format | Yanis (P2) | oui | **oui, 33/33** (32 + 1 retardataire) | retard, forfait + retrait, erreur corrigée tard, SIGKILL en pleine ronde ; 62 matchs |
| **T3** | Rencontre : A 16 joueurs poules → tableau, B 8 joueurs suisse 2 vies → tableau, 8 tables partagées | Léa (P3), 1366×768, dock ouvert | oui, les deux épreuves | **A : oui, 16/16 en variante moteur de N26** (non en variante repêchage, que Léa voulait) ; **B : oui, 8/8** | N26, quatre sorties de page ; 46 matchs |

**Reprise de T2 après SIGKILL** (6 matchs en cours) : 96 événements avant et après, état de
l'écran identique hors chronomètre, aucun avertissement ni de la Direction ni de la page murale,
page murale juste sans réécriture ; `tournament verify` en fin de tournoi : 164 événements,
aucun avertissement. La reprise est intacte ; seul le retour coûte (5 clics, E14).

## 2. Interruptions des joueurs (P5)

« Je joue où ? » : une par joueur affecté à un match, sans demander s'il lit sa table au mur.
« Est-ce que je joue ? » : exemption, retrait, élimination. V0 = sans page murale ; V1 = avec la
page telle que le persona l'a configurée (délai de rafraîchissement, 30 s au plus, compté à part).

| Tournoi | Joueurs affectés | « Je joue où ? » V0 | V1 | « Est-ce que je joue ? » V0 | V1 | Ce qui reste en V1 |
|---|---|---|---|---|---|---|
| T1 | 54 | 54 | **0** | 19 | 4 | 4 exemptés que la page ne nomme pas (E9) |
| T2 | 124 | 124 | **16** | 21 | 21 | 16 avant que Yanis trouve la page (E8) ; éliminés, retirée et exemptés jamais annoncés (E9) |
| T3 | 92 | 92 | **4** | 25 | 13 | 2 matchs lancés sans table, absents du mur (E4) ; 3 exemptions, 10 éliminations de phase (E9) |
| **Total** | 270 | 270 | **20** | 65 | 38 | |

Une page murale configurée et une table pour chaque match lancé donnent 0 « je joue où ? » :
chaque interruption restante a une cause nommée dans les écarts (découverte de la page, match
sans table). « Est-ce que je joue ? » n'est presque pas réduit, faute du mot « exempt » ou
« éliminé » sur la page.

## 3. Synthèse par persona

| Persona | Gestes (clics / frappes / défilements) | Pire friction | Blocages |
|---|---|---|---|
| **P1 Sophie** (T1, 1920×1080) | 82 / 366 (≈ 350 de saisie) / 0, 63 actions | le silence : 25 actions sans retour (inscriptions, Phase suivante, Tirage, Clore, page murale, E20) et la répétition (2 clics par résultat) ; focus perdu 38 fois (E6) | E7 : clos par la file, le tournoi reste « en cours » |
| **P2 Yanis** (T2, 1920×1080) | 234 / 644 (592 d'inscription) / 0, 171 actions, 6 hésitations | forfait + retrait : 19 gestes, deux écrans, deux « Supprimer » (E10) ; puis corriger un ancien résultat (14 gestes, E11) et trouver la page murale (6 clics, E8) | E7 : clos par la file, le tournoi reste « en cours » |
| **P3 Léa** (T3, 1366×768) | 235 / 497 (452 d'inscription) / 107 crans (104 imposés pour atteindre une cible), 160 actions | un rechargement ferme la Direction et perd tout (6 clics, E14) ; le panneau Rencontre hors écran (E17) ; le piège de la Bascule (E16) | N26 : repêchage impossible (E5) ; match lancé sans table (E4) ; au 1er passage, un joueur dans deux matchs (E3) ; « Clore » laisse l'état à `running` (E7) |
| **P4 Hélène** (T1, clavier, puis 150 %) | 2 clics forcés / 530 (292 Tab) / 0, 27 actions | 62 Tab pour « Lancer » ou une case de table (E12) | ouvrir un tournoi : double-clic seul (E2) ; Tab sur body bascule sur la Recherche (E1) — contournés en sortant du clavier ou en connaissant un raccourci |
| **P5 joueurs** | — | « est-ce que je joue ? » jamais réduit en T2 (21) | aucun joueur réel (voir § 4) |

Un match coûte 2 clics de résultat + le lancement : 1 clic seul, ou « Tout lancer + Confirmer »
pour une ronde (T1) ; 3 clics depuis la salle d'une Rencontre (T3).

## 4. Ce que la simulation ne prouve pas

- **Aucun joueur réel.** Les interruptions sont comptées par un modèle : le joueur lit sa table
  si son nom est écrit à côté d'un numéro dans le fichier de la page murale. Ni la distance à
  l'écran, ni la lecture d'une page qui tourne entre deux vues, ni l'attente du rafraîchissement
  (jusqu'à 30 s), ni le joueur qui demande quand même ne sont mesurés.
- **Aucun directeur réel.** Les personas suivent un script ; les hésitations de Yanis sont
  déclarées, pas observées. Le temps réel d'une saisie, la fatigue, le bruit de la salle,
  l'arbitrage d'un litige ne sont pas mesurés.
- **Pas de vraie fenêtre Wails** : dialogues, presse-papiers et fermeture de fenêtre sont figés ;
  la coupure de T2 laisse la page vivante, ce que l'application ne ferait pas.
- **Appariements non reproductibles** (`pairing: random`) : chaque passage joue d'autres
  matchs ; T3-E1 (un joueur dans deux matchs) n'a été vu qu'une fois sur trois.
- **Hélène à 150 %** n'a rejoué qu'une partie du tournoi ; le masque du dock (E19) n'est pas
  rejoué.

**#380 reste ouverte** : son critère est un vrai tournoi de club dirigé. La simulation montre
que la Direction mène trois tournois jusqu'au bout avec un classement juste et une reprise
intacte, et elle donne la liste des écarts à traiter avant ce tournoi réel.

## 5. Décision à éclairer : #457 (écran des joueurs)

#457 doit rouvrir, après #380, la question d'un écran des joueurs (téléphone, route du démon)
fermée par l'ADR-0047, avec le compte réel des interruptions. Ce que la simulation apporte :

- **« Je joue où ? » ne justifie pas, à lui seul, un écran des joueurs.** La page murale, une
  fois configurée, en supprime 250 sur 270 ; les 20 restantes viennent de deux défauts
  corrigibles côté Direction : la page qu'on ne trouve pas (E8, 16) et le match lancé sans
  table (E4, 4).
- **« Est-ce que je joue ? » est le vrai reste** : 38 sur 65 en V1. La page murale peut le
  porter (E9 : écrire « exempté », « éliminé », « qualifié »), sans écran personnel.
- **Ce qu'un écran personnel apporterait et que la page ne peut pas** : l'annonce sans attendre
  le rafraîchissement, et la lecture hors de la salle. La simulation ne sait pas le chiffrer.
- **Décision proposée** (l'utilisateur a jugé que la simulation suffit pour trancher) : pas
  d'écran personnel ; la page murale devient l'écran des joueurs, à condition de corriger E4, E8
  et E9. Projet : [adr-0047-amendement.md](adr-0047-amendement.md).
