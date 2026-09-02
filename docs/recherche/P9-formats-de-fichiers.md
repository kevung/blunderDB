# Formats de fichiers de match et de position du backgammon en 2026 — recensement et priorisation pour blunderDB

## TL;DR
- **Le gros du travail est déjà fait** : blunderDB 0.35.0 importe XG (.xg/.xgp), GnuBG (.sgf), BGBlitz (.bgf/txt), Jellyfish (.mat/.txt) et les positions collées (XGID, GNU ID, textes d'analyse). Les vrais manques prioritaires sont (1) le collage d'**OGID** (écosystème HedgeHog/OpenGammon, en forte croissance), (2) le **replay bgammon.org** (natif Go, spec publique, trivial à parser) et (3) les **formats texte hérités que GnuBG sait déjà lire** (.sgg GridGammon, oldmoves FIBS, .tmg, .gam, Snowie .txt, .bgf).
- **Point critique** : « OGXM / OGXM-JSON » présenté comme un conteneur binaire TLV signé Ed25519 **n'a aucune existence publique vérifiable** — aucune spec, aucun dépôt, aucune trace web. À ce jour c'est un format fantôme : ne pas écrire de parseur sur une spec inventée. En revanche **OGID** (identifiant de position base-26 à champs séparés par « : ») existe bien et est ré-implémentable par rétro-ingénierie depuis AnkiGammon (MIT).
- **Beaucoup de plateformes n'ont pas besoin d'un parseur dédié** : Backgammon Galaxy, Backgammon Studio/Heroes et GammonSpace produisent des fichiers **importables dans XG** (voire des .xg / .mat) — donc déjà couverts indirectement. Concentrer l'effort sur ce qui apporte de l'analyse nouvelle ou touche des communautés Go/open-source.

## Key Findings

### 1. État des lieux blunderDB (référence)
blunderDB est écrit en Go (backend) + Svelte 5 (frontend Wails), dépôt `github.com/kevung/blunderDB`, version 0.35.0 (2 septembre 2026). Import de matchs depuis XG (XG/XGP), GnuBG (SGF), Jellyfish (MAT/TXT), BGBlitz (BGF/TXT) depuis la v0.10.0 ; import de positions XGP avec analyse (v0.12.0) ; lecture des commentaires et **flags de révision XG** (v0.30.0) ; lecture de la **chance (luck) par coup** depuis XG et gnuBG (v0.33.0) ; export .mat (v0.29.0) ; évaluateur embarqué gammonNet en Go (v0.34.0). L'auteur (Kévin Unger, « postmanpat » sur Heroes) est lui-même joueur sur l'écosystème Heroes/Studio — signe que la demande OGID/Heroes est proche de sa base d'utilisateurs.

### 2. Le format XG est la clé de voûte (déjà supporté, spec publique complète)
La spécification .XG/.XGP est **publique et figée** depuis 2013-2014 (dernière modification 31/12/2013), publiée par Xavier Dufaure de Citres (GameSite 2000 Ltd) sur `extremegammon.com/xgformat.aspx`, avec le fichier Pascal `XG_format.pas` et `ZLIBArchive.pas`. C'est un **RichGameFormat** (magic `RGMH` = $484D4752) contenant un thumbnail JPG, puis un flux **ZLIB** ; citation verbatim de la spec : « *4 files are generated temp.xg // full game fixed size (2560 bytes) using TSaveRec… temp.xgr // rollouts indexed in temp.xg fixed size (2184 bytes) using TRolloutContext temp.xgc // comments… using RTF format.* » Soit : temp.xg (jeu complet, enregistrements fixes `TSaveRec` de 2560 octets), temp.xgi (header d'accès rapide), temp.xgr (rollouts, `TRolloutContext` de 2184 octets), temp.xgc (commentaires RTF, séparés par CRLF). Little-endian, chaînes ANSI (compatibilité XG1) + Unicode UTF-16 (champs v24+).

