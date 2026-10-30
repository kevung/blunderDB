# 08 — UX des outils d'annotation « humain dans la boucle »

Contexte. Je conçois une fonction pour blunderDB, une application de bureau de backgammon
(Go + Svelte dans une webview via Wails v2, binaire unique pour Linux/WebKitGTK, Windows/WebView2,
macOS/WKWebView). Le panneau de transcription lit une vidéo de match (fichier local) ; l'utilisateur
saisit les dés au clavier, l'application liste les coups légaux classés par un réseau de neurones de
backgammon, l'utilisateur choisit le coup joué. Je veux qu'une analyse des images de la vidéo
détecte les lancers, lise les dés, reconnaisse le coup joué et les gestes de videau, pour que
l'utilisateur ne fasse plus que confirmer ou corriger. Contraintes : aucun décodage vidéo hors de la
webview, pas de dépendance native (pas de CGO, pas de ffmpeg, pas d'OpenCV natif), une caméra de
stream généralement fixe mais d'angle, de lumière et de matériel variables.

Exigences sur la réponse. Travaille comme pour une revue de littérature : privilégie les sources
primaires (articles avec DOI ou arXiv, dépôts avec code, documentation des éditeurs de navigateurs,
bugzilla/chromium-status, caniuse) et date chaque fait. Couvre 2015–2026, en insistant sur
2023–2026. Quand deux sources se contredisent, dis-le. Quand tu ne trouves pas, dis « non trouvé »
plutôt que d'extrapoler. Termine par : 1. **Verdict** ; 2. **Recommandation technique** ;
3. **Incertitudes** (avec l'expérience à mener) ; 4. **Sources** numérotées (titre, auteurs ou
éditeur, année, lien, une ligne de contenu, fiabilité primaire / secondaire / blog). Réponds en
français.

---

## Question

Le panneau actuel repose sur des principes écrits : aucune frappe n'est avalée, un indicateur
permanent dit l'effet de la prochaine touche, le moteur propose et ne choisit jamais à la place de
l'utilisateur, tout geste a un chemin clavier. L'assistant visuel doit s'y insérer en deux niveaux :
**assisté** (il pré-remplit, l'utilisateur confirme d'une touche) et **automatique** (il valide
seul après un délai visible si la confiance suffit, sinon il met la vidéo en pause et attend).

1. **Patterns documentés** d'annotation vidéo semi-automatique : annotation sportive (tagging
   d'événements en direct et en différé), retransmission d'échecs (DGT LiveChess, arbitrage avec
   correction), sous-titrage assisté par reconnaissance vocale, outils de labellisation (CVAT, Label
   Studio, Supervisely) avec pré-annotation par modèle. Pour chacun : comment la proposition est
   montrée, comment on la corrige, comment on la refuse, comment on revient en arrière.
2. **Le coût de la vérification** : études (IHM, facteurs humains, « automation complacency »,
   « trust in automation ») sur le seuil d'erreur au-delà duquel l'utilisateur revérifie tout et
   l'assistance devient plus lente que le manuel ; chiffres publiés ; effet du *type* d'erreur
   (fausse détection vs détection manquée) sur la confiance.
3. **Affichage de la confiance** : représentations qui marchent (couleur, délai proportionnel,
   explication locale « j'ai lu 6-3 car… », surimpression de la lecture sur l'image) et celles qui
   ne marchent pas (pourcentages bruts).
4. **Validation par délai** (« accepter si silence pendant N secondes ») : précédents, valeurs de N,
   risques (clic réflexe, inattention), alternatives (validation par le lancer suivant : « si tu as
   lancé, c'est que le coup est joué »).
5. **Pause sur incertitude** : comment reprendre sans perdre le fil (image gelée, zoom sur la zone
   douteuse, proposition de deux lectures à départager d'une touche).
6. **Saut d'événement en événement** : UX des lignes de temps à marqueurs (outils de montage,
   annotation sportive) ; raccourcis usuels ; comment montrer les événements incertains.
7. **Mesures de réussite** : métriques d'ergonomie à relever (gestes par tour, temps par tour,
   corrections par match, abandons du mode automatique) et protocoles légers pour les mesurer sur
   soi-même et quelques utilisateurs.

Attendu : les cinq mécanismes à retenir, avec pour chacun la source qui le justifie, et le
protocole de mesure qui dira si l'assistant fait gagner du temps.
