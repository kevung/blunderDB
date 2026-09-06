# Transcription de matchs

Saisir dans blunderDB un match joué ailleurs — au club, sur un plateau, depuis une vidéo ou
une feuille de score — le corriger, l'enregistrer comme un Match ordinaire analysé par
gammonNet, l'exporter en `.mat`.

Cadrage : grill du 2026-09-06/07, dix-huit décisions ; décisions durables dans
[ADR-0043](../../docs/adr/0043-transcribing-a-match-is-not-playing-one.md) et
[ADR-0044](../../docs/adr/0044-a-transcription-is-a-draft-that-owns-its-match.md) ;
vocabulaire dans `CONTEXT.md`, section « Recording a match ».

| Document | Contenu |
|---|---|
| [fonctionnel.md](fonctionnel.md) | le document, les Actions, le Replay, les Incohérences, les gestes, l'enregistrement, le `.mat`, vingt-deux flux |
| [ux.md](ux.md) | placement, machine à états du clavier, budgets KLM par flux, retour visuel, vérification |
| [integration.md](integration.md) | impact sur la base, le moteur, Wails, le frontend, la documentation ; ce qui n'est pas touché |
| [plan.md](plan.md) | quatre lots, vingt-sept issues, ordre et parallélisme, risques |
| [issues/](issues/README.md) | une issue verticale par fichier, prête à coller dans GitHub |

## Questions résolues sous hypothèse, à confirmer

Les six points laissés ouverts par le grill ont reçu la recommandation ci-dessous ; les
documents la suivent. Un désaccord se corrige dans le document, pas dans le code.

1. **Le mot** : *Action* (glossaire), « action » à l'écran. « Entry » est dans la liste des
   mots à éviter pour une Position, « Fault » est pris par l'Entraînement, « élément » ne dit
   rien.
2. **Raccourci et commande** : `Ctrl+Maj+T` (seuls `Ctrl+H/J/A/Z` sont libres, sans
   mnémonique ; `Ctrl+Maj+I/F/S` existent déjà) ; `transcribe`, alias `tr`.
3. **Transcripteur** : `metadata.user` de la base par défaut, modifiable dans le volet.
4. **Touches** : `r` puis `1`/`2`/`3` pour la résignation ; `s` pour changer le camp ;
   `Ctrl+Z`/`Ctrl+Maj+Z` pour annuler/rétablir.
5. **Fermer un brouillon** : bouton « Fermer » ; confirmation si jamais enregistré ; un Match
   enregistré reste et devient définitif.
6. **Plusieurs brouillons** : stockés sans limite, listés dans le panneau ; **un seul ouvert**
   dans l'interface à la fois.