Le format contient **tout** : coups, décisions de cube, analyses multi-ply, rollouts complets (intervalle de confiance, écart-type, durée), luck (`ErrLuck`, `InitEq`), flags de révision (`Flagged`, `FlaggedDouble`), commentaires RTF, horloges (`TTimeSetting` : tsNone/tsFischer/tsBronstein), métadonnées de tournoi (event/round/location/date), Crawford/Jacoby/Beaver/AutoDouble, Elo, niveaux d'analyse (table PLAYERLEVEL : 1-ply à 7-ply, Rollout, XGRoller/+/++).

La table **SITE ID** interne à XG est précieuse pour la stratégie d'import (elle indique de quel site un match a été transcrit) : 0 GammonSite, 1 FIBS, 2 TrueMoney Games, 3 GridGammon, 4 DailyGammon, 5 NetGammon, 6 VOG, 7 Gammon Empire/Play65, 8 Club Games, 9 PartyGammon, 10 XcitingGames, 11 BGRoom, 12 DiceArena, 13 Safe Harbor Games, 14 GameAccount, 15 XG Mobile.

Implémentations de référence : `xgdatatools` de Michael Petch (Python ; dépôt d'origine `vcs.capp-sysware.com/gitweb/…/xgdatatools.git` souvent hors-ligne ; miroir `github.com/oysteijo/xgdatatools`), et le convertisseur en JavaScript navigateur d'AnkiGammon (« *Convert eXtreme Gammon .xg match files to Jellyfish .mat text format* », `ankigammon.com/tools/position-converter`, avec auto-détection XGID/GNUID/OGID, tout côté client).

### 3. OGID existe ; OGXM/TLV/Ed25519 n'existe pas publiquement
- **OGID (OpenGammon Position ID)** : format d'**identifiant de position** (analogue à XGID/GNU ID). Citation verbatim d'AnkiGammon (`github.com/Deinonychus999/AnkiGammon`) : « *OGID (OpenGammon Position ID) - Base-26 position encoding with colon-separated fields · Example: `cccccggggg:ddddiiiiii:N0N:63:W:IW:4:3:7:1:15`* » et « *All formats fully support position encoding, cube state, dice, and match metadata.* » Détection : « base-26 pattern with colons ». Un codec OGID fonctionnel existe en Python (paquet `ankigammon`, licence MIT) et en JavaScript (outil `ankigammon.com/tools/position-converter`, tourne côté navigateur). **La spec formelle n'est pas publiée** par OpenGammon ; opengammon.com est une app JS sans docs statiques lisibles.
- **OGXM / OGXM-JSON** : **aucune preuve d'existence** après recherche ciblée. Requêtes sur « OGXM », « OGXM-JSON », OGXM combiné à backgammon/OpenGammon/HedgeHog/TLV/Ed25519 → zéro résultat pertinent. Aucune spec TLV, aucun magic byte, aucune signature Ed25519 documentée, aucun dépôt. **Recommandation : ne pas développer de parseur OGXM ; traiter comme inexistant tant qu'aucune source primaire n'est produite.** Ce que l'on peut vérifier de HedgeHog/OpenGammon : app web (hedgehog-bg.com) avec analyse match-par-match, base de matchs (limitée à 10 en gratuit), fonctions « transcribe » et « study ». Moteurs multiples sélectionnables : **Aureus** joue au moins au niveau champion du monde (PR 1.9 mesuré en BGBlitz 4-ply / PR 1.6 en XG2 Roller++, « meilleur que tout joueur humain »), tandis que **Fox 0.32** est bien plus faible (PR 5.9, niveau expert), selon la revue Gammonrants d'août 2026. Offres payantes : « *expensive paid tiers that set you back 60 Euros or even 120 Euros a year to get 'Every model we run, including the frontier networks no other plan can select'* ». **Aucun format d'export de match documenté publiquement.**

### 4. bgammon.org : la cible open-source la plus facile
Serveur écrit en Go par **Trevor Slocum** (Seattle, WA), dépôt `code.rocket9labs.com/tslocum/bgammon` (miroirs GitHub/Codeberg), client Boxcars (Go/Ebitengine). Annoncé le 23 novembre 2023 sur la liste bug-gnubg (« *A new backgammon server is now available at https://bgammon.org… The server and client are free and open source* »). Protocole documenté dans **PROTOCOL.md**, **format de replay documenté dans REPLAY.md**. Les replays de match sont téléchargeables (commande `/download`) et sauvegardés côté compte. Rating Glicko-2, cube, horloge 30 s + 30 s. **Contenu : coups uniquement, pas d'analyse de moteur.** Import trivial en Go (mêmes structures de données, spec publique, licences open-source). Communauté modeste : environ **100 parties/jour** (backgammon, acey-deucey, tabula) à la date de juillet 2024, d'après le billet « The first 10,000 games at bgammon.org » de Slocum.

### 5. Plateformes qui passent déjà par XG (couvertes indirectement)
- **Backgammon Galaxy** (backgammongalaxy.com, fondateur **Marc Brockmann Olsen**, né le 15 janvier 1986, Grandmaster G2 danois dans les fédérations BMAB/UBC, PDG de Backgammon Galaxy, présent dans le top-32 des « Giants of Backgammon » 2017 et 2019) : moteur d'analyse **XG2**, « blunder log » automatique, et export/téléchargement de **fichiers XG** (les paquets de matchs UBC sont distribués en .xg à ouvrir dans XG ou gnuBG). Nouveauté « Trackgammon » (transcription par vision par ordinateur). → déjà importable via le parseur XG existant.
- **Backgammon Studio / Heroes** (Michael Petch) : fournit un mécanisme d'intégration XG (zip `XG_BackgammonStudio_site.zip` à déposer dans le dossier XG) qui déclare Backgammon Studio comme « site » d'import dans XG ; les matchs joués sont récupérables et analysés par XG. Matchs 11 points, tournois organisés (Challonge). → couvert via XG.
- **GammonSpace** : d'après les guides d'import XG, permet de télécharger la transcription de partie dans un format importable par XG. → couvert via XG.

### 6. Formats texte hérités que GnuBG sait déjà lire (candidats « faciles »)
GnuBG importe : **.sgg** (GridGammon / GamesGrid Save Game, format Snowie), **oldmoves/.fibs** (FIBS — ne lit que le premier match du fichier), **.tmg** (TrueMoneyGames), **.gam** (GammonEmpire et PartyGammon/Play65), **Snowie .txt**, **.bgf** (BGRoom), **.pos** (Jellyfish Position), **.mat** (Jellyfish). Le format natif GridGammon **.cbg** est **non documenté** et non supporté même par gnuBG. Ces parseurs texte sont simples et le code source gnuBG (GPL) sert de référence directe. Contenu : coups seulement (pas d'analyse), sauf .mat/.sgf qui peuvent porter des commentaires.

### 7. Formats d'export texte d'analyse XG multilingues
Point important sous-estimé dans la demande : **XG ne se localise pas dans toutes les langues demandées.** D'après AnkiGammon (qui automatise l'UI de XG), XG expose ses commandes de menu en **anglais, allemand, français, espagnol, japonais, grec et russe** (XG 2.10 et 2.19). blunderDB gère déjà FR/EN/JA/DE ; les ajouts XG réellement pertinents sont donc **ES (espagnol), EL (grec) et RU (russe)**. En revanche **IT, FI, NL, PT ne sont pas des langues d'interface XG** — il n'existe probablement pas d'export texte XG natif dans ces langues (les joueurs italiens/finlandais/néerlandais/portugais utilisent XG en anglais). GnuBG, lui, se traduit en cs_CZ, da_DK, es_ES, fi_FI, fr_FR, de_DE, el_GR, is_IS, it_IT, ja_JP, ro_RO, ru_RU, tr_TR : les libellés multilingues à couvrir côté **texte d'analyse gnuBG** diffèrent de ceux de XG. **Je n'ai pas pu récupérer d'échantillon réel de copier-coller par langue** (libellés ES/IT/RU/EL/FI/NL/PT de « Player Winning Chances », « Best move », « Rolled », « Analyzed in Rollout », « Double/Take », « No double », « Blunder », etc.) ; à confirmer sur des fichiers réels ou dans les ressources de localisation.

## Details

### Tableau comparatif

| Format / plateforme | Spec (lieu) | Public / RE / fermé | Contenu (coups / analyse) | Binaire ou texte, encodage | Parseur Go existant ? | Volumétrie / demande |
|---|---|---|---|---|---|---|
| **XG (.xg/.xgp)** | extremegammon.com/xgformat.aspx, figée 2013 | Public | Coups + analyse XG multi-ply + rollouts + luck + flags + commentaires RTF + horloges + tournoi | Binaire, RichGameFormat + ZLIB, ANSI+UTF-16, little-endian | Non (réf. Python/Java Petch) — blunderDB a le sien | Très forte (standard de facto) |
| **GnuBG (.sgf)** | Manuel gnuBG (gnu.org) | Public | Coups + analyse gnuBG + commentaires | Texte SGF | Réf. C gnuBG | Forte |
| **BGBlitz (.bgf)** | Fermé (Frank Berger) | Fermé / RE | Coups + analyse BGBlitz | Binaire/texte | Non | Moyenne |
| **Jellyfish (.mat)** | De facto, texte | Public (RE) | Coups (+ commentaires) ; pas d'analyse riche | Texte | Plusieurs | Forte (pivot universel) |
| **OGID** (position) | Exemple seulement (AnkiGammon) | RE (spec non publiée) | Position + cube + dés + score/longueur ; PAS l'analyse | Texte base-26, champs « : » | Non (Python/JS MIT dispo) | Croissante (HedgeHog) |
| **OGXM / OGXM-JSON** | **Introuvable** | **Inexistant publiquement** | — | — | — | Non vérifiable |
| **bgammon.org (replay)** | PROTOCOL.md + REPLAY.md (dépôt tslocum) | Public open-source | Coups seulement, pas d'analyse | Texte, spec Go | **Oui (tslocum, natif Go)** | Faible (~100 parties/j en 2024) |
| **Backgammon Galaxy** | Exporte des .xg | (via XG) | Coups + analyse XG2 | → XG | via parseur XG | Très forte |
| **Backgammon Studio/Heroes** | Intégration XG (site) | (via XG) | Coups + analyse XG | → XG | via parseur XG | Forte (base proche de l'auteur) |
| **GammonSpace** | Transcription → XG | (via XG) | Coups (analyse via XG) | → XG | via parseur XG | Moyenne |
| **GridGammon (.sgg)** | Format Snowie, doc gnuBG | Public (RE) | Coups seulement | Texte | Réf. C gnuBG | Moyenne (déclin) |
| **GridGammon (.cbg)** | Non documenté | Fermé | Coups | Binaire | Non | Faible |
| **FIBS oldmoves (.fibs)** | doc gnuBG | Public (RE) | Coups (1er match seulement) | Texte | Réf. C gnuBG | Faible (héritage) |
| **TrueMoneyGames (.tmg)** | doc gnuBG | Public (RE) | Coups | Texte | Réf. C gnuBG | Faible |
| **GammonEmpire / Play65 / PartyGammon (.gam)** | doc gnuBG | Public (RE) | Coups | Texte | Réf. C gnuBG | Faible-moyenne |
| **BGRoom (.bgf)** | doc gnuBG | Public (RE) | Coups | Texte/binaire | Réf. C gnuBG | Faible |
| **Snowie (.txt)** | doc gnuBG / Snowie | Public (RE) | Coups (+ analyse Snowie limitée) | Texte | Réf. C gnuBG | Faible (héritage) |
| **Backgammon NJ** | Introuvable | Fermé | Non vérifié | Non vérifié | Non | Faible |

### Ordre de priorité argumenté (valeur × faisabilité)

**P1 — Formats texte hérités via portage de la logique gnuBG (.sgg, oldmoves/.fibs, .tmg, .gam, Snowie .txt, .bgf).** Valeur moyenne mais **faisabilité maximale** : specs stables, code de référence GPL de gnuBG, formats texte simples ; un seul module d'import texte factorisé couvre 6 formats d'un coup et complète la couverture « héritage ». Meilleur ratio effort/couverture.

**P2 — Collage OGID.** Valeur croissante : HedgeHog/OpenGammon est l'app gratuite la plus forte de 2026 et la base d'utilisateurs de blunderDB (Heroes) y est adjacente. Faisabilité bonne : codec ré-implémentable depuis AnkiGammon (MIT) ; c'est un simple identifiant de position, cohérent avec le pipeline « positions collées » déjà présent. **Pré-requis : cloner AnkiGammon et extraire la logique exacte du décodeur (alphabet base-26, sémantique des 11 champs, checksum éventuel) — non vérifiée à ce stade.**

**P3 — Replay bgammon.org.** Faisabilité maximale (Go natif, PROTOCOL.md + REPLAY.md publics, licences open-source, mêmes structures de données que blunderDB) mais valeur limitée (coups sans analyse, communauté d'environ 100 parties/jour). À faire si l'on veut un « quick win » vitrine dans l'écosystème Go/FOSS.

**P4 — Rien à faire de spécifique pour Galaxy / Studio-Heroes / GammonSpace** : elles exportent déjà des fichiers XG que blunderDB lit. Action utile : documenter le workflow « télécharger le .xg puis importer » dans la FAQ plutôt que coder un parseur.

**P0 négatif — NE PAS développer OGXM** tant qu'aucune spec primaire n'existe. Idem un éventuel *match-file* OGID : seul l'identifiant de position est réel.

### OGID — résumé technique pour un parseur (avec niveau de confiance)
- **Confiance élevée** : chaîne texte, ASCII, **11 champs séparés par « : »** ; les deux premiers champs = 10 lettres base-26 chacun encodant l'état du plateau ; exemple canonique `cccccggggg:ddddiiiiii:N0N:63:W:IW:4:3:7:1:15`. Détectable par motif « lettres base-26 + colonnes ». Supporte position + état du cube + dés + métadonnées de match (confirmé verbatim par AnkiGammon).
- **Confiance faible (inférence, à vérifier dans le code)** : champ 4 `63` = dés (6-3) ; champ 5 `W` = joueur au trait (White) ; champs finaux `4:3:7:1:15` = score/longueur de match/cube. L'alphabet exact (a–z→0–25 ?), le mapping point→lettre, la gestion barre/sortie, l'encodage du cube (`N0N`), les flags (`IW`) et l'existence d'un checksum **ne sont pas vérifiés**.
- **Chemin de vérification** : `git clone https://github.com/Deinonychus999/AnkiGammon`, grep `ogid`/`base26`/`OGID` dans le dossier `ankigammon/` ; ou lire le JS de `ankigammon.com/tools/position-converter`. Licence MIT → réutilisable.

### OGXM — ce qui est demandé vs. ce qui existe
La demande décrit OGXM comme un conteneur binaire TLV (type-length-value), avec blocs d'analyse, signatures Ed25519 couvrant les données, et un codec OGID associé. **Aucun de ces éléments n'est trouvable publiquement.** Il est possible que « OGXM » soit un nom de travail interne, une confusion, ou un format non encore publié. Conclusion : documenter l'absence, surveiller opengammon.com / hedgehog-bg.com, et ne rien coder à l'aveugle. Il serait vain de spécifier ici une structure TLV et un schéma Ed25519 fictifs : aucune donnée primaire ne les soutient.

## Recommendations
1. **Sprint 1 (P1) — un module d'import texte « legacy » factorisé** couvrant .sgg, oldmoves/.fibs, .tmg, .gam, Snowie .txt, .bgf, en portant la logique du code gnuBG (GPL). Seuil de bascule : si <2 % des utilisateurs réclament ces formats après 3 mois d'observation des tickets GitHub/Discord, geler l'effort restant.
2. **Sprint 2 (P2) — collage OGID.** D'abord extraire et tester le décodeur d'AnkiGammon sur des OGID réels copiés depuis HedgeHog ; publier des tests de round-trip OGID↔XGID↔GNU ID. Faire passer en priorité haute si HedgeHog ajoute un **export de matchs** (surveiller les notes de version hedgehog-bg.com).
3. **Sprint 3 (P3) — replay bgammon.org** en Go natif (lecture REPLAY.md). Quick win technique, réutilise les paquets Go de tslocum.
4. **Documentation, pas code** : ajouter à la FAQ blunderDB le workflow d'export .xg depuis Backgammon Galaxy, Studio/Heroes et GammonSpace (ces plateformes sont déjà couvertes via le parseur XG).
5. **Multilingue XG** : n'ajouter que **ES, EL, RU** aux marqueurs de texte d'analyse XG (FR/EN/JA/DE déjà faits) ; ne pas chercher IT/FI/NL/PT côté XG (langues non supportées par l'UI XG). Traiter IT/FI/RU/EL/ES/NL/PT séparément pour le **texte d'analyse gnuBG** (langues effectivement traduites par gnuBG).
6. **OGXM** : aucune action de développement. Rouvrir le sujet uniquement sur publication d'une spec primaire.

**Benchmarks qui changent la priorisation** : (a) publication d'une spec OGXM ou d'un export de matchs HedgeHog → OGID/OGXM remonte en P1 ; (b) migration massive des joueurs vers Backgammon Galaxy 2.0 avec export non-XG → réévaluer ; (c) tickets utilisateurs répétés sur un format précis → le remonter indépendamment du classement ci-dessus.

## Caveats
- **OGXM/OGXM-JSON, Ed25519, TLV** : non trouvés ; conclusion « inexistant publiquement » basée sur l'absence totale de résultats — un accès non-public (Discord privé, spec interne OpenGammon) pourrait exister mais n'est pas vérifiable.
- **Sémantique détaillée d'OGID** : non vérifiée dans le code source ; l'inférence champ-par-champ ne doit pas être codée telle quelle sans lire le décodeur AnkiGammon.
- **Backgammon NJ** (app mobile) : format d'export non identifié dans les sources publiques ; à investiguer directement sur l'app. De même **Backgammon Blitz, Play65, iBackgammon, Backgammon+ (Tencent), Lightning/VIP Backgammon, Motif, TD-Gammon archives** : pas de format d'export/analyse public identifié — la plupart n'exportent rien d'analysable, d'où leur absence du tableau de priorités.
- **Échantillons de texte d'analyse XG multilingues** (libellés ES/IT/RU/EL/FI/NL/PT) : non récupérés dans cette itération ; l'affirmation « XG ne se localise qu'en EN/DE/FR/ES/JA/EL/RU » provient d'AnkiGammon et devrait être confirmée sur des installations réelles / ressources de localisation XG.
- Les volumétries d'usage sont qualitatives (peu de chiffres publics fiables) ; l'app Android « OpenGammon » affiche des téléchargements très faibles mais ne reflète pas l'usage web de HedgeHog, qui est l'app réellement en croissance en 2026.
- **eXtreme Gammon XGP/XGR, .mat Jellyfish, Snowie .sno** : XGR est le sous-fichier interne de rollout du conteneur .xg (pas un format d'échange autonome) ; blunderDB lit déjà XGP et .mat ; « .sno » n'est pas confirmé comme extension d'échange courante (Snowie exporte en texte .txt, déjà géré via gnuBG).