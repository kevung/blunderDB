<!-- Lot du plan tasks/plan-amelioration-2026-09b/README.md. -->

# Lot H — Documentation, traductions, distribution, communauté, onboarding

État vérifié le 2026-09-02 : doc 9 langues complète au tag (96 chaînes en
retard sur `main`, lot A.7) ; 871 clés d'interface × 9 verrouillées par 5
tests ; 111 flags CLI documentés ; 25 ADR ; canaux réellement publiés : AUR,
`.deb`/`.rpm`, tarballs, binaires bruts, `.app`, `.exe`, GHCR (amd64+arm64),
PDF ×9. Non publiés : winget, Homebrew, Flathub. Le socle est bon ; les écarts
sont fins mais visibles (aucune image dans le manuel, aide intégrée en retard
de deux versions, page d'installation qui ne mentionne ni Flatpak ni
`.sha256`). Les urgences (notices tierces, nom du binaire, démo, `.po`) sont
au lot A.

H.1 à H.6 = **étape 1** ; H.7 à H.14 = **étape 2**.

---

## H.1 — Page d'installation et notes de release [S] — adoption (#243)

`telecharge_install.rst` : 0 mention de Flatpak, de GHCR, de `.sha256` ; notes
de release : 31 assets sans guide de choix, « 22 ADRs indexed » faux.
- [x] Tableau par OS/canal (le README l'a déjà) + Flatpak + GHCR + vérification
      `sha256sum -c` ; (E.7) signature minisign reste avec E.7.
- [x] Bloc « Installation » de 8 lignes injecté par la CI en tête de chaque
      release (`build.yml`, job release) ; chiffre des ADR calculé, pas écrit.
- [x] `.po` dans le même commit.

## H.2 — `CONTRIBUTING.md` et ce qui va avec [S] — adoption (#244)

Le README renvoie les contributeurs vers `CLAUDE.md`, écrit pour un agent ;
pas de `CODE_OF_CONDUCT.md` (Discord public) ; Discord visible seulement dans
`index.rst` FR ; `contactLinks` vides ; description GitHub « Backgammon
position database software » et 3 topics.
- [x] `CONTRIBUTING.md` (60 lignes : setup, `make check`, une PR = sa doc FR +
      ses `.po`, invariants renvoyés à `CLAUDE.md`, ADR) ; `CODE_OF_CONDUCT.md`
      (Contributor Covenant 2.1, contact `blunderdb@proton.me`).
- [x] Section *Community* du README (Discord, Discussions « Annonces »),
      lien Discord dans l'onglet À propos (9 langues) — `contactLinks` n'existe
      pas dans le code, rien à remplir.
- [ ] Description et topics GitHub (`gnubg`, `extreme-gammon`, `wails`,
      `svelte`, `go`, `spaced-repetition`, `backgammon-analysis`) — à lancer
      depuis une session où `gh` atteint le trousseau :
      `gh repo edit kevung/blunderDB --description "Backgammon blunder analysis: import your matches, search positions by structure and mistake, measure your play, study with spaced repetition — with an embedded evaluator" --add-topic backgammon,blunder,database,gnubg,extreme-gammon,wails,svelte,go,spaced-repetition,backgammon-analysis`

## H.3 — Homebrew tap, Flatpak, metainfo [S puis M] — adoption (#245)

Tap `kevung/homebrew-tap` inexistant (procédure écrite, 30 min) ; manifeste
Flatpak commité non constructible (`0.27.1`, sha `000…`) ; `metainfo.xml`
périmé (« EPC calculator »), 0 `<screenshots>`, 0 `<branding>`, 12 `<release>`
sans description ; `packaging/flatpak/` sans README ; winget rendu jamais
soumis.
- [ ] Créer le tap, y pousser le cask 0.35.x, ajouter le job qui met à jour le
      tap sur tag (comme `aur.yml`).
- [ ] `metainfo.xml` : description à jour, `<screenshots>` (H.5),
      `<branding>`, `<release>` avec description générée depuis les notes.
- [ ] `packaging/flatpak/README.md` ; manifeste commité = gabarit **et**
      constructible (URL de la dernière release, sha renseigné par
      `release.sh`).
- [ ] Soumission winget (PR `microsoft/winget-pkgs`) et Flathub (build
      from-source offline : prompt P16) — deux tickets humains, suivis au
      BACKLOG.

## H.4 — Tutoriels de bout en bout et FAQ [M] — adoption (#246)

`guide_utilisateur.rst` est un catalogue de 23 gestes isolés ; aucun parcours ;
FAQ (15 questions) : 0 gammonNet/Anki/filigrane/Docker, `faq.rst:181` sans
l'évaluateur, #101 fermée sans changement ; « calculateur EPC »
(`guide_utilisateur.rst:293-296`, `faq.rst:120`) ; vidéos YouTube de janvier
2025 en tête d'`index.rst:191,194`.
- [x] Quatre tutoriels : *mon premier import* (du `.xg` à la première
      recherche d'erreur), *étudier un match*, *une session Anki*, *déployer
      `serve` derrière un proxy* (G.1).
- [x] Page « comment progresser avec blunderDB » (routine hebdomadaire :
      importer → filtrer les décisions coûteuses (`bl`) → commenter → paquet
      Anki → PR par tournoi) — la page qui vaut dix fonctionnalités.
- [x] FAQ : +6 questions (ai-je besoin d'XG ? que vaut l'évaluateur ? où sont
      mes données ? partager une base ? mode serveur ? PR vs Snowie) ;
      corriger les coquilles (`faq.rst` : personalisée, aggrège, philisophie,
      fonctionne-t'il, architecture logicielle sans gammonNet ; `index.rst:7`
      aggréger).
- [x] Retirer les vidéos jusqu'à #102 ; script de démo en 3 min
      (importer → blunders → paquet Anki) sur la base `demo` régénérée (A.8).
- [x] Toctree : guide avant manuel ; intro d'`index.rst` cite les 12 sections
      (les 4 annexes techniques comptées comme 3 : sécurité Windows/macOS
      regroupée).

## H.5 — Captures d'écran générées [M] — découvrabilité (#247)

0 `figure::` dans `manuel.rst` (1 436 l.) et `guide_utilisateur.rst` ; 11 PNG
de janvier 2025 (SmartScreen) ; `img/smartscreen_fr.png` orpheline. Le harnais
existe (`SCREENSHOT=1 npx playwright test screenshot`, commit `e46b896d`).
- [ ] 12-15 captures panneau par panneau sur mock Wails avec la base démo,
      régénérées par une cible `make screenshots` et vérifiées à la release
      (skill `release-blunderdb`).
- [ ] Injectées dans `manuel.rst`, `guide_utilisateur.rst`, `metainfo.xml`,
      README ; supprimer l'orpheline.

## H.6 — Terminologie périmée dans l'interface et l'aide [S] — qualité (#248)

`config.bearoffIntro` (« panneau EPC ») ×9 ; `config.gammonnetIntro`
(« (ADR-0011) ») ×9 ; `help/fr.js` « Bearoff » ×7 vs « Eval » ×1 ; filigrane :
0 occurrence dans les 9 fichiers d'aide (toute la diffusion contrôlée de
0.31.0 absente) ; `tasks/BACKLOG.md:118` le notait.
- [x] Les 18 chaînes ; remise à niveau des 4 onglets d'aide × 9 (Eval,
      filigrane, `.dbx`, identité, `analyze`, Stats Joueurs, révision masquée) —
      dernière fois à la main avant H.7.

