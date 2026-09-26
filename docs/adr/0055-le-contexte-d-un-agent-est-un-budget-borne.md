# Le contexte d'un agent est un budget borné

Statut : acceptée.

## Contexte

Chaque tour renvoie au modèle la conversation entière : 99 % des tokens consommés sur ce dépôt
sont de la relecture de cache, et le coût d'une session croît comme sa longueur multipliée par
la taille de son contexte. Les pires postes mesurés : des sessions d'orchestration de 1 400 à
2 100 tours à ~500k de contexte moyen, qui faisaient tout elles-mêmes (22 % du total à trois),
et des ouvriers au-delà de 150 appels (27 %). La lecture de code fait ~65 % de la relecture.

## Décision

1. **Un orchestrateur ne produit pas de volume.** Un lot d'issues se traite par le skill
   `/traiter-lot` : une issue (ou une zone de code) = un sous-agent `ouvrier` frais ;
   l'orchestrateur lit, décide, délègue, enregistre un verdict. Il n'édite pas, ne lance pas de
   suite, ne lit pas de code.
2. **Un ouvrier a un plafond** : 150 appels d'outil ou le 3e rouge sur le même test. Il rend
   alors `à reprendre` avec un état de reprise en trois lignes, et un ouvrier neuf reprend.
3. **Une session principale a un seuil** : au-delà de 200k tokens de contexte, elle consigne son
   état et propose un `/clear`.
4. **On lit une plage, pas un fichier** : `grep -n` pour situer, puis `sed -n a,b` ou `Read`
   avec `offset`/`limit`. Une suite rend ses rouges, jamais son log. Pas de `sleep`.
5. **Le palier de modèle suit la réfutabilité** : chaque appel `Agent` nomme son modèle. Sonnet
   quand une erreur produit un rouge (implémenter une fiche validée, exécuter, chercher) ; Opus
   quand elle produit un silence (cadrage, revue, diagnostic, migration, arithmétique du moteur).
6. **Une passe de coût par vague** : `scripts/cout-tokens.py --depuis <début>`, consignée dans
   `tasks/cout-journal.md` avec ses cibles. Elle produit un gain ou une garde.
7. **Ce qui est chargé à chaque tour reste court** : `CLAUDE.md` garde les règles et les
   invariants en une ligne avec leur ADR ; le détail d'un sous-système va dans un `CLAUDE.md`
   de son dossier, chargé seulement quand on y travaille.

## Conséquences

- Un aller-retour de plus par issue, et l'orchestrateur juge sur un verdict sans voir le code :
  la qualité repose sur le contrat de retour de l'ouvrier (`.claude/agents/ouvrier.md`).
- Écartée — traiter en ligne les issues triviales : « trivial » se juge sur une issue qu'on n'a
  pas lue, et chaque exception légitime la suivante.
- Écartée — un essaim d'ouvriers qui écrivent en même temps dans un même worktree : ils
  divergent. Des ouvriers en parallèle travaillent sur des zones de fichiers disjointes.
- Une règle seulement écrite ne tient pas (gammonGo a gardé des centaines de `sleep` après leur
  interdiction) : les règles 2, 3 et 4 sont tenues par un hook.

## Garde

`.claude/hooks/budget.py` (branché dans `.claude/settings.json`) : refuse `sleep`, le `cat` ou le
`Read` sans plage d'un fichier de plus de 400 lignes ; ordonne `à reprendre` au 150e appel d'un
sous-agent ; signale le seuil de contexte de la session principale. `scripts/cout-tokens.py`
mesure le reste.
