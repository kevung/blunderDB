# Un commentaire signé porte la parole de son émetteur, sans changer sa rétention

Status: accepted.
See also: ADR-0007 (rien n'est écrit chez le destinataire), `pkg/blunderdb/issuance/comment.go`

## Context

Une coach suit des élèves par échange de fichiers : l'élève exporte, la coach commente et
renvoie, l'élève importe. Le champ *auteur* d'un Commentaire est du texte libre : rien ne
distingue la note de la coach d'une note que l'élève aurait signée du même nom. Le Filigrane
dit déjà, signé par l'Identité d'émetteur, d'où vient un fichier ; il ne dit rien de chaque
note.

## Decision

1. **La signature est écrite par le producteur, à l'export filigrané.** Le même geste qui
   scelle le Filigrane scelle, avec la même identité, chaque Commentaire d'origine `user`
   qui n'en porte pas encore (`ingest.ExportOptions.Signer`). Un export sans filigrane ne
   signe rien et ne crée pas d'identité. Le démon signe avec la sienne, comme son filigrane.
2. **Ce qui est signé** : le texte exact, l'auteur nommé, le hachage Zobrist de la Position
   commentée, le nom de l'émetteur et l'instant (`issuance.CommentStatement`), dans la même
   enveloppe Ed25519 que le Filigrane, rangée telle quelle dans `comment.signature` (schéma
   2.47.0, colonne nullable). Une note signée déplacée sur un autre plateau ne se vérifie
   plus : la signature atteste des mots et de l'endroit où ils sont posés.
3. **Une signature voyage telle quelle.** Un Commentaire qui en porte déjà une — reçu d'une
   coach, puis réexporté par l'élève — garde celle de son émetteur : la resigner mettrait le
   nom de l'exportateur sur la note d'un autre. `signature` rejoint
   `issuance.CarriedCommentColumns`.
4. **L'import conserve, le lecteur vérifie.** L'import (bureau et `ingest.DBImporter`)
   recopie la signature et n'écrit rien d'autre : ni registre, ni journal, ni lignée
   (ADR-0007). La source est lue telle quelle, jamais migrée : un fichier antérieur à
   2.47.0 n'a pas la colonne, elle est sondée (`sqlite.Storage.FileComments`, et côté bureau
   `readSourceCommentColumns`) et ses Commentaires arrivent non signés. La vérification
   n'est pas stockée : chaque lecture d'un Commentaire (`sqlshared.scanCommentEntry`, qui
   relit le hachage de sa Position) la refait et rend `CommentSigner{Name, Fingerprint,
   Verified}`. Une note altérée dans le fichier est importée et affichée *non vérifiée*,
   jamais rejetée ni présentée comme non signée.
5. **Modifier un Commentaire efface sa signature** (`Update`, `Upsert`), comme cela fait
   déjà de lui un Commentaire de l'utilisateur : les mots signés ne sont plus là. La
   corbeille, la migration vers PostgreSQL et `Rehome` la conservent.
6. **La rétention ne lit pas la signature, elle lit l'origine, qui voyage désormais.**
   `origin` rejoint `issuance.CarriedCommentColumns` : une note d'origine `user` chez le
   producteur arrive `user` et retient sa Position — c'est la note de la coach que l'élève
   veut garder quand il supprime le match ; une remarque qu'un fichier XG avait apportée au
   producteur arrive `xg` et part avec le match, comme chez lui. Une source sans la colonne
   (antérieure à 2.19.0) donne `unknown`. Un fichier exporté avant 2.47.0 porte `unknown`
   partout, puisque l'origine n'y voyageait pas : ses Commentaires ne retiennent rien,
   comme le faisait déjà l'import du bureau, alors que `ingest.DBImporter` les écrivait
   `user`. Les trois modes recopient désormais l'origine de la même façon.
   `positionIsHeldSQL` et la purge des orphelins restent inchangés dans leurs trois copies.

## Consequences

- La vérification prouve qu'une clé a signé ces mots, pas qu'elle appartient à la personne
  nommée : comme pour le Filigrane, c'est l'empreinte publiée par l'émetteur qui le dit.
- L'export JSON et le backend `remote` ne transportent pas encore la signature : un
  Commentaire écrit à travers eux arrive non signé, ce qui ne trompe personne.
