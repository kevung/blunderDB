---
name: traiter-lot
description: Traiter un lot d'issues blunderDB en orchestrateur — une issue (ou une zone de code) = un ouvrier frais, l'orchestrateur ne lit ni n'édite de code et ne garde que des verdicts. Invoqué par l'humain uniquement.
disable-model-invocation: true
---

# Traiter un lot d'issues

Cadre : ADR-0055. Vaut dès deux issues ; pour une seule, travaillez normalement.

**Vous ne produisez pas de volume.** Vous lisez l'issue, décidez, déléguez, enregistrez un
verdict, fusionnez. Pas d'`Edit`, pas de `Write` dans le dépôt, pas de suite de tests, pas de
lecture de code. Un `Bash` de plus de quelques lignes de sortie est une faute du dispositif.

## Boucle, une issue à la fois

1. **Cadrer** : `gh issue view <n> --json title,body,labels` — rien d'autre. Une issue caduque
   se ferme avec son motif.
2. **Terrain** : `git worktree list`. Une zone tenue par une autre session s'évite.
3. **Grouper par zone de code**, pas par numéro : des issues qui touchent les mêmes fichiers vont
   au même ouvrier, qui ne relit le code qu'une fois — au plus trois issues par ouvrier.
4. **Choisir le modèle** (ADR-0055 règle 5) : `sonnet` par défaut ; `opus` si l'issue touche une
   migration ou `DatabaseVersion`, le prédicat de rétention, l'arithmétique de
   `engine/gammonnet`, le filigrane (`issuance`), ou si l'ouvrier doit écrire lui-même l'oracle
   d'une règle silencieuse.
5. **Déléguer** : `Agent(subagent_type: "ouvrier", model: …, prompt: …)`. Le prompt tient en
   cinq lignes : numéros d'issue, chemin absolu du worktree et nom de branche, ce qui doit être
   vrai à la fin si l'issue est floue. Le contrat de l'ouvrier est dans sa définition : ne le
   recopiez pas.
6. **Plafond** : un `à reprendre` part à un ouvrier **neuf**, avec l'état de reprise — jamais un
   `SendMessage` qui prolongerait le contexte enlisé. Une reprise, pas deux : au second plafond,
   l'issue est rouge.
7. **Revue** — sauf documentation ou renommage pur : `Agent(model: "opus")` en lecture seule,
   à qui l'on donne l'issue, la branche et le SHA, avec un seul axe : *l'issue est-elle
   satisfaite, et rien de plus ?* Pour une migration, un invariant de `CLAUDE.md` ou le moteur,
   un second relecteur, axe *conventions et invariants du dépôt*. Un rouge de revue vaut **un**
   renvoi au même ouvrier (`SendMessage`, son contexte est intact) ; toujours rouge, l'issue
   est rouge et la branche reste.
8. **Fusionner** un vert : `git merge <branche>` dans le checkout principal, `git worktree
   remove`, `git branch -d`. Le commit porte `Closes #n` ; sinon `gh issue close <n>`.

Un seul ouvrier écrit à la fois dans une zone donnée ; deux ouvriers en parallèle travaillent
sur des fichiers disjoints, chacun dans son worktree.

## Fin de lot — et plafond de l'orchestrateur

Au-delà de **dix issues**, ou quand le hook signale le seuil de contexte, arrêtez le lot.

Rendez un tableau, une ligne par issue : numéro, verdict, branche, modèle, étages de revue
(écrire `0` quand il n'y en a pas), écarts. Puis la prochaine issue non traitée : c'est l'état
de reprise, que l'utilisateur relance après `/clear`.

Proposez enfin la passe de coût : `scripts/cout-tokens.py --depuis <début du lot>`, à consigner
dans `tasks/cout-journal.md`.
