Recherche du 2026-10-30 — claude.ai, recherche approfondie.
Prompt : ../prompts/07-youtube-acces-legitime.md

---

# YouTube et analyse automatique des images : existe-t-il un accès légitime pour blunderDB ? (état au 30 octobre 2026)

**Non : aucune voie documentée ne permet à blunderDB d'analyser automatiquement les images d'une vidéo YouTube tout en restant dans le cadre des conditions d'utilisation de YouTube.** Les CGU grand public (version Google LLC du 15 décembre 2023), les Developer Policies des API et la conception même du lecteur intégré excluent tous les chemins que j'ai examinés, pour un développeur qui ne dispose pas d'une autorisation écrite de YouTube : téléchargement, storyboards, capture de la fenêtre, extension de navigateur. Ce qui reste légitime, c'est l'analyse d'un fichier vidéo que l'utilisateur détient licitement par une autre voie que YouTube. L'application peut aussi continuer à se synchroniser sur le lecteur YouTube par l'horodatage seul, comme aujourd'hui.

## TL;DR

- **Verdict : non.** Les CGU YouTube interdisent d'« accéder, reproduire, télécharger […] ou utiliser autrement » un contenu hors de ce que le service autorise expressément (restriction n° 1). Elles interdisent aussi l'accès « par des moyens automatisés » (n° 3). Les Developer Policies, auxquelles blunderDB est soumis comme client de l'IFrame Player API, interdisent en plus de mettre en cache ou de stocker le contenu audiovisuel (III.E.1), d'utiliser des API non documentées (III.D.7, ce qui vise les storyboards) et de « créer des données dérivées » des API Data (III.E.4.h). Cette dernière règle couvre précisément ce que ferait une lecture automatique des dés.
- **Techniquement**, seule la capture d'écran de la propre fenêtre de l'application (`getDisplayMedia`) donnerait des pixels exploitables, et de façon fiable uniquement sous WebView2. Sous WebKitGTK, il faut activer `enable-media-stream`, gérer le signal `permission-request` et disposer de PipeWire et du portail. Sous WKWebView, il faut une méthode de délégué privée plus l'autorisation « Enregistrement de l'écran ». Cette voie ne contourne pas de DRM pour les vidéos non protégées, mais elle reste un usage automatisé que la licence ne couvre pas. Elle n'est donc pas « légitime » au sens des CGU.
- **Recommandation :** l'analyse d'images est réservée aux fichiers locaux. Pour YouTube, garder le mode assisté actuel (horodatage, état, vitesse), sans aucune capture. Ne jamais nommer ni lier yt-dlp ou un outil équivalent, compte tenu de la jurisprudence de Hambourg 2023-2024 sur le simple hébergement du site de youtube-dl. Diriger l'utilisateur vers le fichier source du streamer ou de l'organisateur, ou vers ses propres enregistrements.

## Key Findings

1. **Les CGU grand public sont la barrière principale, et elle est contractuelle.** La version US (« Dated: December 15, 2023 », prestataire Google LLC) n'autorise que la visualisation « for your personal, non-commercial use » et l'affichage via le lecteur intégrable. La licence que les créateurs accordent aux autres utilisateurs vaut « only as enabled by a feature of the Service » et « does not grant any rights […] independent of the Service ».\[1\]
2. **Les versions se contredisent sur un point important.** La traduction de courtoisie EEE (Google Ireland) que j'ai consultée est datée du 5 janvier 2022. Ses restrictions n° 1 et n° 3 comportent une exception supplémentaire, « (c) as permitted by applicable law »,\[2\] que la version US du 15 décembre 2023 n'a pas.\[1\] Pour un utilisateur européen, une exception légale (fouille de textes et de données, copie transitoire) pourrait donc en théorie lever l'interdiction contractuelle. Je n'ai pas pu vérifier si cette version de 2022 est toujours la version EEE en vigueur.
3. **Les Developer Policies sont plus strictes que les CGU** et s'appliquent à blunderDB, qui embarque l'IFrame Player API. Leur révision de janvier 2017 précise que les « API Data » incluent le contenu audiovisuel.\[3\] Dériver des dés ou des coups à partir des images tombe donc sous III.E.4.h.
4. **Les storyboards ne sont pas exploitables.** Ils ne sont exposés par aucune API documentée. On ne les obtient qu'en lisant la réponse interne `player`/`get_video_info` (`storyboards.playerStoryboardSpecRenderer.spec`). Leur résolution maximale courante est de 160×90 px, une vignette toutes les ~10 s,\[4\] ce qui ne suffit pas pour lire des dés.
5. **YouTube durcit l'accès technique (2024-2026), ce qui pèse juridiquement.** Depuis le 29 avril 2025, le client `web` ne fournit plus que des flux SABR.\[5\] Des PO tokens (attestation d'origine) sont liés à chaque vidéo. Selon le guide de yt-dlp, le client `tv` sans cookies reçoit des formats « DRM'd ».\[6\] Plus les mesures techniques s'accumulent, plus il est facile de qualifier un téléchargeur d'outil de contournement de « mesures techniques efficaces ».
6. **La jurisprudence pèse contre tout ce qui facilite le téléchargement.** Le LG Hamburg (310 O 316/21, mars 2023) puis l'OLG Hamburg (5 U 54/23, 21 novembre 2024, sans autorisation de pourvoi devant le BGH) ont jugé l'hébergeur Uberspace responsable comme complice (« Teilnehmer im Sinne der Beihilfe ») pour avoir seulement hébergé le site de youtube-dl.\[7\]\[8\]\[9\]\[10\] Aux États-Unis, le district du Connecticut a jugé en 2022 que le « rolling cipher » est une mesure technique au sens du §1201(a).\[11\] L'appel (2d Cir. n° 22-2760) n'était toujours pas tranché au printemps 2026.\[12\]\[13\]

## Details

### 1. Conditions d'utilisation : clauses précises

**YouTube Terms of Service, version US (« Dated: December 15, 2023 », section « Your Use of the Service » → « Permissions and Restrictions »).** Le principe est le suivant : « You may access and use the Service as made available to you […] You may view or listen to Content for your personal, non-commercial use. You may also show YouTube videos through the embeddable YouTube player. »\[1\] Les restrictions qui concernent la question :
- n° 1 : interdiction d'« access, reproduce, download, distribute, transmit, broadcast, display, sell, license, alter, modify or otherwise use any part of the Service or any Content except: (a) as expressly authorized by the Service; or (b) with prior written permission from YouTube and, if applicable, the respective rights holders ».\[1\] C'est la clause qui vise à la fois le téléchargement et l'« utilisation autrement » qu'est une analyse automatique.
- n° 2 : interdiction de « circumvent, disable, fraudulently engage with, or otherwise interfere with any part of the Service […] including security-related features or features that (a) prevent or restrict the copying or other use of Content ».\[1\]
- n° 3 : interdiction d'« access the Service using any automated means (such as robots, botnets or scrapers) » sauf moteurs de recherche publics selon robots.txt, ou permission écrite préalable.\[1\]
- n° 4 : interdiction de collecter « any information that might identify a person (for example, usernames or faces) ».\[1\] Ce point compte si l'analyse traite des visages de joueurs.
- n° 9 : usage uniquement « personal, non-commercial ».\[1\]
- Section « License to Other Users » : licence d'usage « only as enabled by a feature of the Service (such as video playback or embeds) […] this license does not grant any rights or permissions for a user to make use of your Content independent of the Service ».\[1\]

