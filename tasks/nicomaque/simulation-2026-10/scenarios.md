# Simulation 2026-10 — personas, tournois, incidents, mesure

Exécution du protocole de #380 **par simulation** (décision de l'utilisateur : des tournois
simulés tiennent lieu du tournoi de club). Le produit n'est pas modifié ; un défaut trouvé
devient une ligne d'écart dans [rapport/ecarts.md](rapport/ecarts.md).

Tout se joue sur **la vraie Direction** : le front Svelte réel, servi par Vite, branché par
HTTP sur un vrai `*database.Database` (shim `outils/shim/`, vrai Nicomaque, vraie SQLite).
Aucun `installWailsMock` : les liaisons `main.Config`, `gui.App` et `runtime` sont écrites à
neuf dans `outils/e2e/realBackend.js`.

## 1. Personas

Ceux de la simulation 2026-09 ([scenarios.md](../simulation-2026-09/scenarios.md) § 1) sont
repris ; les rôles P1-P5 de la commande s'y rattachent, deux personas sont ajoutés.

| Rôle | Persona | Contraintes de jeu | Viewport | Tournoi entier |
|---|---|---|---|---|
| P1 directeur expérimenté, pressé | **Sophie** | souris + clavier, le moins de gestes, cherche les raccourcis | 1920×1080 | T1 |
| P2 bénévole novice | **Yanis** | lit les libellés, se trompe (mauvais score, mauvaise ronde) et doit annuler ; on compte ses hésitations (écrans ouverts pour rien) | 1920×1080 | T2 (avec ses incidents) |
| P3 directrice-joueuse, portable | **Léa** | 1366×768, dock ouvert, quitte la page entre ses matchs et y revient : retrouve-t-elle son état ? | 1366×768 | T3 |
| P4 accessibilité *(nouvelle)* | **Hélène** | clavier seul (aucun clic), puis zoom 150 % ; tout atteignable, focus visible | 1366×768 @150 % | T1 (rejoué) |
| P5 joueur *(nouveau)* | **les joueurs** | consultent la page murale ; chaque joueur qui ne peut pas savoir sa table sans demander est une interruption « je joue où ? » (#457) | — | T1, T2, T3 |

Chaque action est mesurée aux deux viewports 1366×768 et 1920×1080 (cible hors écran,
masquée par le dock), quel que soit le viewport du persona.

## 2. Les tournois

| | Joueurs | Format | Persona | Incidents |
|---|---|---|---|---|
| **T1** | 16 | suisse 2 vies continu → tableau à exemptions (préréglage `suisse_tableau`) | Sophie (P1), rejoué par Hélène (P4) | aucun : le chemin nominal |
| **T2** | 32 | suisse 2 vies continu → tableau à exemptions | Yanis (P2) | retard (I2), forfait (I1), résultat saisi par erreur puis corrigé (I3), **fermeture brutale du processus en pleine ronde** puis reprise (I5) |
| **T3** | Rencontre (ADR-0056) : épreuve A 16 joueurs poules → tableau (préréglage `poules`), épreuve B 8 joueurs suisse 2 vies | Léa (P3) | **N26** : un qualifié de poule se retire avant le tableau ; Léa quitte la page et revient trois fois |

Les résultats sont tirés par une graine fixe à partir des cotes ; la feuille papier note
chaque appariement et chaque résultat **voulu** par le directeur, indépendamment de ce que
l'écran affiche.

## 3. La feuille papier oracle

`outils/oracle/oracle.py` recalcule le classement final depuis la feuille JSON écrite par la
spec, sans code de l'application (réimplémentation des règles documentées du moteur). Le
critère de #380 : classement de l'application **identique** à celui de la feuille. Pour T2,
on vérifie en plus qu'après la fermeture brutale le rejeu de la Direction redonne le même
état et **aucun avertissement résiduel** (`State.Warnings`, `blunderdb tournament verify`).

## 4. La mesure, geste par geste

Chaque action d'un directeur passe par `outils/e2e/meter.js`, qui enregistre en JSONL :
persona, action, clics, frappes, crans de molette et pixels défilés par conteneur (scrollTop
remis à 0 avant chaque action, comme la mesure 2026-09 l'a appris ; cible atteignable
seulement en défilant un conteneur `overflow:hidden` = écart), cible hors écran ou masquée
par le dock à 1366×768 et 1920×1080, focus perdu après l'action, cible < 24 px, texte
tronqué, retour visuel absent. Les écarts ont la forme : ce que le directeur voulait, ce que
la page proposait, ce qu'il a fait à la place, nombre de gestes.

## 5. Le modèle des interruptions « je joue où ? » (P5)

À chaque **affectation** d'un joueur à un match (proposition lancée, exemption, match du
tableau), le joueur cherche sa table. Il la sait **sans demander** si et seulement si la page
murale est configurée, projetée, et contient, au moment où le match est lancé, son nom à côté
d'un numéro de table (contrôlé dans le fichier écrit par l'application) ; le délai de
rafraîchissement de la page (`minRefresh`) est compté à part. Sinon c'est une interruption.
Deux variantes sont comptées par tournoi : **V0** sans page murale (ce que fait un directeur
qui ne la découvre pas) et **V1** avec la page telle que le persona l'a configurée.
Une exemption, un retrait ou une élimination produisent aussi « est-ce que je joue ? » :
comptés séparément.
