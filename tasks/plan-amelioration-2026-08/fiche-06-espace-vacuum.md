# Fiche 06 — Espace disque : compactage accessible

Branche : `feat/vacuum`

## Objectif

Donner à l'utilisateur un moyen de récupérer l'espace libéré par les
suppressions (matchs, tournois, purges) : aujourd'hui aucune commande
VACUUM n'existe (la seule occurrence du mot dans le code est un commentaire
de préparation de démo, `internal/gui/app.go:24`), donc une base de travail
ne rétrécit jamais.

## Tâches

- [ ] Méthode `Vacuum()` sur le wrapper `Database` (et le contrat Storage si
      pertinent — côté Postgres, no-op documenté ou `VACUUM (ANALYZE)` ciblé).
      Attention : `VACUUM` SQLite ne s'exécute pas dans une transaction et
      exige ~2× l'espace du fichier ; vérifier l'espace libre avant, message
      d'erreur clair sinon.
- [ ] Sous-commande CLI `blunderdb vacuum <db>` (fichier `internal/cli/cli_vacuum.go`,
      enregistrement dans `cli.go`, aide `printUsage`), affichant la taille
      avant/après.
- [ ] GUI : entrée « Compacter la base » (menu/ConfigModal), avec taille
      gagnée dans la barre d'état ; passer par la même méthode bound.
- [ ] Enchaîner `ANALYZE` après le VACUUM (synergie fiche 05).
- [ ] Doc française dans la même branche : `doc/source/manuel.rst` (section
      maintenance) + `doc/source/cli.rst` + `CLI_USAGE.md` + aide intégrée
      (`frontend/src/i18n/help/*.js` — les 9 langues pour la nouvelle entrée
      de commande si une commande `:vacuum` est exposée ; sinon doc GUI/CLI
      seulement).
- [ ] Tests : CLI (créer, remplir, supprimer, vacuum, taille réduite) ;
      GUI-level via la méthode `Database`.

## Critères de fin

- Scénario mesurable : base gonflée par des suppressions → vacuum → fichier
  réduit ; testé automatiquement.
- Doc FR livrée dans la branche.

## Risques & garde-fous

- Ne jamais vacuum automatiquement à l'ouverture (coût imprévisible sur de
  grosses bases) : action explicite de l'utilisateur uniquement.
- WAL : checkpoint (`wal_checkpoint(TRUNCATE)`) avant VACUUM pour que la
  taille affichée soit honnête.
