# État d'exécution du plan 2026-09b

Dernière mise à jour : **2026-09-03**. Ce fichier est le point de reprise : il dit ce
qui est fusionné, ce qui attend dans une branche, et ce qui est bloqué sur une action
humaine. Le plan lui-même est dans [README.md](README.md) ; les fiches sont dans les
fichiers de lot, chacune avec le numéro de son issue GitHub.

## Ce qui est en place

- **145 issues** créées, `#155` à `#300`, une par fiche, avec le numéro repris dans le
  titre de la fiche. Cinq jalons (`Étape 0 — Colmater` … `Étape 4 — Fond`), labels
  `lot:X`, `effort:S|M|L` et un label de domaine.
- **Les quatorze recherches externes P5-P18 sont rendues** et versées sous
  [`docs/recherche/`](../../docs/recherche/README.md). Quatre ont corrigé une fiche :
  I.4 et I.5 réécrites (le format OGXM n'existe pas ; Heroes passe par XG), C.5 devenue
  une question de modèle amont, B.12 passée à zstd avec dictionnaire.
- **Le lot A (étape 0) est entièrement fusionné.**

## Fusionné sur `main`

| Fiche | Issue | Ce qui a changé |
|---|---|---|
| A.1 | #155 | Un `X-Tenant-ID` non numérique est refusé (400) au lieu de retomber sur le tenant 0. Amendement d'ADR-0005. |
| A.2 | #156 | `metadata.load/save/setVersion` hors de `/v1` ; l'état de session part dans `session_state`, sous RLS. **Schéma 2.17.0.** |
| A.3 | #157 | Le wrapper ouvre SQLite par `sqlite.DSN` : les clés étrangères s'appliquent à toutes les connexions. `verify` compte les orphelins. |
| A.4, A.5 | #158, #159 | Jeton de workflow en lecture, secret scanning, 52 actions épinglées par empreinte, ruleset anti-force-push, job de release isolé, seeds de fuzz rejoués. |
| A.6 | #160 | Sept sites ne renvoient plus l'erreur interne brute ; identifiants d'import opaques ; exemption de limite de corps restreinte aux vraies routes d'upload. |
| A.7 | #161 | Les 96 chaînes en retard traduites ; règle « un `.rst` = ses huit `.po` » dans le modèle de PR et `CLAUDE.md`. |
| A.8 | #162 | Base de démonstration régénérée : noms fictifs, collections, commentaires, paquet Anki, analyses gammonNet. `scripts/build-demo-db.sh`. |
| A.9, A.10 | #163, #164 | `THIRD_PARTY.md` installé par tous les paquets ; lien `/usr/bin/blunderdb`. |
| A.11 | #165 | Conteneur `.dbx` version 2 : en-tête authentifié par l'AEAD, lectures bornées, décompression bornée, paramètres Argon2id stockés. |
| A.12 | #166 | `/healthz` ne touche plus au stockage ; sous-commande `healthcheck` et `HEALTHCHECK` de l'image. |
| A.13 | #167 | Le filtre d'erreur retient l'erreur **maximale** au lieu d'une valeur dépendant de l'itération d'une table de hachage. |
| A.14 | #168 | BACKLOG re-daté, index des ADR complété, #119 fermée. |
| B.1, B.9 | #169, #177 | Versions majeures comparées numériquement ; migrations réparables ; `verify` diffe le schéma. |
| B.2, B.7 | #170, #175 | Crawford propagé à la conversion GnuBG ; import qui refuse un plateau à plus de quinze pions ; erreur de position dupliquée nommée. |
| B.4 | #172 | Révision Anki transactionnelle ; `pkg/blunderdb/anki.ScheduleNext` partagé ; note hors bornes refusée (elle faisait paniquer le démon). |
| C.1 | #188 | Tests moteur manquants : valuation terminale, propriétés de la MET, deux cibles de fuzz. Gold régénéré depuis le C amont v1.2.1. |
| D.1 | #201 | Six corrections d'ergonomie, plus la régénération de `frontend/wailsjs` sans laquelle l'application ne montait plus sous Vite. |
| H.1, H.2 | #243, #244 | Page d'installation complète, bloc d'installation injecté dans les notes de release, `CONTRIBUTING.md`, code de conduite, section communauté. |

## Branches en cours au moment de l'arrêt

Chacune est un worktree `../blunderDB-<nom>` sur la branche `feat/<nom>`, partie de
`main`. Pour reprendre : lire le rapport de l'agent s'il existe, sinon relire la fiche,
finir, puis fusionner depuis la racine (`git merge --no-ff`), vérifier
`go build && go vet && go test`, et supprimer le worktree.

| Branche | Fiches | Issues |
|---|---|---|
| `feat/b3-b5-b17-schema` | B.3, B.5, B.17 — vague de schéma **2.18.0** : Jacoby et beaver hors du hash Zobrist (ADR d'amendement à écrire), UNIQUE sur `analysis`, `zobrist_hash NOT NULL`, contraintes `CHECK`, clés étrangères d'`anki_review_log` | #171, #173, #185 |
| `feat/c3-money-match` | C.3, C.6 — cohérence money/match, libellé « Équité (money) / (match) », `race.Money` → `CubeVerdict`, une seule règle de verdict | #190, #193 |
| `feat/c4-b6-lot-erreurs` | C.4, B.6 — le lot ne retente plus l'inévaluable, `--stale` déclenchable, erreurs SQL propagées au lieu d'être avalées | #191, #174 |
| `feat/ci-race-couverture` | C.2, E.1, E.2 — détecteur de courses sur le moteur, gold en nocturne, `test-os` bloquant, couverture mesurée juste avec un plancher | #189, #217, #218 |
| `feat/g1-g2-deploiement` | G.1, G.2 — compose avec proxy authentifiant, arguments positionnels refusés, rate limit par défaut | #229, #230 |
| `feat/e4-e5-hygiene-tests` | E.4, E.5 — fixtures qui ne sautent plus en silence, paquet `tests/` redistribué, `gofmt` appliqué, hooks versionnés, `make check` aligné | #220, #221 |
| `feat/h6-terminologie` | H.6 — « Bearoff » → « Eval » et diffusion contrôlée dans l'aide intégrée, neuf langues | #248 |
| `feat/d2-d6-d11-front` | D.2, D.6, D.11 — bascule réelle des panneaux, état d'erreur nommé du panneau Eval, XGID encodé sans perte | #202, #206, #211 |
| `feat/c5-cube-efficiency` | C.5 — l'efficacité du videau tranchée par **ADR-0028** : divergence de modèle assumée (pas un trou de port, le C fait pareil), correctif proposé pour l'amont gammonNet, instrument de mesure et réplique en profondeur des deux cas rouges de la porte. Aucun changement de comportement | #192 |

## Bloqué sur une action humaine

1. **`gh` ne s'authentifie plus** : jeton dans un trousseau que D-Bus ne déverrouille pas,
   toute commande finit en `HTTP 401` ou se bloque. Relancer `gh auth login`. Sans lui :
   - les issues traitées ne peuvent pas être fermées (les commits portent leur `Closes #N`,
     donc elles se fermeront au prochain push) ;
   - la description et les sujets du dépôt ne sont pas posés (fiche H.2, commande exacte
     notée dans `lot-H-docs-distribution.md`) ;
   - la version **0.35.1**, que la fin du lot A appelle, n'est pas publiée.
2. **`/tmp` est un tmpfs de 7 Go saturé** : des éditions de liens Go ont échoué en
   « disk quota exceeded ». Exporter `GOTMPDIR` et `TMPDIR` vers `~/.cache/gotmp`, ou
   faire du ménage.
3. **Playwright** : les navigateurs installés ne correspondent pas à la version attendue
   par le paquet, et `playwright install` se bloque. Contournement en place par liens
   symboliques dans `~/.cache/ms-playwright/`.

## Suite

L'étape 1 est aux trois quarts. Ce qui reste à lancer, une fois les branches ci-dessus
fusionnées : B.8 (codes de retour et sortie machine de la CLI), D.3, D.4, D.5 (parseur de recherche
unique, navigation au clavier, budget de warnings), E.3 (`t.Parallel`, à faire seul car
il touche toute la suite), G.3, G.4, G.6, G.7 (verrou de migration PostgreSQL, plafonds,
délais, tests d'isolation), H.3, H.4, H.5 (canaux de distribution, tutoriels, captures).
Puis l'étape 2, et le lot I pour le produit.

**Un passage amont gammonNet est désormais à ouvrir** : ADR-0028 point 4 décrit le
correctif (`cube_x` indexé par le propriétaire LOCAL, spec §4 et §8 step 2) que C.5 a
tranché mais qui ne peut pas être écrit ici. Il rejoint la forme close de `levelSolve`
(C.7) dans la file amont ; les deux régénèrent les golds et périment les analyses
stockées, donc autant les faire dans le même tag.
