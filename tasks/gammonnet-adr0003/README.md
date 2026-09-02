# Verticale ADR-0003 — ce que blunderDB reprend du chantier gammonNet 2026-09-02/03

Issue [#212](https://github.com/kevung/blunderDB/issues/212). Quatre postes descendus
d'amont en application de son **ADR-0003** : *une optimisation est conceptuelle si son
gain survit à un changement de langage ; les conceptuelles se décident en amont et le
portage suit* — et **aucune ne se reprend sans être remesurée ici**, ce qui est l'autre
moitié de la règle.

| poste | verdict | mesure |
|---|---|---|
| 1. Valuer le videau par lot sur les candidats | **réfuté ici**, et remplacé par une levée qui rend plus | [poste-1-videau-par-lot.md](poste-1-videau-par-lot.md) |
| 2. Garde-fou sur l'arrondi des tuiles | livré | [poste-2-garde-fou-tuile.md](poste-2-garde-fou-tuile.md) |
| 3. Resserrer la zone d'égalité du gold | voir la fiche | [poste-3-zone-egalite-gold.md](poste-3-zone-egalite-gold.md) |
| 4. Deux mesures à refaire ici | voir la fiche | [poste-4-mesures-separees.md](poste-4-mesures-separees.md) |

## La machine, et l'instrument

Toutes les mesures de ces fiches : Ryzen 7 PRO 6850U (8 cœurs / 16 fils, AVX2), Go
1.25.13, Linux 7.1.9-arch1-2, noyau `avx2`. **La machine n'était pas oisive** — d'autres
chantiers y tournaient, charge relevée de 4 à 108 selon le moment. Aucun chiffre absolu
de ces fiches ne doit être cité hors de son rapport.

Deux règles d'instrument, l'une importée d'amont et l'autre trouvée ici :

- **Chronométrer en entrelacé, dans le même processus.** Le chantier T85 d'amont a lu le
  même poste à 10,6 %, 26 % ou 20,5 % le même après-midi selon qu'il soustrayait deux
  exécutions consécutives, alternait trois passes, ou entrelaçait décision par décision.
  Un facteur 2,5, sur un seuil d'abandon de 5 %. Les instruments d'ici
  (`cube_measure_test.go`) alternent donc les configurations à quelques microsecondes
  ou millisecondes l'une de l'autre, et rendent des médianes de rapports par paire.
- **Épingler.** Sous charge 20 et plus, l'entrelacement seul ne suffit plus : les
  relevés dérivaient d'un facteur 2 entre deux exécutions. `GOMAXPROCS=2 taskset -c 12,13`
  sur le binaire de test compilé rend des tableaux reproductibles à ~2 % près sur trois
  exécutions consécutives. **C'est la recette à reprendre pour rejouer ces chiffres.**

```bash
go test -c -o /tmp/gn.test ./pkg/blunderdb/engine/gammonnet/
BLUNDERDB_MEASURE=1 GOMAXPROCS=2 taskset -c 12,13 /tmp/gn.test -test.run TestMeasureCubePost -test.v
```
