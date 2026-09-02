<!-- Fiche du plan tasks/gammonnet-perf/README.md. Le contexte, les décisions
et le tableau d'ensemble vivent dans le README ; cette fiche ne porte que son
propre travail et ses propres chiffres. -->

# F5 — Consigner [S] — #149

- [ ] ADR-0024 : passer de *proposed* à *accepted*, y inscrire les chiffres finaux de F0.
- [ ] `CLAUDE.md`, section *Invariants* : « Le noyau réseau n'utilise jamais FMA ni
      réassociation ; tout nouveau chemin de calcul passe `kernel_identity_test.go` contre le
      repli pur Go ; `WithWorkers` et le lot inter-positions sont des exigences de
      production, pas des outils de test. »
- [ ] `docs/adr/README.md` : ligne 0024, relation « exécute le suivi borné de 0011 ».
- [ ] `tasks/BACKLOG.md` : les frais hors réseau, avec le nouveau profil (post-F2) comme
      chiffre d'entrée.
- [ ] Manuel utilisateur : rien (aucune durée n'y est citée). `CLI_USAGE.md` : `--jobs`.

---