**Version EEE (Google Ireland Limited, « Dated: January 5, 2022 », traduction anglaise de courtoisie).** Les restrictions n° 1 et n° 3 se terminent par « or (c) as permitted by applicable law ». La n° 2 est quasiment identique.\[2\] La version US de 2023 ne contient pas ce (c). C'est une contradiction entre versions, que je signale sans la trancher (voir Incertitudes).

**Historique (2015-2017).** Les anciennes CGU disposaient : « You shall not download any Content unless you see a "download" or similar link displayed by YouTube on the Service for that Content » et « You agree not to access Content through any technology or means other than the video playback pages of the Service itself, the Embeddable Player, or other explicitly authorized means » (texte archivé, version 2013-2017).\[14\]\[15\] Le sens n'a pas changé, seule la rédaction a évolué.

**YouTube API Services, Developer Policies** (page en vigueur consultée le 30 octobre 2026 ; la date de dernière révision n'est pas affichée dans l'extrait ; l'ensemble des Terms et Policies est entré en vigueur le 10 février 2017 selon la Revision History)\[3\] :
- **III.A.1** : le client doit afficher un lien vers les CGU YouTube et indiquer que l'utilisateur s'y engage.\[16\] blunderDB doit donc déjà le faire.
- **III.D.7 « Undocumented Services »** : « You must not use undocumented APIs without express permission […] You must not reverse engineer undocumented YouTube API services ».\[16\]
- **III.E « Handling YouTube Data and Content »**, préambule : « you and your API Clients have no further permissions or rights to API Data, including to temporarily stored API Data ».\[16\]
- **III.E.1 « Audiovisual Content »** : « must not, and must not encourage, enable, or require others to: download, import, backup, cache, or store copies of YouTube audiovisual content without YouTube's prior written approval, make content available for offline playback […] ».\[16\] La formule « encourage, enable » vise directement le fait de *suggérer* un outil.
- **III.E.4.h** : « must not […] access or use API Data to create new or derived data or metrics ».\[16\] Depuis la révision du 27 janvier 2017, « YouTube API Data includes YouTube audiovisual content ».\[3\]
- **III.E.6 « Scraping »** : interdiction de « directly or indirectly, scrape YouTube Applications […] or obtain scraped YouTube data or content ».\[16\]
- **I. Additional Prohibitions**, points 7-8 (numérotation reprise de citations tierces, à confirmer sur la page) : ne pas « separate, isolate, or modify the audio or video components » ni les promouvoir séparément.\[17\]\[18\]
- Le guide « Complying with YouTube's Developer Policies » (publié le 15 juin 2020) donne comme exemples de violation le fait de « Allow users to download videos for offline play outside of the YT Premium experience » et de « separate or isolate video or audio components ».\[3\]\[19\]

**YouTube API Services Terms of Service.** Les versions sont régionales (Americas, EMEA soumise au droit anglais, APAC). Les modifications prennent effet au plus tôt 30 jours après publication.\[20\]\[21\] Je n'ai trouvé aucune clause qui autorise une analyse d'images.

### 2. IFrame Player API, Data API, storyboards, cross-origin

- **IFrame Player API** : elle n'expose que l'état du lecteur, le temps, la vitesse, la durée et les métadonnées. Je n'ai trouvé aucune méthode qui renvoie des pixels, des images ou un flux. Cela correspond à ce que vous observez avec `postMessage`.
- **Data API** : la ressource `videos` fournit des miniatures fixes (`default`, `medium`, `high`, `standard`, `maxres`, jusqu'à 1280×720). Ce sont des vignettes choisies ou générées une fois pour toute la vidéo, sans rapport avec le déroulé d'une partie. Je n'ai pas récupéré la page de référence pendant cette recherche : les valeurs viennent de la documentation connue et restent à revérifier. Je n'ai trouvé aucun champ `storyboard` ni aucune « image à intervalle » dans une API documentée.
- **Storyboards (sprites de survol)** : ils ne sont accessibles que de façon non officielle. Le parseur `yt-storyboard` (npm/GitHub) lit `storyboards.playerStoryboardSpecRenderer.spec` dans l'API interne `get_video_info`.\[4\] Invidious les récupère via l'endpoint interne `player` d'InnerTube.\[22\] Exemple de spec : `48#27#100#10#10#0#default | 80#45#102#10#10#10000#M$M | 160#90#102#5#5#10000#M$M`.\[4\] Cela donne trois niveaux : 48×27 en grilles 10×10, 80×45 en grilles 10×10, et 160×90 en grilles 5×5, avec un intervalle de 10 000 ms dans cet exemple. Les descriptions disponibles parlent de « maximum 25 stills » par feuille en haute qualité.\[23\] À 160×90 et une image toutes les ~10 s, on ne peut ni lire les dés ni suivre les coups. L'usage viole en outre III.D.7 (API non documentée) et la restriction n° 1 des CGU. **Verdict : inutilisable, juridiquement et techniquement.**
- **Cross-origin** : la politique de même origine interdit au document parent de lire le DOM de l'iframe `youtube.com`, donc l'élément `<video>`. Un canvas ne peut pas dessiner une iframe. La question du *taint* (canvas « sali » par une image ou vidéo cross-origin sans CORS) ne se pose même pas : on ne peut pas atteindre l'élément vidéo. **Aucun accès aux pixels depuis la page.**

### 3. Capture d'écran de la fenêtre

| Webview | Support documenté | Conditions | Dialogue utilisateur |
|---|---|---|---|
| **WebView2 (Windows)** | `getDisplayMedia` est pris en charge (Chromium). L'événement `ScreenCaptureStarting` existe sur `CoreWebView2` et `CoreWebView2Frame` : expérimental dans les SDK prerelease 1.0.2646 à 1.0.2839, stable à partir de 1.0.2903.40. Il permet `Cancel` (renvoie `NotAllowedError`), `Handled` et `GetDeferral`.\[24\]\[25\] | Aucune dépendance supplémentaire. Le sélecteur Chromium s'affiche. | Sélecteur écran/fenêtre/onglet de Chromium, à chaque session. |
| **WebKitGTK (Linux)** | `getDisplayMedia` existe, mais « Desktop portal (PipeWire) mandatory » (FOSDEM 2025, Philippe Normand / Igalia). Le WebKit Contributors Meeting 2023 décrit « two layers of permissions » : la demande de permission WebKit plus le portail.\[26\]\[27\] | `enable-media-stream` vaut `FALSE` par défaut dans l'API 4.x et `TRUE` dans l'API 6.0 (2.46). L'embarqueur doit gérer le signal `permission-request`, faute de quoi la demande est refusée silencieusement.\[28\]\[29\]\[30\] Il faut aussi les plugins GStreamer `pipewiresrc` et une WebKitGTK compilée avec WebRTC/MediaStream. Wails v2 ne gère pas ce signal pour la capture d'écran (non trouvé). | Sélecteur du portail xdg-desktop-portal (GNOME/KDE), qui peut mémoriser le choix.\[27\] Sous X11 sans backend screencast du portail : non documenté, non trouvé. |
| **WKWebView (macOS)** | Safari prend en charge `getDisplayMedia` depuis Safari 13 (WWDC 2019, session 518).\[31\]\[32\] Dans WKWebView, la demande est refusée sauf si le délégué UI implémente la méthode **privée** `_webView:requestDisplayCapturePermissionForOrigin:…` (Switchboard PR #21 ; Tauri issue #2600).\[33\]\[34\] | Autorisation TCC « Enregistrement de l'écran » (`CGRequestScreenCaptureAccess`).\[33\] Une API privée peut casser à chaque version de macOS\[34\] et pose problème pour le Mac App Store. Il faudrait du code Objective-C, donc du CGO, ce qui va contre votre contrainte. | Invite TCC système, puis sélecteur macOS. Un redémarrage de l'app est souvent nécessaire après la première autorisation. |

- **Contradiction entre sources** : un commentaire dans l'issue Tauri #2600 affirme que « Linux does not have webrtc support yet ».\[33\] Les sources Igalia (2023-2025) documentent au contraire `getDisplayMedia` dans WebKitGTK.\[27\]\[35\] Les deux sont vrais selon les options de compilation de la distribution : la disponibilité varie d'une distribution à l'autre. Un ticket tiers (pacto-app #363) indique par ailleurs que wry ≥ 0.56 a dû ajouter une détection de permission DisplayCapture pour WebKitGTK < 2.42.\[36\]
- **Résolution et cadence** : celles de la fenêtre capturée, typiquement 30 images/s côté Chromium. Je n'ai pas trouvé de chiffres documentés propres à chaque webview. Pour lire des dés, il faut que le lecteur soit affiché en grand, sans recouvrement ni contrôles par-dessus.
- **Expérience utilisateur réaliste** : sous Windows, un sélecteur à chaque ouverture. Sous Linux, un portail plus des prérequis système souvent absents. Sous macOS, une autorisation système intimidante, un redémarrage et une API privée. Il faut aussi garder la fenêtre visible et non masquée pendant toute la lecture.
- **Contournement ou non ?** Pour une vidéo **non protégée**, YouTube ne chiffre pas l'affichage. Capturer ce que l'écran montre ne neutralise ni le « rolling cipher » ni les PO tokens, qui encadrent la récupération des flux et non leur affichage. Au sens strict, ce n'est donc pas un contournement d'une mesure technique (restriction n° 2 ; art. 6 de la directive 2001/29 ; §1201). Pour les vidéos sous **DRM** (Widevine, contenus payants), la capture renvoie en principe des images noires dans les navigateurs Chromium. Je n'ai pas trouvé de documentation primaire sur ce comportement dans les webviews, et le contourner serait clairement illicite. En revanche, la capture **reste contraire aux CGU** : c'est un « use » non « expressly authorized » (n° 1), automatisé (n° 3). Pour blunderDB, c'est aussi la création de « derived data » à partir des API Data (III.E.4.h). **Je n'ai trouvé aucune position publique de YouTube sur la capture d'écran en tant que telle.**

### 4. Téléchargement par un outil tiers (yt-dlp et équivalents)

**Mesures techniques de YouTube (2024-2026), d'après le dépôt yt-dlp :**
- Le cadre de plugins PO Token est apparu pour yt-dlp 2024.09.27.\[37\]
- Le 29 avril 2025, « the web client has removed all https formats (adaptiveFormats) and only has SABR protocol streaming url ».\[5\]
- En septembre 2025, SABR est imposé même avec cookies Premium et PO token (issue #14390 : « SABR is a streaming protocol developed by YouTube which yt-dlp does not yet support »).\[38\]
- Le guide PO Token actuel indique que « YouTube now binds PO Tokens to the video ID ». Pour le client `tv` : « All formats DRM'd if cookies (logged-in or active guest) aren't passed ».\[6\]

**Action de Google :** je n'ai trouvé **aucune action en justice de Google contre yt-dlp** (non trouvé). En juin 2023, l'équipe juridique de YouTube a écrit au projet Invidious (issue #3872 du 8 juin 2023) en invoquant les CGU et les API Terms. Les mainteneurs ont répondu ne jamais les avoir acceptées et ne pas utiliser l'API.\[39\]

**États-Unis :**
- RIAA contre youtube-dl / GitHub : avis DMCA du 23 octobre 2020 fondé sur le §1201. Le dépôt a été rétabli le 16 novembre 2020, 24 jours après l'avis, à la suite d'une lettre de l'EFF expliquant que la manière dont youtube-dl « watches » une vidéo « is indistinguishable to YouTube from a normal user » (TechCrunch, 16 novembre 2020), et qu'il n'y a pas de contournement car le flux n'est pas chiffré. Abby Vollmer, directrice Platform Policy de GitHub, a annoncé le 16 novembre 2020 que « GitHub will establish and donate $1M to a developer defense fund to help protect open source developers on GitHub from unwarranted DMCA Section 1201 takedown claims », ainsi qu'un examen technique et juridique des demandes fondées sur le §1201 ; le PDG Nat Friedman ajoutait : « Section 1201 of the DMCA is broken and needs to be fixed. »
- **Yout LLC v. RIAA** : le D. Conn. (juge Underhill, 30 septembre 2022) a jugé que Yout n'avait pas plausiblement allégué l'absence de mesure technique : « even if the YouTube technological measure can be circumvented, it may still be effective ».\[11\]\[40\]\[41\] En appel (2d Cir. n° 22-2760), l'audience a eu lieu le 5 février 2024.\[42\] Un amicus tardif de Suno et Udio (octobre 2025) a relancé le débat entre contrôle d'accès et contrôle de copie.\[43\]\[44\] Des lettres 28(j) ont invoqué *Cox v. Sony* (Cour suprême, 25 mars 2026) en avril 2026.\[12\]\[13\]\[43\]\[45\] **Aucune décision d'appel trouvée au 30 octobre 2026.**
- Dans l'affaire Udio (S.D.N.Y., réponse des maisons de disques d'octobre 2025), les plaignants invoquent le contournement du rolling cipher.\[46\]

**Allemagne :**
- LG Hamburg, 310 O 316/21 (fin mars 2023) : Uberspace doit cesser d'héberger youtube-dl.org et répond comme complice du contournement de « wirksame technische Schutzmaßnahmen ». Une ordonnance de référé de 2017 avait déjà qualifié le rolling cipher de mesure technique efficace.\[7\]\[47\]\[48\]
- OLG Hamburg, 5 U 54/23, 21 novembre 2024 : appel rejeté, responsabilité en dommages-intérêts admise en principe, pas d'autorisation de « Revision » devant le BGH.\[9\]\[10\]

**UE :**
- Directive 2001/29, art. 6 : protection juridique des mesures techniques efficaces.\[49\]
- Directive 2019/790 : l'art. 3 (fouille de textes et de données, TDM, par les organismes de recherche) et l'art. 4 (TDM général, sous réserve d'un « accès licite » et de l'absence de réserve des droits « par des procédés lisibles par machine » pour les contenus en ligne) pourraient couvrir l'extraction d'informations à partir d'images. Mais selon l'art. 7(1), seules les clauses contractuelles contraires aux art. 3, 5 et 6 sont inopposables, pas celles contraires à l'art. 4 (texte rappelé de mémoire, non relu pendant cette recherche). Pour un usage grand public, qui relève de l'art. 4, les CGU peuvent donc écarter l'exception.
- CJUE, **C-496/24, Stichting de Thuiskopie, 16 avril 2026** (ECLI:EU:C:2026:296) : les copies « hors ligne » chiffrées fournies par une plateforme de streaming ne relèvent pas de l'exception de copie privée. Elles relèvent de la mise à disposition par le fournisseur et ne sont pas réalisées par la personne physique.\[49\]\[50\]\[51\] Cela confirme que la copie privée ne sert pas à valider des copies à partir d'une plateforme qui en contrôle l'usage.\[51\]\[52\]

**France :**
- CPI L.122-5 2° : copies « réalisées à partir d'une source licite », condition introduite par la loi n° 2011-1898 du 20 décembre 2011, après Cass. crim. 30 mai 2006.\[53\]\[54\]
- CPI L.122-5 6° : reproduction provisoire transitoire.\[53\]
- CPI L.331-5 et suivants : définition et protection des mesures techniques. Les sanctions pénales du contournement et de la fourniture de moyens figurent à L.335-3-1 et L.335-3-2, et l'incitation à l'usage d'un logiciel manifestement destiné à la mise à disposition non autorisée d'œuvres à L.335-2-1. Je n'ai pas relu ces textes sur Légifrance pendant cette recherche : à vérifier avant toute publication.
- La source YouTube est licite pour la *visualisation*, mais un fichier obtenu en contournant une mesure technique n'est pas une base solide pour invoquer la copie privée.\[55\]\[56\]

**Risque pour une application qui *suggère* un outil sans l'embarquer :**
- Côté contrat : III.E.1 interdit d'« encourage, enable » le téléchargement. Une simple mention viole donc déjà les Developer Policies, et YouTube peut couper l'accès du client.
- Aux États-Unis : *MGM v. Grokster* (2005) rend responsable celui qui diffuse un moyen « with the object of promoting its use to infringe », preuve à l'appui d'expressions ou d'actes positifs. Le §1201(a)(2) vise aussi quiconque « offers to the public, provides, or otherwise traffics ».
- En Allemagne : l'affaire Uberspace montre qu'un simple hébergement du site d'un téléchargeur suffit à caractériser une complicité.\[7\]
- Je n'ai trouvé aucune décision visant un logiciel *tiers* qui se borne à recommander yt-dlp (non trouvé). Le risque reste réel et totalement inutile.

**Pratique des outils d'annotation sportive (Kinovea, LongoMatch, Hudl, tennis, football) :** je n'ai trouvé pendant cette recherche aucune documentation de ces éditeurs sur YouTube (non trouvé). À ma connaissance, non vérifiée ici, ces outils travaillent sur des fichiers importés ou des flux de caméras propres au club, c'est-à-dire sur des sources possédées par l'utilisateur.

### 5. Alternatives honnêtes

- **Fichier fourni par l'utilisateur** : c'est la voie propre. Il peut s'agir de l'enregistrement de *sa propre* table ou caméra, ou du fichier source transmis par le streamer ou l'organisateur. Attention : un enregistrement OBS d'une vidéo YouTube d'un tiers reste une reproduction du contenu YouTube, contraire à la restriction n° 1. Il ne faut donc pas la recommander.
- **YouTube et Creative Commons** : YouTube propose une seule licence CC, « Creative Commons Attribution » (CC BY), qui permet à d'autres créateurs de « reuse your work subject to the terms of the CC BY license » (Aide YouTube).\[57\]\[58\]\[59\] Il y a une contradiction de version : Wikimedia Commons parle de CC BY 4.0 tout en renvoyant au modèle {{YouTube CC-BY}} pour la CC BY 3.0.\[60\] Le CC BY règle la question du **droit d'auteur**, mais pas celle du **contrat** : la restriction n° 1 (version US) exige l'autorisation de YouTube *et* des ayants droit. Je n'ai trouvé aucun bouton de téléchargement pour les vidéos CC destiné au public (non trouvé). En pratique, demander le fichier source au créateur d'une vidéo CC BY est à la fois licite et simple.
- **Twitch** : seuls le propriétaire de la chaîne et ses Editors peuvent télécharger un VOD, via Creator Dashboard → Content → Video Producer → « Download ».\[61\] Selon des sources secondaires, les VOD ne sont pas téléchargeables via l'API publique.\[62\] Les CGU Twitch (la page allemande affiche « Last modified on 08/12/2026 », tandis que la page anglaise affiche encore « Last modified on 03/26/2024 » : contradiction non résolue) excluent tout « downloading (except page caching) […] except as expressly permitted ». Pour blunderDB, un streamer peut donc exporter son VOD et le transmettre légalement.
- **Vimeo (bouton de téléchargement activé par le propriétaire), PeerTube (téléchargement réglé par l'instance), Kick** : je n'ai récupéré aucun texte primaire pendant cette recherche (non trouvé). À vérifier au cas par cas : un téléchargement proposé par la plateforme elle-même est en principe la situation « expressly authorized ».
- **Chaînes de backgammon sous licence permettant le téléchargement (Backgammon Galaxy, USBGF, WBF, Backgammon Masters, archives CC)** : non trouvé. Je n'ai identifié aucune archive publique de matchs vidéo sous licence ouverte. La démarche réaliste consiste à demander directement aux organisateurs une licence ou le fichier source.

### 6. Prior art

- **Extensions de navigateur** : `chrome.tabCapture` « captures the visible area of the currently active tab ». La capture ne démarre qu'après une action explicite de l'utilisateur, sur le modèle d'`activeTab`.\[63\] Le projet open source **fenshot** (reconnaissance d'échiquier, en local, extension Chrome/Firefox/Edge) propose une « one-click scan of the visible page » : il analyse ce que l'utilisateur voit, sans télécharger.\[64\] C'est la voie la plus proche d'un prior art « propre ». Je n'ai trouvé aucun texte où YouTube l'autorise, et l'extension ne vise pas spécifiquement les vidéos YouTube.
- **Recherche académique** :
  - Google n'a pas diffusé les vidéos de YouTube-8M (arXiv 1609.08675, 2016). Il a publié des caractéristiques précalculées, extraites à 1 image/s sur les 360 premières secondes (« we decode each video at 1 frame-per-second up to the first 360 seconds (6 minutes) », §3.3), soit plus de 1,9 milliard d'images pour environ 8 millions de vidéos ; la licence Apache 2.0 évoquée par ailleurs n'apparaît pas dans l'article.
  - Kinetics diffuse des listes d'identifiants. Le téléchargeur tiers renvoie l'utilisateur aux CGU (« be sure that you have checked YouTube Terms Of Service and you are compliant »).\[65\]\[66\]
  - AViD (NeurIPS 2020) relève que « downloading videos from YouTube is often blocked » et a choisi des vidéos sous licence Creative Commons venant d'autres sources, ce qui lui permet de « download, modify and distribute ».\[67\]
  - V3C (arXiv 1810.04401) cite les CGU YouTube dans sa justification.\[68\]
  - La recherche sérieuse contourne donc le problème en changeant de source, ou en ne distribuant que des caractéristiques issues d'une partie liée à YouTube.
- **Partenariat** : le seul chemin contractuellement propre est la « prior written permission from YouTube », plus celle des ayants droit. Je n'ai trouvé aucun programme public de ce type pour l'analyse d'images.

## Ce que la documentation de blunderDB peut annoncer

Formulation recommandée pour la documentation (sans exposition de l'auteur) :

> « L'analyse automatique des images (détection des lancers, lecture des dés, reconnaissance des coups et du videau) fonctionne uniquement sur des fichiers vidéo locaux que vous avez le droit d'utiliser : vos propres enregistrements, ou des fichiers fournis par leur auteur ou l'organisateur du tournoi. Pour les vidéos YouTube, blunderDB affiche le lecteur officiel de YouTube et ne lit que la position de lecture, l'état et la vitesse ; il n'accède pas aux images, ne télécharge rien, ne conserve aucune copie et ne recommande aucun outil de téléchargement. L'utilisation des vidéos YouTube est soumise aux Conditions d'utilisation de YouTube (https://www.youtube.com/t/terms). »

À éviter dans toute la documentation, le dépôt, les issues et le code : toute mention ou tout lien vers yt-dlp, youtube-dl ou un « convertisseur », ainsi que toute instruction pour « enregistrer l'écran pendant la lecture YouTube ».

## Message recommandé à l'utilisateur (mode assisté, vidéo YouTube)

> **Mode assisté — vidéo YouTube**
> Pour respecter les conditions d'utilisation de YouTube, blunderDB n'analyse pas les images des vidéos YouTube. Vous saisissez les dés ; blunderDB se cale sur le temps de la vidéo et vous propose les coups légaux classés.
> Pour profiter de la détection automatique des dés et des coups, ouvrez un fichier vidéo local dont vous disposez légitimement (par exemple le fichier que le streamer ou l'organisateur peut vous transmettre, ou votre propre enregistrement).
> [Continuer avec YouTube] [Ouvrir un fichier local…] [En savoir plus]

## Verdict

**Non.** En octobre 2026, il n'existe pas d'accès légitime aux images d'une vidéo YouTube pour une analyse automatique dans blunderDB, sauf autorisation écrite de YouTube et des ayants droit. Les CGU (restrictions n° 1 et n° 3, version US du 15 décembre 2023) et les Developer Policies (III.D.7, III.E.1, III.E.4.h, III.E.6) ferment toutes les voies : API, storyboards, téléchargement, capture. La capture d'écran n'est pas un contournement de DRM pour les vidéos non protégées, mais elle reste un usage non autorisé du service. Le seul point ouvert est la clause « as permitted by applicable law » de la version EEE, qui pourrait jouer avec une exception légale : je la juge trop incertaine pour bâtir un produit dessus.

## Recommandation technique

1. **Séparer strictement les deux modes.** Le pipeline d'analyse (décodage par `<video>` local, WebCodecs ou canvas, dans la webview, sans CGO) ne s'active que pour un `File` local. Le mode YouTube reste limité à `postMessage` (temps, état, vitesse) et à la saisie clavier, plus le classement par le réseau de neurones.
2. **Ne pas implémenter `getDisplayMedia` pour YouTube**, même là où c'est faisable (WebView2). Les gains (sélecteur à chaque session, fenêtre à garder visible) ne compensent ni la violation de III.E.4.h ni les écarts entre plateformes. Sous macOS, il faudrait une API privée et du CGO.
3. **Supprimer toute aide au téléchargement**, même indirecte : pas de lien, pas de « conseil », pas d'import d'URL qui appellerait un outil externe.
4. **Faciliter l'obtention licite des fichiers** : glisser-déposer, import de VOD Twitch exportés par le streamer, et une page « Demander le fichier à l'organisateur » avec un modèle de message.
5. **Se conformer à III.A** : afficher un lien vers les CGU YouTube et une politique de confidentialité mentionnant l'usage des YouTube API Services.

## Incertitudes (et expérience à mener pour chacune)

1. **Version EEE des CGU (5 janvier 2022 contre 15 décembre 2023 aux États-Unis, clause « (c) as permitted by applicable law »)** : ouvrir youtube.com/t/terms depuis une IP française, sans compte puis connecté. Noter la date « Dated » et le texte exact des restrictions n° 1 à 3, et archiver une capture.
2. **Numérotation exacte et date de révision des Developer Policies (I.7-I.8, III.E.4.h)** : lire la page complète et la Revision History, ou s'abonner au flux RSS, puis consigner la date de la dernière modification.
3. **`getDisplayMedia` sous Wails v2 et WebKitGTK** : construire un prototype minimal sur Ubuntu 24.04 (Wayland, puis X11) et Fedora (WebKitGTK 2.4x). Vérifier si l'appel renvoie un flux, `NotAllowedError` ou `undefined`, sans modifier Wails. Recommencer en activant `enable-media-stream` via un patch local, pour mesurer l'effort.
4. **WebView2 sous Wails v2** : vérifier que le sélecteur Chromium apparaît par défaut, si `preferCurrentTab`/`selfBrowserSurface` fonctionne et quelle cadence effective on obtient (`track.getSettings().frameRate`).
5. **Images noires sur vidéos DRM dans les webviews** : tester un film YouTube loué (Widevine) dans WebView2 avec capture de fenêtre, et constater si l'image est noire. À n'utiliser que comme garde-fou, pas comme fonctionnalité.
6. **Statut d'appel de Yout v. RIAA (2d Cir. 22-2760)** : consulter directement le docket (CourtListener/PACER) chaque trimestre.
7. **Textes français L.331-5, L.335-2-1, L.335-3-1 et L.335-3-2, et L.122-5-3 (TDM)** : relire les versions consolidées sur Légifrance avant de publier la documentation, et faire valider la formulation par un avocat en propriété intellectuelle si l'application devient commerciale.
8. **Archives de matchs de backgammon sous licence ouverte** : écrire à l'USBGF, à la WBF, à Backgammon Galaxy et aux organisateurs de tournois diffusés pour demander le fichier source ou une licence. Recenser les vidéos YouTube CC BY de backgammon avec le filtre « Creative Commons » de la recherche YouTube.
9. **Twitch, Vimeo, PeerTube, Kick** : relire les CGU primaires de chaque plateforme (sections licence et téléchargement) et noter la date de version.

## Sources

1. YouTube Terms of Service (Google LLC), « Dated: December 15, 2023 », https://www.youtube.com/static?template=terms. Contient « Permissions and Restrictions » n° 1-10 et « License to Other Users ». Primaire.
2. YouTube Terms of Service, version EEE (Google Ireland), « Dated: January 5, 2022 », traduction de courtoisie, https://youtube.com/static?gl=SE&hl=en&template=terms. Restrictions n° 1 et n° 3 avec « (c) as permitted by applicable law ». Primaire.
3. YouTube API Services – Developer Policies, Google for Developers, consulté le 30 octobre 2026, https://developers.google.com/youtube/terms/developer-policies. Sections III.A.1, III.D.7, III.E.1, III.E.4.h, III.E.6. Primaire.
4. Complying with YouTube's Developer Policies, Google, 2020, https://developers.google.com/youtube/terms/developer-policies-guide. Exemples d'interdictions (téléchargement, séparation audio/vidéo). Primaire.
5. YouTube API Services Terms of Service – Revision History, Google, 2017-2020, https://developers.google.com/youtube/terms/revision-history. Entrée en vigueur au 10 février 2017 ; les « API Data » incluent le contenu audiovisuel. Primaire.
6. YouTube API Services Terms of Service (Americas / EMEA), Google, https://developers.google.com/youtube/terms/api-services-terms-of-service et …-emea. Versions régionales, droit anglais pour l'EMEA, préavis de 30 jours. Primaire.
7. Anciennes CGU YouTube (archive tosback2, vers 2013-2017), https://github.com/tosdr/tosback2/…/youtube.com/Terms%20of%20Service.txt. « You shall not download any Content unless you see a "download" […] link ». Primaire (archive).
8. « License types on YouTube », Aide YouTube, https://support.google.com/youtube/answer/2797468. Licence CC BY et réutilisation. Primaire.
9. Commons:YouTube files, Wikimedia Commons, https://commons.wikimedia.org/wiki/Commons:YouTube_files. CC BY 4.0 contre 3.0, procédure de revue de licence. Secondaire.
10. yt-dlp issue #12482 « web only has SABR formats », 29 avril 2025, https://github.com/yt-dlp/yt-dlp/issues/12482. Fin des formats https côté web. Primaire (dépôt).
11. yt-dlp issue #14390, septembre 2025, https://github.com/yt-dlp/yt-dlp/issues/14390. SABR imposé même avec Premium et PO token. Primaire (dépôt).
12. yt-dlp Wiki « PO Token Guide », 2025-2026, https://github.com/yt-dlp/yt-dlp/wiki/Po-Token-Guide. Tokens liés à la vidéo ; client tv « DRM'd » sans cookies. Primaire (dépôt).
13. Invidious issue #3872 « YouTube legal team contacted us », 8 juin 2023, https://github.com/iv-org/invidious/issues/3872. Mise en demeure fondée sur les CGU et les API Terms. Primaire (dépôt).
14. GitHub DMCA « 2020-11-16-RIAA-reversal », GitHub, 2020, https://github.com/github/dmca/blob/master/2020/11/2020-11-16-RIAA-reversal.md. Rétablissement de youtube-dl. Primaire.
15. « GitHub Reinstates youtube-dl After RIAA's Abuse of the DMCA », EFF, novembre 2020, https://www.eff.org/deeplinks/2020/11/github-reinstates-youtube-dl-after-riaas-abuse-dmca. Argument d'absence de contournement. Secondaire (partie prenante).
16. « Gerichtsurteil: Musikindustrie geht erfolgreich gegen Youtube-Downloader vor », netzpolitik.org, 2023, https://netzpolitik.org/2023/gerichtsurteil-musikindustrie-geht-erfolgreich-gegen-youtube-downloader-vor/. LG Hamburg, complicité de l'hébergeur. Secondaire.
17. « Landgericht Hamburg: Uberspace darf Youtube-DL nicht länger hosten », heise online, 2023, https://www.heise.de/news/Landgericht-Hamburg-Uberspace-darf-Youtube-DL-nicht-laenger-hosten-8529311.html. Az. 310 O 316/21. Secondaire.
18. « Entscheidung des OLG Hamburg: youtube-dl.org bleibt gesperrt », netzpolitik.org, novembre 2024, https://netzpolitik.org/2024/entscheidung-des-olg-hamburg-youtube-dl-org-bleibt-gesperrt/. Appel rejeté. Secondaire. Référence 5 U 54/23 du 21 novembre 2024 relevée via RA M. Seidlitz (X) renvoyant à openjur.de/u/2498326. Décision non lue directement.
19. Yout LLC v. RIAA, D. Conn., 30 septembre 2022, https://www.courtlistener.com/opinion/9631682/yout-llc-v-recording-industry-association-of-america-inc/. Le rolling cipher est une mesure technique (§1201(a)). Primaire.
20. Mémoires d'appel Yout et RIAA, 2d Cir. n° 22-2760, Courthouse News, 2024, https://www.courthousenews.com/wp-content/uploads/2024/02/youts-RIAA-22-2760-appellee-brief.pdf. Positions des parties. Primaire.
21. « Yout and RIAA Clash in Court Over YouTube's Alleged Copyright Barriers », TorrentFreak, novembre 2025, et « Yout.com hopes Supreme Court's Cox ruling helps its case », 3 avril 2026, https://torrentfreak.com/. Appel pendant, amicus Suno/Udio, lettres sur Cox. Secondaire.
22. Réponse des maisons de disques à la requête d'Udio, S.D.N.Y., octobre 2025, https://www.musicbusinessworldwide.com/files/2025/10/Record-co-response-to-Udio-motion-to-dismiss.pdf. Le rolling cipher comme contrôle d'accès. Primaire.
23. CJUE, C-496/24, Stichting de Thuiskopie e.a. c. HP Nederland e.a., 16 avril 2026, ECLI:EU:C:2026:296, texte via https://www.ipcuria.eu/case?reference=C-496%2F24. Copies hors ligne de streaming exclues de la copie privée. Primaire. Analyse : J.P. Quintais, Kluwer Copyright Blog, 13 mai 2026. Secondaire.
24. « L'exception de copie privée à l'ère du streaming », Kohen Avocats, juin 2026, https://kohenavocats.fr/2026/06/09/exception-copie-privee-streaming-cjue-stichting-thuiskopie-2026/. Lien avec L.122-5 2° et la source licite. Blog.
25. « Le régime de l'exception de la copie privée en droit français », Droit & Technologies, https://droit-technologie.org/actualites/le-regime-de-lexception-de-la-copie-privee-en-droit-francais/. Cass. crim. 30 mai 2006, source licite. Secondaire.
26. « P2P, streaming et respect du droit d'auteur », Le Petit Juriste, https://www.lepetitjuriste.fr/p2p-streaming-et-respect-du-droit-dauteur-responsabilite-de-linternaute-et-de-lediteur/. Loi n° 2011-1898 et source licite. Secondaire.
27. WebKit Bugzilla 186294 « Implement getDisplayMedia », 2018-2019, https://bugs.webkit.org/show_bug.cgi?id=186294. Livré dans Safari 13. Primaire.
28. WebKit Contributors Meeting 2023, notes, https://docs.webkit.org/Other/Contributor%20Meetings/ContributorMeeting2023.html. Double permission WebKit et portail sous Linux. Primaire.
29. « WebRTC support in WebKitGTK and WPEWebKit with GStreamer », Ph. Normand, FOSDEM 2025, https://archive.fosdem.org/2025/…/WebRTC_su_taGGWpV.pdf. « Desktop portal (PipeWire) mandatory ». Primaire.
30. « WebRTC in WebKitGTK and WPE, status updates, part I », Base-Art (Igalia), https://base-art.net/Articles/webrtc-in-webkitgtk-and-wpe-status-updates-part-i/. `getDisplayMedia` via pipewiresrc. Blog (développeur du projet).
31. WebKitGTK reference, « Settings:enable-media-stream », https://webkitgtk.org/reference/webkit2gtk/stable/property.Settings.enable-media-stream.html. Défaut FALSE (API 4.x). Copie fossies pour l'API 6.0 2.46.3 : défaut TRUE. Primaire.
32. Microsoft Learn, « CoreWebView2.ScreenCaptureStarting » et « CoreWebView2ScreenCaptureStartingEventArgs », https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/winrt/microsoft_web_webview2_core/corewebview2screencapturestartingeventargs. Cancel, Handled, Deferral. Primaire.
33. « What's New for Web Developers », WWDC19 session 518, Apple, 2019, https://developer.apple.com/videos/play/wwdc2019/518/. `getDisplayMedia` dans Safari 13. Primaire.
34. Tauri issue #2600 « getDisplayMedia() has permission issue », https://github.com/tauri-apps/tauri/issues/2600. WKWebView refuse sans délégué privé ; avis contradictoire sur Linux. Dépôt.
35. Switchboard PR #21 « Allow screen sharing in web views », https://github.com/bradymholt/switchboard/pull/21. Méthode privée `_webView:requestDisplayCapturePermissionForOrigin:`. Dépôt.
36. Wails, « Permissions » (documentation), https://wails.io/features/windows/permissions/, et discussion #3579 (avril 2026), https://github.com/wailsapp/wails/discussions/3579. Absence d'invite native WebKitGTK ; refus silencieux. Primaire (projet), version de Wails à confirmer.
37. pacto-app issue #363, https://github.com/covenant-gov/pacto-app/issues/363. wry ≥ 0.56 détecte la permission DisplayCapture pour WebKitGTK < 2.42. Dépôt (secondaire).
38. chrome.tabCapture, Chrome for Developers, https://developer.chrome.com/docs/extensions/reference/api/tabCapture. Capture de la zone visible après invocation par l'utilisateur. Primaire.
39. yt-storyboard, E. van Zummeren, https://github.com/evanzummeren/yt-storyboard. Format de la spec et niveaux 48×27, 80×45, 160×90. Dépôt.
40. Invidious, commits « Add video previews » et « Fix storyboards », https://git.linux.ucla.edu/lug/invidious/commit/6d92775ab546969230366b5bddbe51d36f73e7c7. Storyboards via l'endpoint interne `player`. Dépôt.
41. fenshot, S. Coriiu, https://github.com/scoriiu/fenshot. Reconnaissance d'échiquier en local, extension « scan of the visible page ». Dépôt.
42. Abu-El-Haija et al., « YouTube-8M: A Large-Scale Video Classification Benchmark », arXiv:1609.08675, 2016, https://arxiv.org/abs/1609.08675. Diffusion de caractéristiques, pas de vidéos. Primaire.
43. Piergiovanni & Ryoo, « AViD Dataset », NeurIPS 2020, https://papers.neurips.cc/paper_files/paper/2020/file/c28e5b0c9841b5ef396f9f519bf6c217-Paper.pdf. Téléchargement YouTube « often blocked », choix de vidéos CC d'autres sources. Primaire.
44. Rossetto et al., « V3C – a Research Video Collection », arXiv:1810.04401, 2018, https://arxiv.org/abs/1810.04401. Cite les CGU YouTube. Primaire.
45. Showmax, kinetics-downloader, https://github.com/Showmax/kinetics-downloader. Avertissement sur les CGU YouTube. Dépôt.
46. « How to download Twitch VODs before they expire », Clyper, 2026, https://www.clyper.io/guides/how-to-download-twitch-vods, et « How to Download Twitch VODs: A 2026 Creator's Guide », Contesimal, https://contesimal.ai/blog/download-twitch-vods/. Téléchargement limité au propriétaire et à ses Editors ; extraits des CGU Twitch. Blog.
47. « Applying a CC licence on YouTube », Université d'Édimbourg, https://open.ed.ac.uk/how-to-guides/adding-a-creative-commons-licence-to-your-resource/applying-a-cc-licence-on-youtube/. La licence standard interdit le téléchargement ; CC BY. Secondaire.

## Sources

1. [Terms of Service](https://www.youtube.com/static?template=terms)
2. [Terms of Service](https://youtube.com/static?gl=SE&hl=en&template=terms)
3. [YouTube API Services Terms of Service - Revision History | Google for Developers](https://developers.google.com/youtube/terms/revision-history)
4. [yt storyboard](https://www.npmjs.com/package/yt-storyboard)
5. [\[youtube\] \`web\` only has SABR formats · Issue #12482 · yt-dlp/yt-dlp · GitHub](https://github.com/yt-dlp/yt-dlp/issues/12482)
6. [YouTube PO Token Guide - yt-dlp](https://github.com/yt-dlp/yt-dlp/wiki/Po-Token-Guide)
7. [Gerichtsurteil: Musikindustrie geht erfolgreich gegen Youtube-Downloader vor - netzpolitik.org](https://netzpolitik.org/2023/gerichtsurteil-musikindustrie-geht-erfolgreich-gegen-youtube-downloader-vor/)
8. [Entscheidung des OLG Hamburg: youtube-dl.org bleibt gesperrt - netzpolitik.org](https://netzpolitik.org/2024/entscheidung-des-olg-hamburg-youtube-dl-org-bleibt-gesperrt/)
9. [RA Michael Seidlitz on X: "Uberspace haftet wegen Hosting von Youtube-DL von Stefan Krempl https://t.co/fm1nKFWdcm OLG Hamburg, Urteil vom 21.11.2024, 5 U 54/23 https://t.co/xpK4GSQyoi Stellungnahmen der Parteien: https://t.co/aBtasuucnG https://t.co/YtwkVSLPgi" / X](https://x.com/MichaelSeidlitz/status/1862133144373116967)
10. [OLG Hamburg weist Berufung zurück: Host-Provider von YouTube-DL haftet für Urheberrechtsverstöße - Backstage PRO](https://www.backstagepro.de/thema/olg-hamburg-weist-berufung-zurueck-host-provider-von-youtube-dl-haftet-fuer-urheberrechtsverstoesse-2024-11-26-30wX4BmZLs)
11. [RIAA Thwarts Yout's Attempt to Declare YouTube-Ripping Legal \* TorrentFreak](https://torrentfreak.com/riaa-thwarts-youts-attempt-to-declare-youtube-ripping-legal-221002/)
12. [Yout v. RIAA Legal Battle Invokes Supreme Court's Cox Decision](https://www.digitalmusicnews.com/2026/04/13/yout-v-riaa-cox-decision/)
13. [Yout.com Hopes Supreme Court's Cox Ruling Helps Its Case; RIAA Disagrees \* TorrentFreak](https://torrentfreak.com/yout-com-hopes-supreme-courts-cox-ruling-helps-its-case-riaa-disagrees/)
14. [forked from pde/tosback2](https://github.com/tosdr/tosback2/blob/ee541535605b418306ba0961b16290aec4bcbbd2/crawl_reviewed/youtube.com/Terms%20of%20Service.txt)
15. [webarchive.library.unt.edu](https://webarchive.library.unt.edu/web/20130213220921mp_/http://www.youtube.com/t/terms)
16. [YouTube API Services - Developer Policies | Google for Developers](https://developers.google.com/youtube/terms/developer-policies)
17. [The End of ProTube 2](https://news.ycombinator.com/item?id=15164854)
18. [YouTube integration - #89 by danny - Feature Suggestions - Roon Labs Community](https://community.roonlabs.com/t/youtube-integration/24132/89)
19. [Complying with YouTube's Developer Policies | Google for Developers](https://developers.google.com/youtube/terms/developer-policies-guide)
20. [YouTube API Services Terms of Service (EMEA) | Google for Developers](https://developers.google.com/youtube/terms/api-services-terms-of-service-emea)
21. [YouTube API Services Terms of Service | Google for Developers](https://developers.google.com/youtube/terms/api-services-terms-of-service)
22. [Regression from #4037 | Fix storyboards](https://git.linux.ucla.edu/lug/invidious/commit/6b17bb525095a62b163489c565edb0ca29eb1a93)
23. [GitHub - evanzummeren/yt-storyboard: A parser for YouTube storyboards · GitHub](https://github.com/evanzummeren/yt-storyboard)
24. [CoreWebView2.ScreenCaptureStarting Event (Microsoft.Web.WebView2.Core)](https://learn.microsoft.com/en-us/dotnet/api/microsoft.web.webview2.core.corewebview2.screencapturestarting?view=webview2-dotnet-1.0.3405.78)
25. [CoreWebView2ScreenCaptureStartingEventArgs](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/winrt/microsoft_web_webview2_core/corewebview2screencapturestartingeventargs?view=webview2-winrt-1.0.4258.31)
26. [WebRTC support in WebKitGTK and WebRTC support in WebKitGTK and](https://archive.fosdem.org/2025/events/attachments/fosdem-2025-4651-webrtc-support-in-webkitgtk-and-wpewebkit-with-gstreamer-status-update/slides/238203/WebRTC_su_taGGWpV.pdf)
27. [WebKit Contributors Meeting 2023 - WebKit Documentation](https://docs.webkit.org/Other/Contributor%20Meetings/ContributorMeeting2023.html)
28. [property.Settings.enable media stream](https://webkitgtk.org/reference/webkit2gtk/stable/property.Settings.enable-media-stream.html)
29. [fix(viewer): allow microphone access on Linux by zackees · Pull Request #228 · zackees/fastled-wasm · GitHub](https://github.com/zackees/fastled-wasm/pull/228)
30. [WebKitGTK+: .../property.Settings.enable-media-stream.html](https://fossies.org/linux/webkitgtk/Documentation/webkitgtk-6.0/property.Settings.enable-media-stream.html)
31. [What's New for Web Developers - WWDC19 - Videos - Apple Developer](https://developer.apple.com/videos/play/wwdc2019/518/)
32. [186294](https://bugs.webkit.org/show_bug.cgi?id=186294)
33. [github.com](https://github.com/tauri-apps/tauri/issues/2600)
34. [Allow screen sharing in web views by bradymholt · Pull Request #21 · bradymholt/switchboard · GitHub](https://github.com/bradymholt/switchboard/pull/21)
35. [WebRTC in WebKitGTK and WPE, status updates, part I - Base-Art](https://base-art.net/Articles/webrtc-in-webkitgtk-and-wpe-status-updates-part-i/)
36. [Screen sharing (single presenter) for squad voice calls · Issue #363 · covenant-gov/pacto-app · GitHub](https://github.com/covenant-gov/pacto-app/issues/363)
37. [No project description provided](https://pypi.org/project/yt-dlp-get-pot/)
38. [It seems like SABR is forced even with valid premium cookies and PO Token Provider · Issue #14390 · yt-dlp/yt-dlp · GitHub](https://github.com/yt-dlp/yt-dlp/issues/14390)
39. [github.com](https://github.com/iv-org/invidious/issues/3872)
40. [Yout LLC v. Recording Industry Association of America, Inc.](https://www.courtlistener.com/opinion/9631682/yout-llc-v-recording-industry-association-of-america-inc/)
41. [Yout LLC v. Recording Industry Association of America, Inc.](https://www.courtlistener.com/opinion/9631896/yout-llc-v-recording-industry-association-of-america-inc/)
42. [Oral Argument for Yout LLC v. Recording Industry Association of America, Inc.](https://www.courtlistener.com/audio/90475/yout-llc-v-recording-industry-association-of-america-inc/)
43. [Yout and RIAA Clash in Court Over YouTube's Alleged Copyright Barriers \* TorrentFreak](https://torrentfreak.com/yout-and-riaa-clash-in-court-over-youtubes-alleged-copyright-barriers/)
44. [Facing ‘stream-ripping’ allegations from labels, Suno & Udio weigh in on separate case to challenge ruling that makes YouTube downloading illegal - Music Business Worldwide](https://www.musicbusinessworldwide.com/facing-stream-ripping-allegations-from-labels-suno-udio-weigh-in-on-separate-case-to-challenge-ruling-that-makes-youtube-downloading-illegal/)
45. [Cox Communications, Inc. v. Sony Music Entertainment](https://en.wikipedia.org/wiki/Cox_Communications,_Inc._v._Sony_Music_Entertainment)
46. [UNITED STATES DISTRICT COURT FOR THE SOUTHERN DISTRICT OF NEW YORK](https://www.musicbusinessworldwide.com/files/2025/10/Record-co-response-to-Udio-motion-to-dismiss.pdf)
47. [Youtube-DL: Uberspace geht gegen Hosting-Verbot in Berufung](https://www.heise.de/news/Youtube-DL-Uberspace-geht-gegen-Hosting-Verbot-in-Berufung-8651600.html)
48. [Landgericht Hamburg: Uberspace darf Youtube-DL nicht länger hosten](https://www.heise.de/news/Landgericht-Hamburg-Uberspace-darf-Youtube-DL-nicht-laenger-hosten-8529311.html)
49. [CJEU - Judgment C-496/24 Stichting de Thuiskopie 16 Apr 2026](https://www.ipcuria.eu/case?reference=C-496%2F24)
50. [Offline streaming: the CJEU rules out the private copying exception - Nomos](https://www.nomosparis.com/en/offline-streaming-the-cjeu-rules-out-the-private-copying-exception/?print=print)
51. [Streaming Without Copying: The CJEU Redefines Private Use in Stichting de Thuiskopie (C-496/24)](https://legalblogs.wolterskluwer.com/copyright-blog/streaming-without-copying-the-cjeu-redefines-private-use-in-stichting-de-thuiskopie-c-49624/)
52. [L'exception de copie privée à l'ère du streaming : l'arrêt Stichting de Thuiskopie du 16 avril 2026 et le reflux d'une exception historique du droit d'auteur - Maître Reda Kohen, avocat en droit immobilier et droit des affaires à Paris](https://kohenavocats.fr/2026/06/09/exception-copie-privee-streaming-cjue-stichting-thuiskopie-2026/)
53. [P2P, streaming et respect du droit d'auteur: responsabilité de l'internaute et de l'éditeur. - Le petit juriste](https://www.lepetitjuriste.fr/p2p-streaming-et-respect-du-droit-dauteur-responsabilite-de-linternaute-et-de-lediteur/)
54. [Le régime de l’exception de la copie privée en droit français • Droit & Technologies](https://droit-technologie.org/actualites/le-regime-de-lexception-de-la-copie-privee-en-droit-francais/)
55. [La Copie Privée - Lucy Finance](https://lucyfinance.fr/la/)
56. [QU’EST-CE QU’UNE COPIE PRIVEE ?](https://www.murielle-cahen.fr/quest-ce-quune-copie-privee/)
57. [Creative Commons Attribution on YouTube: How It Works - LegalClarity](https://legalclarity.org/what-is-creative-commons-attribution-on-youtube/)
58. [YouTube Creative Commons License Explained](https://www.hashtagnetwork.net/copyright/creative-commons)
59. [License types on YouTube - YouTube Help](https://support.google.com/youtube/answer/2797468?hl=en)
60. [Commons:YouTube files - Wikimedia Commons](https://commons.wikimedia.org/wiki/Commons:YouTube_files)
61. [How to download Twitch VODs before they expire · Clyper](https://www.clyper.io/guides/how-to-download-twitch-vods)
62. [How to Download Twitch VODs: A 2026 Creator's Guide](https://contesimal.ai/blog/download-twitch-vods/)
63. [chrome.tabCapture | Reference | Chrome for Developers](https://developer.chrome.com/docs/extensions/mv2/reference/tabCapture)
64. [GitHub - scoriiu/fenshot: Screenshot in, FEN out. Open-source chessboard recognition in your browser. · GitHub](https://github.com/scoriiu/fenshot)
65. [atalaydenknalbant/Kinetics-700 · Datasets at Hugging Face](https://huggingface.co/datasets/atalaydenknalbant/Kinetics-700)
66. [GitHub - Showmax/kinetics-downloader: Download DeepMind's Kinetics dataset.](https://github.com/Showmax/kinetics-downloader)
67. [AViD Dataset: Anonymized Videos from Diverse Countries AJ Piergiovanni](https://papers.neurips.cc/paper_files/paper/2020/file/c28e5b0c9841b5ef396f9f519bf6c217-Paper.pdf)
68. [V3C - a Research Video Collection](https://arxiv.org/pdf/1810.04401)
