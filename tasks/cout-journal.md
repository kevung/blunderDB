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

### Passe de sobriété (ADR, commentaires)

- **Relevé** : ADR 422 → 161 Ko ; commentaires du code 2 903 → 2 164 Ko (29 → 24 %), aucune
  ligne de code modifiée. Vingt ouvriers ; le plafond de 150 appels a arrêté proprement quatre
  d'entre eux, repris sans perte par un ouvrier neuf. Le seuil de contexte a signalé la
  session principale à 283k.
- **Écart** : les ouvriers Sonnet n'ont fait que retirer l'historique (−0,7 à −4 % sur leur
  zone) et ont atteint le plafond trois fois sur quatre ; une seconde passe Opus a été
  nécessaire partout (−34 à −40 %). Condenser un commentaire est un jugement dont l'erreur est
  silencieuse : c'était Opus d'emblée (ADR-0055 règle 5), la règle était juste, son
  application non.
- **Décision** : aucune règle nouvelle ; un travail éditorial (commentaires, ADR, doc) part sur
  Opus.

### Lot final 0.37.0 — 04/10 → 05/10

- **Relevé** : 1,01 G tokens (147 M pondérés), 96 % de relecture de cache. Huit sessions
  principales = 10 % du coût, contexte moyen 132k (pire 147k) ; 212 sous-agents = 90 %.
  Cinq ouvriers au-delà de 150 appels = 12 % ; `Agent()` sans modèle 0 ; contexte fixe 31k ;
  commentaires 22 % des octets du code.
- **Écart** : le contexte moyen, le modèle explicite et le contexte fixe tiennent leur cible.
  `sleep` 57 au lieu de 0, le plafond d'appels 12 % au lieu de 10 %, les commentaires 22 % au
  lieu de 20 % : le hook ne tient pas tout.
- **Décision** : aucune règle nouvelle ; à la prochaine passe, chercher d'où viennent les
  `sleep` (ouvriers de mesure longue, attentes de CI) avant de durcir `budget.py`.
