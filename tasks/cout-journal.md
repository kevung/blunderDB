# Journal de coût — tokens et contexte

Une passe de coût par vague ou par lot (ADR-0055 règle 6) :
`scripts/cout-tokens.py --depuis <début>`. Une entrée = le relevé, l'écart aux cibles, la
décision (un gain ou une garde). Rien d'autre.

## Indicateurs et cibles

| Indicateur | Ligne de base (26/08 → 26/09) | Cible |
|---|---|---|
| Contexte moyen des sessions principales | 347k | < 150k |
| Coût des sous-agents au-delà de 150 appels | 27 % | < 10 % |
| `sleep` | 656 | 0 |
| `Agent()` sans modèle explicite | 111 | 0 |
| Contexte fixe au premier appel | 42k | < 35k |
| Relecture de code (estimation) | 69 % | à suivre |
| Commentaires dans le code | 27 % des octets | ≤ 20 % |

## Entrées

### Ligne de base — 26/08 → 26/09

- **Relevé** : 10,5 G tokens (1,26 G pondérés), 99 % de relecture de cache. Sessions
  principales 51 % du coût ; les trois pires (1 400 à 2 100 tours à ~500k de contexte, dont
  une sans aucun sous-agent) ~22 % à elles seules. 52 ouvriers au-delà de 150 appels = 27 %.
- **Décision** : ADR-0055 — skill `traiter-lot` et agent `ouvrier`, hook `budget.py` (sleep,
  lecture sans plage, plafond d'ouvrier, seuil de contexte), `CLAUDE.md` allégé et `CLAUDE.md`
  de dossier pour gammonNet, passe de sobriété sur les commentaires.
- **À vérifier à la prochaine passe** : `sleep` à zéro et contexte moyen sous 150k ; sinon le
  hook ne tient pas et il faut le durcir.
