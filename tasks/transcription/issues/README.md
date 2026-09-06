# Issues verticales — transcription de matchs

Une issue par fichier, prête à coller dans GitHub (`gh` est cassé sur cette machine). Le
plan et l'ordre sont dans [../plan.md](../plan.md).

## Format

Chaque fichier suit la convention des fiches du dépôt
(`tasks/plan-amelioration-2026-09b/`), enrichie de ce qu'une issue verticale exige :

```
# T<lot>.<n> — <Titre> [S|M|L]

<Une à trois phrases : le problème, pas la solution.>

**Tranche verticale** : <ce que l'issue traverse : Go → binding → Svelte → tests → doc>
**Prérequis** : <issues>
**Livrable** : <ce qui marche à la fin, du point de vue de l'utilisateur ou de l'appelant>

## Travail
- <points de code, fichier par fichier>

## Recette
- <critères vérifiables, dont un budget KLM ou une latence quand la fiche en a un>

## Invariants touchés
- <lignes de CLAUDE.md concernées>
```

Une issue est **verticale** : elle laisse l'application utilisable et testable. Une issue
qui n'ajoute que du code Go sans appelant est acceptable au lot 0 seulement, où le paquet
est lui-même le livrable et ses tests l'appelant.

## Index

Lot 0 — T0.1 à T0.7 · Lot 1 — T1.1 à T1.10 · Lot 2 — T2.1 à T2.5 · Lot 3 — T3.1 à T3.5.