---

## H.7 — Une seule source pour l'aide intégrée [L] — dette structurelle (#249)

`frontend/src/i18n/help/*.js` : 11 620 lignes de HTML à la main × 9,
dupliquant `manuel.rst`/`raccourcis.rst`/`cmd_mode.rst` ; aucun mécanisme de
synchronisation ; effet mesuré en H.6. Ces fichiers sont aussi injectés via
`{@html}` et exclus du lint (D.15).
- [ ] Décision (ADR) entre : (a) `help/*.js` = artefact de build produit par
      Sphinx (`-b html` par section, 9 langues, CSS réduite) à
      `npm run build`, versionné ou non ; (b) aide réduite à un sommaire +
      liens profonds vers le site, 11 000 lignes supprimées, mode hors ligne
      perdu. Recommandation : (a), avec un test qui compare les titres de
      sections aux `.rst`.
- [ ] `helpVocabulary.sync.test.js` étendu aux 4 onglets × 9.

## H.8 — Doc développeur [S/M] — onboarding contributeur (#250)

`CLAUDE.md` : `cmd/` sans `calibrace`, « `tasks/FOLLOWUPS.md` liste les
suivis » faux, `engine/gammonnet` absent du tour (C.12) ; `docs/adr/README.md`
24/25 ; 0 diagramme ; 5 paquets sans doc comment ; `CLI_USAGE.md:1191-1195`
renvoie à `doc/archive` périmé et `:5-13` donne `wails build` sans
`-tags webkit2_41` ; `doc/build.py:112-118` nomme les PDF par SHA, `import os` ×2 ;
8 références ADR dans la doc utilisateur non publiées.
- [ ] Corriger `CLAUDE.md` (et y mentionner `Dockerfile.hostile`, `nightly`).
- [ ] `ARCHITECTURE.md` : 3 diagrammes mermaid (dispatch des modes, couches
      `database/` ↔ `storage/` ↔ backends, chemin d'un import `.xg` →
      parser → ingest → Zobrist → GUI/CLI/serveur).
