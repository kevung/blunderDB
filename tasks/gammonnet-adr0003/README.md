# Verticale ADR-0003 — ce que blunderDB reprend du chantier gammonNet 2026-09-02/03

Issue [#212](https://github.com/kevung/blunderDB/issues/212). Quatre postes descendus
d'amont en application de son **ADR-0003** : *une optimisation est conceptuelle si son
gain survit à un changement de langage ; les conceptuelles se décident en amont et le
portage suit* — et **aucune ne se reprend sans être remesurée ici**, ce qui est l'autre
moitié de la règle.

| poste | verdict | chiffre | fiche |
|---|---|---|---|
| 1. Valuer le videau par lot sur les candidats | **réfuté ici** — le lot rend la décision **12,3 % plus lente** ; livré à sa place, la **levée** qu'il a fait trouver | **−16,5 %** sur une décision 2-ply au score | [poste 1](poste-1-videau-par-lot.md) |
| 2. Garde-fou sur l'arrondi des tuiles | **livré** | rien à mesurer : une assertion là où il n'y en avait aucune | [poste 2](poste-2-garde-fou-tuile.md) |
| 3. Resserrer la zone d'égalité du gold | **livré** — la publication amont (v1.3.0) était disponible, le gold a été rejoué contre elle d'abord | 0 ex æquo employé sur 85 + 123 décisions, et l'employer devient un échec | [poste 3](poste-3-zone-egalite-gold.md) |
| 4. Deux mesures à refaire ici | **mesuré, rien à livrer** | sparsité : **4,1 %** au grand réseau, **≈0** au petit ; tuile à huit : ne transfère pas à largeur de lot 8 | [poste 4](poste-4-mesures-separees.md) |

## Ce que ces quatre postes renvoient en amont

L'ADR-0003 dit que **les mesures remontent toujours, y compris celles qui
réfutent**. Trois choses à consigner là-bas :

1. **Le videau par lot ne survit pas au changement de langage.** La
   spécification §7.1 annonce que la forme « se transportera telle quelle en
   Go » ; mesuré ici, elle rend ×0,89 sur la décision entière là où le C rend
   ×1,13. La raison est nommée : gcc inline `level_live`, le compilateur Go la
   juge trop complexe, et une fois les constantes de segment sorties des
   soixante pas la bissection cesse d'être bornée par la latence — il ne reste
   plus rien à recouvrir. Cette **levée** rend 16,5 % ici et n'a jamais été
   mesurée en C, où elle est probablement nulle.
2. **La preuve de bit-identité de T85 vaut pour ses drapeaux.** Sous
   `-O3 -march=native`, le gold match+videau bouge 41 équités de candidat d'au
   plus 2,2e-16 entre le commit du lot et son parent : `-ffp-contract=fast`
   fond `y0 + (y1−y0)·t` en FMA dans une forme et pas dans l'autre. Aucun coup
   choisi ne change. Le harnais de comparaison amont gagnerait à porter
   `-ffp-contract=off`, ou à dire sous quels drapeaux la bit-identité est
   affirmée.
3. **La forme sans branche est un gain sous lot et une perte sous scalaire.**
   Le C n'applique ses `select` qu'au pas cadencé, ce qui est le bon choix ;
   ici, l'appliquer aussi à la bissection sérielle coûtait 10 à 20 %.

## Ce qui ne redescend pas, et il faut le dire

Repris de l'issue et vérifié en passant :

- **Le drapeau de réassociation flottante** : sans objet deux fois — Go n'en a
  pas, et l'ADR-0024 l'interdit de toute façon.
- **La largeur de lot 16 retenue pour WebAssembly** : elle vient de ce que
  SIMD128 n'a que quatre voies, ce qui fait dégénérer la tuile à largeur 32.
  Sans rapport avec AVX2 et ses huit voies.
- **La règle d'ordonnancement d'amont** confirme ce que ce dépôt a fait plutôt
  qu'elle ne le corrige : « le nombre de tâches paie quand une tâche coûte cher
  à calculer et rien à transmettre », et ici les tâches sont des goroutines qui
  partagent la mémoire — la transmission ne coûte rien, et aplatir la frontière
  a bien payé (#148).
- **Une confirmation mutuelle** : les deux dépôts ont mesuré le même mur
  mémoire, indépendamment, dans deux langages et deux exécutions — un facteur
  quatre sur huit cœurs de part et d'autre. Ce n'est plus une particularité de
  l'un.

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
