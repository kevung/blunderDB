# La Leçon HTML est un document du moteur dont le bureau fournit les plateaux

Status: accepted.
See also: ADR-0066 (les Leçons), ADR-0007 (rien n'est écrit chez le destinataire), `pkg/blunderdb/report`

## Context

Un auteur veut publier une Leçon sans que le lecteur installe quoi que ce soit : un fichier
HTML unique. Le plateau y est dessiné par `renderPositionSVG`, celui du rapport HTML, dans la
palette choisie à l'écran. Mais la CLI et le démon n'ont pas d'écran ni de two.js : ils doivent
produire la même page.

## Decision

1. **Le document est construit par le moteur** (`report.BuildLesson`), par-dessus
   `storage.Storage` : le bureau, `lesson export --html` et `/v1/lessons.html` n'ont qu'un
   générateur. Titre, description, Étapes, analyse stockée repliée dans un `<details>`
   (réponse cachée, comme un quiz), origine du Filigrane en pied de page.
2. **Le bureau fournit les plateaux** : `LessonHTML(id, langue, diagrammes)` reçoit un SVG par
   position, dessiné par `renderPositionSVG` — jamais un second dessinateur côté écran. Sans
   écran, le moteur retombe sur `report.Diagram`, le repli que le rapport HTML utilise déjà.
3. **Aucune dépendance externe dans la page** : CSS inline, SVG inline, ni script ni adresse
   réseau. Le texte importé est échappé fragment par fragment.
4. **Une Étape qui montre une Collection imprime ses `MaxCollectionBoards` premières positions**
   et dit combien manquent : un document que l'on lit, pas un vidage.
5. **Les mots du document** (Étapes, Voir l'analyse…) vivent dans `report/lesson.go`, pas dans
   les catalogues du frontend : la page se rend sans écran, et n'a pas à dériver d'un JSON.
6. **Lire ou exporter n'écrit rien** (ADR-0007) : aucune progression, aucune trace.