- [ ] Publier `docs/adr/` sur le site (`/adr/`, anglais, non traduit) et lier ;
      sinon reformuler les 8 renvois.
- [ ] `CLI_USAGE.md` : squelette généré depuis les `flag.FlagSet`
      (`go run ./cmd/cli-doc-gen`), exemples à la main ; corriger les deux
      points ; `build.py` corrigé.
- [ ] Doc comments des 5 paquets.

## H.9 — Glossaire et historique lisibles [S/M] — qualité (#251)

Pas de glossaire utilisateur (CONTEXT.md est développeur) ; historique en
`csv-table` (ligne 0.35.0 = 2 100 caractères, un msgid par version : chaque
correction invalide 8 traductions).
- [ ] `glossaire.rst` (25-30 entrées dérivées de CONTEXT.md : position,
      référentiel, régime, verdict, score away et sentinelles, PR, Snowie,
      équité normalisée, ligne avant le jet, filigrane…).
- [ ] Historique : bloc « Dernière version » (6 puces) en tête ; tableau en
      annexe converti en liste de définitions (un msgid par puce) — à faire en
      une fois, avec régénération des `.po` (les anciens msgstr sont
      réutilisables par `msgmerge` seulement partiellement ; accepter le coût).

## H.10 — Site : négociation de langue et page d'accueil [S/M] — adoption (#252)

`gh-pages/index.html` = `meta refresh` vers `en/` ; racine = sommaire Sphinx ;
pas de page produit.
- [ ] Redirection JS sur `navigator.language` avec repli `en/`.
- [ ] Page d'accueil statique (capture, trois phrases, boutons par OS, lien
      doc/Discord/Releases), générée depuis un gabarit dans `doc/`.

## H.11 — Roadmap publique et Discussions [S] — communauté (#253)

`tasks/BACKLOG.md` fait office de roadmap (technique, FR, non lié) ; 0
milestone, 0 Project ; Discussions vides.
- [ ] Page `roadmap.rst` dérivée des lots I/J de ce plan (produit seulement),
      9 langues ; milestones GitHub par étape ; Discussions : catégorie
      « Annonces » alimentée par la CI à chaque release (ou désactiver).

## H.12 — Onboarding dans l'application [M] — adoption (#254)

Visites guidées : 4 zones sur 9 (0 pour Eval, Anki, Stats, Collections,
diffusion) ; pas d'écran de bienvenue (plateau vide au premier lancement,
`tour`/`demo` indécouvrables).
- [ ] Trois visites (eval, anki, stats) via le mécanisme générique de
      `tours.js`.
- [ ] Panneau de bienvenue quand aucune base récente : *Visite guidée*,
      *Base d'exemple*, *Importer mes matchs*, *Ouvrir une base* (lié à I.28).

## H.13 — Traductions : outillage [S] — qualité (#255)

`msgfmt` sur le répertoire entier échoue (en-têtes dupliqués
`annexe_db_scheme`/`annexe_filtres`) ; drapeau `python-format` faux positif
(BACKLOG).
- [ ] Documenter la boucle par fichier dans `doc/README.txt` ; `no-python-format`
      à l'extraction ou reformulation des 2-4 entrées par langue.

## H.14 — Binaire Linux arm64 [M] — adoption (#256)

31 assets, tous x86_64 sauf l'image Docker et le `.app` universel.
- [ ] Job `build` sur `ubuntu-24.04-arm` (webkit2gtk-4.1), `.deb`/`.rpm`/tarball
      arm64 ; AUR `blunderdb-bin` avec `aarch64`. Utile aussi comme filet
      NEON (#151).

---

## Résumé du lot

| Fiche | Effort | Étape |
|---|---|---|
| H.1, H.2, H.6 | S | 1 |
| H.3 | S puis M | 1 |
| H.4, H.5 | M | 1 |
| H.9, H.10, H.11, H.13 | S/M | 2 |
| H.8, H.12, H.14 | M | 2 |
| H.7 | L | 2-3 |

Règle transverse : tout `.rst` modifié embarque ses 8 `.po` dans le même
commit ; les captures et le bloc « dernière version » sont régénérés par la
skill `release-blunderdb`, qu'il faut mettre à jour en conséquence.
