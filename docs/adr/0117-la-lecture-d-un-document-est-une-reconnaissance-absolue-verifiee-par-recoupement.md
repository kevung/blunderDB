# La lecture d'un document est une reconnaissance absolue vérifiée par recoupement ; seul le relevé vérifié écrit en base

Statut : acceptée (principe accepté par l'utilisateur). Écart assumé à l'ADR-0108 règles 2 et 3.
Voir aussi : ADR-0001 (provenance « importée seule »), ADR-0007 (rien chez le destinataire),
ADR-0108 règle 5 (trois états), `tasks/import-document/ETUDE.md`.

## Contexte

Un livre, un article ou une page web montre des positions en diagrammes, dans un ordre qui est
celui de l'étude. L'utilisateur veut les retrouver dans une Collection, dans cet ordre. Un
diagramme imprimé n'est pas un plateau filmé : aucun moteur ne fournit de candidats entre
lesquels choisir, la position est inconnue avant d'être lue. Mais un livre imprime ses propres
contrôles : le compte de pips, quinze pions par camp, la légende (trait, dés), le coup de la
solution. Le texte de l'auteur et ses images sont son œuvre ; une Collection s'exporte et se
partage.

## Décision

1. **La lecture d'un document est une reconnaissance absolue**, contrairement à l'ADR-0108
   règle 3 : la position est lue sans candidats. Elle ne devient sûre que si un recoupement
   indépendant du lecteur la confirme (aujourd'hui : quinze pions par camp et pips imprimés ;
   la légende et le coup de la solution ne sont pas lus) ; jamais par l'équité ni l'avis du moteur (ADR-0108 règle 4). Dans le doute,
   « à vérifier ».
2. **La lecture peut se faire hors de la webview**, contrairement à l'ADR-0108 règle 2 : un
   diagramme d'imprimerie se lit en Go pur (`document/pdf` pour le fichier, la table de glyphes de `document` pour le
   diagramme), sans CGO ni modèle
   appris. Le reste de l'ADR-0108 vaut : rien n'est appris chez l'utilisateur.
3. **La lecture produit un Relevé et n'écrit rien en base.** Le Relevé (`pkg/blunderdb/document`,
   CONTEXT.md « Relevé ») est un fichier JSON versionné, relu strictement : champs inconnus
   refusés, rangs continus depuis 1, position légale. La position y est un XGID, l'encodage que
   le dépôt lit et écrit déjà ; dés, videau, trait, score et longueur du match y sont, et nulle
   part ailleurs. La lecture d'un PDF n'écrit jamais de dés (`00`).
4. **Seul l'import d'un Relevé écrit, et seulement des positions.** Chaque position est écrite
   comme importée seule (ADR-0001), dans une Collection au nom du document, dans l'ordre du
   Relevé. Aucune image, aucun rectangle, aucune prose de l'auteur n'entre en base : la
   description de la Collection porte le titre, l'auteur, l'édition ou l'URL ; chaque position
   porte sa référence (« p. 146, Problem 191 ») en commentaire, lisible hors de la Collection.
5. **Une entrée ni sûre ni vérifiée refuse l'import**, sauf forçage explicite ; un import refusé
   ou interrompu n'écrit rien (une transaction). Une position que le document montre deux fois
   appartient une fois à la Collection, à sa première apparition, et son commentaire cite chaque
   référence. Réimporter le même Relevé n'ajoute rien.
6. **Parité** : `Database.ImportDocumentReleve`, `blunderdb document import` et
   `/v1/documents.importReleve` passent par `document.Import`. Le démon importe un Relevé ; il
   ne lit aucun document.

## Conséquences

- Aucun changement de schéma : la Collection a déjà une description, la position un fil de
  commentaires. Le commentaire de référence est retrouvé par son préfixe (titre du document),
  réécrit quand ses références changent, jamais doublé.
- Le commentaire de référence a l'origine `user` : aucune origine n'existe pour un document, et
  la position importée seule est retenue par sa provenance, pas par son commentaire.
- Le Relevé garde page et rectangle ; les vignettes de vérification se redessinent depuis le
  fichier d'origine, qui reste chez l'utilisateur.
- Rejeté : discriminer entre candidats (il n'y en a pas) ; stocker les vignettes ou le texte
  du livre (une Collection exportée les emporterait) ; un champ dédié à la référence (un
  changement de schéma pour ce qu'un commentaire porte déjà) ; importer une lecture non
  vérifiée sans le dire.
