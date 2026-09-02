# Distribuer une application Wails v2 NON SIGNÉE : plan de soumission multi-canal et configurations concrètes (septembre 2026)

## TL;DR
- **Faisable partout SAUF le tap officiel Homebrew cask** : Flathub (build-from-source hors-ligne), winget (un `.exe` non signé est accepté mais génère des frictions SmartScreen/Defender côté utilisateur), AUR, `.deb`/`.rpm` acceptent tous une app non signée ; en revanche, comme l'annonce l'issue Homebrew/brew #20755, « we are ending support for all casks that fail Gatekeeper checks on September 1st, 2026 » — sur les 7624 casks, 387 échouaient déjà à ce contrôle (Homebrew Discussion #6482). Un `.app` non notarisé ne peut donc plus être hébergé dans `homebrew/cask` : il faut un **tap personnel** où l'utilisateur exécute lui-même le `xattr -rd com.apple.quarantine`.
- **Deux verrous techniques Flathub à régler** : (1) build 100 % hors-ligne → vendoring Go via `flatpak-go-mod` et npm via `flatpak-node-generator` ; (2) WebKitGTK → viser `org.gnome.Platform` 46+ (qui embarque `webkit2gtk-4.1`) et compiler Wails avec `-tags webkit2_41`. Le metainfo AppStream avec `<screenshots>` et `<branding>` est désormais obligatoire pour passer la validation `appstreamcli`.
- **Vérification de version** : appeler l'API GitHub Releases non authentifiée (60 req/h/IP ; l'ETag économise la bande passante mais **PAS le quota sans token**), cacher 24 h, opt-in, et surtout **détecter le canal d'installation** (Flatpak/Snap/deb/cask) via un canal codé en dur au build (`-ldflags -X`) pour ne jamais proposer une mise à jour manuelle à un utilisateur de paquet géré.

## Key Findings

1. **Flathub build hors-ligne.** La documentation Flathub est explicite : « There is no network access during the build process so all dependencies used by the application must be supplied by specifying them in the manifest as sources with publicly accessible URLs ». Pour Go, on génère les sources vendorées avec `flatpak-go-mod` (`go run github.com/dennwc/flatpak-go-mod@latest`) qui produit `go.mod.yml` + `modules.txt` ; pour le frontend JS, `flatpak-node-generator` produit `generated-sources.json`.

2. **WebKitGTK résolu par le runtime.** webkit2gtk-4.1 est embarqué dans `org.gnome.Platform` depuis GNOME 42 (annonce Michael Catanzaro, GNOME Discourse, juillet 2021 : « webkit2gtk-4.1 has been added to the GNOME SDK … built against libsoup 3 »). En visant le runtime 46+, on n'a PAS besoin de compiler webkit soi-même ni de BaseApp. Wails doit être compilé avec `-tags webkit2_41` sinon il cherche 4.0 et échoue (issue wailsapp/wails #3345 ; doc Wails : « If you are using a Linux distribution that does not have webkit2gtk-4.0 (such as Ubuntu 24.04), you will need to add -tags webkit2_41 »).

3. **Exemples réels.** `app.riftshare.RiftShare` est une vraie app Wails v2 sur Flathub — mais **archivée EOL le 1er mai 2026**, sur le runtime GNOME 42 désormais obsolète (utile comme structure, pas comme cible). `org.jousse.vincent.Pomodorolm` (Tauri v2, aussi WebKitGTK, runtime GNOME 46, `shared-modules/libappindicator` pour le tray) est le modèle actif et à jour à suivre.

4. **winget accepte le `.exe` non signé** ; les installeurs doivent être MSIX/MSI/APPX/`.exe` (les zip/portable ont un support limité). L'automatisation se fait via `vedantmgoyal9/winget-releaser` (PAT classique `public_repo` — les PAT fine-grained ne marchent pas —, fork de `microsoft/winget-pkgs`). La validation passe par des pipelines Azure + VirusTotal ; un binaire non signé déclenche des avertissements SmartScreen côté utilisateur mais n'est pas rejeté d'office.

5. **Homebrew cask : rupture de septembre 2026.** Issue Homebrew/brew #20755 : « it's time to deprecate the --no-quarantine flag from brew. It intentionally bypasses macOS security mechanisms, which we already actively discourage » et fin de support des casks échouant à Gatekeeper au 1er sept. 2026. La réponse officielle (Homebrew Discussion #6334) précise : « Starting September 1st, 2026, Homebrew will disable casks that fail Gatekeeper security checks only in the main Homebrew/cask repo … you'll need to host them yourself by creating your own tap ».

## Details

### 1. Flathub (exigences 2026)

**Build hors-ligne et vendoring.** Flathub construit sur son infra, sans réseau, sur x86_64 ET aarch64 par défaut. Il faut donc fournir toutes les dépendances comme `sources`.

- **Go** : `flatpak-go-mod` génère `go.mod.yml` (directives `sources` pointant vers `proxy.golang.org`) et `modules.txt` (à copier dans le dépôt, monté dans `vendor/`). Build avec `go build -mod=vendor`. Alternative bas niveau : `go-modules`/`flatpak-go-vendor-generator.py` du dépôt `flatpak/flatpak-builder-tools`.
- **npm/pnpm/yarn** : `flatpak-node-generator` (dossier `node/` de flatpak-builder-tools) lit `package-lock.json`/`yarn.lock` et produit `generated-sources.json` permettant `npm install --offline`. `GOFLAGS=-mod=vendor`, `GOPATH=$PWD`, et le cache npm doivent être configurés dans `build-options.env`.

**Metainfo/AppStream.** Le fichier doit être placé dans `/app/share/metainfo/<app-id>.metainfo.xml`, validé par `appstreamcli validate` (via `flatpak run org.flatpak.Builder`). Erreurs ET warnings sont fatals. Exigences 2026 : `<screenshots>` obligatoires (URL directe vers une image, pas une branche git mais un tag/commit — hébergement sur `dl.flathub.org/media/` après composition), `<branding>` (couleurs), `<content_rating>` (OARS), `<developer id="...">` (nouvelle forme, `developer_name` déprécié), `<releases>` obligatoire, ID reverse-DNS exact correspondant au manifeste. Le composant doit être de type `desktop-application`.

**Runtime.** Choisir `org.gnome.Platform` 46+ (embarque webkit2gtk-4.1 + GTK3). `org.kde.Platform` pour Qt, `org.freedesktop.Platform` pour le minimal. SDK extension `org.freedesktop.Sdk.Extension.golang` + `node20`.

**Permissions/finish-args.** Pour une app WebKitGTK : `--socket=wayland`, `--socket=fallback-x11`, `--device=dri`, `--share=ipc`, `--share=network`, et le minimum de `--filesystem`. Flathub affiche un badge « Potentially Unsafe » dès qu'on utilise `--filesystem=home`/`host`, l'accès matériel, ou le bus session complet — critique récurrente sur Flathub Discourse (« Every single app … is labeled as Potentially Unsafe »). Les tests automatiques bloquent les permissions dangereuses (bus système, noms de bus non sûrs) et les branches git sans commit ; le blog officiel « Flathub Safety » confirme : « our automated tests block unsafe or outright wrong permissions … disallowing pointing at bare git branches without a specific commit ».

**Procédure de soumission.** Fork `flathub/flathub`, créer une branche `new-pr`, y placer le manifeste nommé `<app-id>.yaml` à la racine. Revue humaine des mainteneurs + tests automatiques + build bot. `flathub.json` à la racine pour restreindre les architectures (`only-arches`/`skip-arches`). Vérification (« verified app », badge bleu) via flathub.org avec un token prouvant la propriété de l'app-id (ex. via GitHub : « The developer with that username or an owner of that organization needs to log in with GitHub and confirm »).

### 2. winget et Homebrew

**winget.** Manifeste multi-fichiers v1.x : `<id>.installer.yaml`, `<id>.locale.<lang>.yaml`, `<id>.yaml` (version), dans `manifests/<lettre>/<Publisher>/<Package>/<Version>/`. `wingetcreate new|update|submit` ou `winget-releaser` en CI. `InstallerType: nsis` (Wails génère un installeur NSIS), `InstallerSwitches` pour le mode silencieux, `Scope`, `Architecture`. Un `.exe` non signé passe la validation mais l'utilisateur voit un avertissement SmartScreen ; VirusTotal peut lever des faux positifs Defender. Tester localement avec `winget settings --enable LocalManifestFiles` puis le script Sandbox.

**Automatisation winget.** `vedantmgoyal9/winget-releaser` (successeur de `vedantmgoyal2009`, basé sur Komac), déclenché sur `release: [released]`, nécessite un PAT classique `public_repo` (« New fine-grained PATs aren't supported by the action »), et un fork de `winget-pkgs` sous le même compte. Au moins une version doit déjà exister dans le repo pour servir de base. Piège : ne pas utiliser le `GITHUB_TOKEN` par défaut (il ne déclenche pas d'autres workflows).

**Homebrew cask/tap.** Les critères de notoriété de `homebrew/cask` sont désormais à deux niveaux (Homebrew/brew PR #21612, `docs/Acceptable-Casks`) : soumission par un **tiers** = « 30 forks, 30 watchers, 75 stars » ; **auto-soumission par le développeur** = « 90 forks, 90 watchers, 225 stars ». Mais depuis Homebrew 5.0.0 et surtout au **1er sept. 2026**, les casks échouant à Gatekeeper ne sont plus hébergés dans le tap officiel et `--no-quarantine` est déprécié. Pour une app non notarisée : **tap personnel** `user/homebrew-tap`, et l'utilisateur exécute `xattr -rd com.apple.quarantine /Applications/MonApp.app`. Un mainteneur Homebrew confirme : « Yes, post-processing is required, as it would be if you download and extract the files using other methods ». Options complémentaires : `auto_updates true`, `depends_on macos:`.

### 3. Vérification de version

**API GitHub.** `GET /repos/:owner/:repo/releases/latest`, limite non authentifiée **60 req/h par IP** (docs GitHub). En-têtes `X-RateLimit-Limit/Remaining/Reset`. User-Agent obligatoire. ETag/`If-None-Match` : renvoie 304 sans corps — MAIS **sans token, un 304 décompte quand même du quota** (mesuré : DEV Community, « an unauthenticated 304 still costs you a request … x-ratelimit-remaining dropped by exactly 3 »). Alternative recommandée pour éviter les limites : héberger un JSON statique de version sur GitHub Pages ou en asset de release. (Note : depuis mai 2025 GitHub a resserré les limites sur les requêtes non authentifiées.)

**Bonnes pratiques.** Opt-in explicite (RGPD/consentement pour toute télémétrie), cache 24 h, backoff exponentiel sur 403, mode hors-ligne, ne jamais bloquer le démarrage.

**Détection du canal (essentiel).** Détecter Flatpak (`/.flatpak-info`, `$FLATPAK_ID`), Snap (`$SNAP`), AppImage (`$APPIMAGE`), Homebrew (chemin `/opt/homebrew/Caskroom`), winget/MSIX. La méthode la plus fiable est d'injecter le canal au build via `-ldflags "-X main.channel=flathub"`, plus robuste que la détection runtime, et de désactiver l'auto-update pour tout canal géré.

**Frameworks de mise à jour.** Sparkle (macOS) exige une signature EdDSA (ed25519) de l'appcast — indépendante de la notarisation Apple ; « Serving updates using DSA only without EdDSA is no longer supported ». La signature EdDSA vérifie que « the update file I just downloaded was really made by this app's developer », couche distincte du certificat de plateforme. WinSparkle pour Windows. Côté Go : `minio/selfupdate`, `rhysd/go-github-selfupdate`, `creativeprojects/go-selfupdate`. **Wails v2 n'a PAS d'updater officiel intégré.** RAISONNABLE : télécharger + vérifier une signature EdDSA/minisign/cosign et des checksums SHA-256 publiés, remplacement atomique du binaire. PAS RAISONNABLE : auto-remplacement dans `/usr` sans droits, mise à jour silencieuse non vérifiée, contournement de Gatekeeper.

**Signature d'artefacts sans certificat payant.** minisign, cosign/sigstore keyless (OIDC, pas de clé longue durée : « prove you are this identity »), checksums signés GPG, et `actions/attest-build-provenance` (« binds some subject … to a SLSA build provenance predicate using the in-toto format » ; signé via Sigstore, vérifiable avec `gh attestation verify`).

### 4. Associations de fichiers

**Linux.** `.desktop` avec `MimeType=`, XML `shared-mime-info` dans `/usr/share/mime/packages/`, `update-mime-database`, `update-desktop-database`, `xdg-mime default`. Depuis nfpm : livrer `.desktop`, XML mime, icônes (y compris `mimetypes/` du thème hicolor et Yaru), et scripts postinstall/postremove qui appellent `update-mime-database /usr/share/mime`, `update-desktop-database /usr/share/applications`, `update-icon-caches`. En Flatpak, les fichiers vont dans `/app/share/mime/packages` et sont exportés automatiquement. Le `%u` dans `Exec=` est important.

**macOS.** `Info.plist` : `CFBundleDocumentTypes` (avec `LSItemContentTypes`, `CFBundleTypeRole`, `LSHandlerRank`), `UTExportedTypeDeclarations`/`UTImportedTypeDeclarations` (UTI custom, ex. `com.example.monext`). Dans Wails v2 on déclare `fileAssociations` dans `wails.json` (`ext`, `name`, `iconName`, `role` → correspond à `CFBundleTypeRole`) ; Wails génère l'Info.plist et les icônes. L'ouverture appelle `Mac.OnFileOpen`. Sur une app non signée, `lsregister` fonctionne mais Gatekeeper bloque le premier lancement (l'utilisateur doit autoriser).

**Windows.** Clés de registre (`HKCR\.ext`, ProgID, `shell\open\command`, `DefaultIcon`) écrites par l'installeur NSIS généré par Wails (`build/windows/installer/project.nsi`) — « On Windows file association is supported only with NSIS installer ». Windows 10/11 impose le choix utilisateur du handler par défaut (impossible de le forcer par programme). Déclaration via `fileAssociations` dans `wails.json`.

**Réception dans l'app.** Linux/Windows : `os.Args[1:]`. macOS : Apple Event `kAEOpenDocuments` → `Mac.OnFileOpen`. `SingleInstanceLock` (avec `UniqueId` UUID + `OnSecondInstanceLaunch`, dont `SecondInstanceData.Args` porte le chemin du fichier) permet de transmettre le fichier à l'instance existante au lieu de lancer une seconde instance. `CFBundleURLTypes` pour les deep links (`OnUrlOpen`).

### 5. Binaires Linux arm64

Les runners `ubuntu-24.04-arm` et `ubuntu-22.04-arm` sont **GA depuis le 7 août 2025 pour les dépôts publics** (GitHub Changelog : « Starting today, Linux and Windows arm64 standard hosted runners in public repositories are generally available … These runners are only available in public repositories and will not work in private repositories »). Matériel : « Powered by the Cobalt 100-based processors, these 4 vCPU runners can deliver up to a 40% performance boost » (Neoverse N2, gratuits pour les dépôts publics dans les limites standard). Pour les dépôts **privés**, GitHub a par la suite (changelog du 29 janvier 2026, « arm64 standard runners are now available in private repositories ») rendu ces runners arm64 standard disponibles et éligibles à l'allocation de minutes gratuites du plan — à vérifier selon votre plan, car auparavant seuls les « larger runners » ARM (Team/Enterprise) étaient utilisables en privé. `libwebkit2gtk-4.1-dev` est disponible sur Ubuntu 24.04 arm64 (build natif recommandé, avec `-tags webkit2_41` ; Ubuntu 22.04 n'a que la 4.0). Alternatives : QEMU (`docker/setup-qemu-action`, lent), cross-compilation avec `zig cc`, self-hosted. Empaquetage : `.deb`/`.rpm` arm64 via nfpm, Flatpak aarch64 construit par le build bot Flathub par défaut, AUR via `arch=('x86_64' 'aarch64')`.

## Recommendations

**Étape 1 — Fondations (avant toute soumission, 1-2 jours).** Publier des releases GitHub avec, pour chaque artefact : checksums SHA-256, une signature minisign OU cosign keyless, et `actions/attest-build-provenance`. Injecter le canal via `-ldflags -X`. Mettre en place une matrice CI incluant `ubuntu-24.04-arm`.

**Étape 2 — Canaux à faible friction (semaine 1).** AUR (PKGBUILD, `arch=('x86_64' 'aarch64')`), `.deb`/`.rpm` via nfpm (avec associations de fichiers et scripts postinstall), et tap Homebrew personnel avec instruction `xattr` documentée dans `caveats`. Ces canaux acceptent l'absence de signature sans condition.

**Étape 3 — winget (semaine 2).** Soumettre le premier manifeste avec `wingetcreate` (fork + PR manuelle), puis automatiser avec `winget-releaser` (PAT classique). Documenter côté utilisateur la marche à suivre pour l'avertissement SmartScreen.

**Étape 4 — Flathub (semaines 3-4, le plus d'effort).** Générer un metainfo AppStream complet (screenshots hébergés sur tag/commit, branding, content_rating, developer id), vendorer Go+npm, viser runtime GNOME 46+, compiler `-tags webkit2_41`, ouvrir la PR `new-pr`. Prévoir plusieurs allers-retours avec les reviewers.

**Seuils qui changent la donne.**
- Obtenir un compte Apple Developer (99 $/an) et notariser → le tap officiel Homebrew cask redevient possible ET Sparkle (EdDSA) devient l'updater idéal. C'est le seul vrai déblocage macOS post-sept. 2026.
- Dépasser 75 stars / 30 forks / 30 watchers (ou 225/90/90 en auto-soumission) → soumission à `homebrew/cask` officiel envisageable, mais uniquement si notarisé.
- Passage en dépôt privé → vérifier la disponibilité/coût des runners ARM (changement de janvier 2026) avant de retirer QEMU.

## Caveats
- **Daté au 2 septembre 2026.** La règle Homebrew Gatekeeper (1er sept. 2026, issue #20755) est toute récente — vérifier son application effective sur des casks réels.
- **Changement runners ARM privés (29 janvier 2026)** rapporté par l'enrichissement mais non revérifié directement dans cette recherche : confirmer les modalités de facturation exactes dans la doc GitHub Actions avant de s'y fier.
- **Wails v2 vs v3** : la v3 est en beta ; les associations de fichiers macOS y ont été portées récemment (PR wailsapp/wails #3873, #4177). Ce rapport cible la v2.
- RiftShare (`app.riftshare.RiftShare`) est archivé/EOL (mai 2026, runtime GNOME 42) — utile comme structure de manifeste, mais préférer le modèle GNOME 46 de Pomodorolm (`org.jousse.vincent.Pomodorolm`).
- Le comportement exact de VirusTotal/Defender/SmartScreen sur un `.exe` Wails non signé est variable et non garanti.
- L'ETag n'économise pas le quota GitHub sans token : sur 60 req/h/IP, préférer un JSON statique pour la vérification de version.

---

### Annexe — Extraits de configuration copiables

**A. Manifeste Flatpak (YAML) pour une app Wails :**
```yaml
app-id: com.example.MonApp
runtime: org.gnome.Platform
runtime-version: '46'
sdk: org.gnome.Sdk
sdk-extensions:
  - org.freedesktop.Sdk.Extension.golang
  - org.freedesktop.Sdk.Extension.node20
command: MonApp
finish-args:
  - --share=ipc
  - --socket=wayland
  - --socket=fallback-x11
  - --device=dri
  - --share=network
  - --filesystem=xdg-documents
build-options:
  append-path: /usr/lib/sdk/golang/bin:/usr/lib/sdk/node20/bin
  env:
    GOFLAGS: '-mod=vendor'
    GOROOT: /usr/lib/sdk/golang
modules:
  - name: monapp
    buildsystem: simple
    build-commands:
      - . /usr/lib/sdk/node20/enable.sh; export GOPATH=$PWD/.go
      - go build -tags "desktop,production,webkit2_41" -ldflags "-w -s -X main.channel=flathub" -o MonApp
      - install -Dm755 MonApp /app/bin/MonApp
      - install -Dm644 build/appicon.png /app/share/icons/hicolor/512x512/apps/com.example.MonApp.png
      - install -Dm644 com.example.MonApp.desktop /app/share/applications/com.example.MonApp.desktop
      - install -Dm644 com.example.MonApp.metainfo.xml /app/share/metainfo/com.example.MonApp.metainfo.xml
    sources:
      - type: git
        url: https://github.com/example/monapp
        tag: v1.0.0
        commit: <sha40>
      - type: file
        path: modules.txt
        dest: vendor
      # ... entrées go.mod.yml générées par flatpak-go-mod (type: archive vers proxy.golang.org) ...
      - generated-sources.json   # généré par flatpak-node-generator
```
> Modèle de référence à jour : `org.jousse.vincent.Pomodorolm` (github.com/flathub/org.jousse.vincent.Pomodorolm), runtime GNOME 46, avec `shared-modules/libappindicator` si tray. Modèle de structure Wails (EOL) : `app.riftshare.RiftShare`.

**B. Metainfo AppStream XML :**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<component type="desktop-application">
  <id>com.example.MonApp</id>
  <metadata_license>CC0-1.0</metadata_license>
  <project_license>MIT</project_license>
  <name>MonApp</name>
  <summary>Un éditeur de fichiers .monext</summary>
  <developer id="com.example">
    <name>Example Corp</name>
  </developer>
  <description>
    <p>MonApp est une application de bureau construite avec Wails.</p>
    <p>Elle permet d'éditer et de visualiser des fichiers .monext.</p>
  </description>
  <launchable type="desktop-id">com.example.MonApp.desktop</launchable>
  <screenshots>
    <screenshot type="default">
      <image>https://raw.githubusercontent.com/example/monapp/v1.0.0/screenshots/main.png</image>
      <caption>Fenêtre principale</caption>
    </screenshot>
  </screenshots>
  <branding>
    <color type="primary" scheme_preference="light">#3584e4</color>
    <color type="primary" scheme_preference="dark">#1a5fb4</color>
  </branding>
  <content_rating type="oars-1.1"/>
  <releases>
    <release version="1.0.0" date="2026-09-01"/>
  </releases>
</component>
```

**C. `.desktop` + shared-mime-info :**
```ini
[Desktop Entry]
Type=Application
Name=MonApp
Exec=/usr/bin/monapp %u
Icon=com.example.MonApp
Terminal=false
Categories=Utility;
MimeType=application/x-mon-app;
```
```xml
<?xml version="1.0" encoding="UTF-8"?>
<mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">
  <mime-type type="application/x-mon-app">
    <comment>Fichier MonApp</comment>
    <glob pattern="*.monext"/>
  </mime-type>
</mime-info>
```
Scripts nfpm postinstall :
```sh
update-mime-database /usr/share/mime
update-desktop-database /usr/share/applications
update-icon-caches /usr/share/icons/*
```

**D. Manifeste winget (installer) :**
```yaml
PackageIdentifier: Example.MonApp
PackageVersion: 1.0.0
InstallerType: nsis
Installers:
  - Architecture: x64
    InstallerUrl: https://github.com/example/monapp/releases/download/v1.0.0/MonApp-amd64-installer.exe
    InstallerSha256: <SHA256>
    Scope: user
    InstallerSwitches:
      Silent: /S
      SilentWithProgress: /S
ManifestType: installer
ManifestVersion: 1.6.0
```

**E. Workflow winget-releaser :**
```yaml
name: Publish to WinGet
on:
  release:
    types: [released]
jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: vedantmgoyal9/winget-releaser@main
        with:
          identifier: Example.MonApp
          installers-regex: '\.exe$'
          token: ${{ secrets.WINGET_TOKEN }}   # PAT classique, scope public_repo
```

**F. Cask Homebrew (tap personnel — app non notarisée) :**
```ruby
cask "monapp" do
  version "1.0.0"
  sha256 "<SHA256>"
  url "https://github.com/example/monapp/releases/download/v#{version}/MonApp-universal.zip"
  name "MonApp"
  desc "Éditeur de fichiers .monext"
  homepage "https://example.com/"
  depends_on macos: ">= :big_sur"
  app "MonApp.app"
  caveats <<~EOS
    MonApp n'est pas signée/notarisée. Après installation, exécutez :
      xattr -rd com.apple.quarantine "/Applications/MonApp.app"
  EOS
end
```

**G. Info.plist macOS (CFBundleDocumentTypes + UTI) :**
```xml
<key>CFBundleDocumentTypes</key>
<array>
  <dict>
    <key>CFBundleTypeName</key><string>MonApp Document</string>
    <key>CFBundleTypeRole</key><string>Editor</string>
    <key>LSHandlerRank</key><string>Owner</string>
    <key>LSItemContentTypes</key>
    <array><string>com.example.monext</string></array>
  </dict>
</array>
<key>UTExportedTypeDeclarations</key>
<array>
  <dict>
    <key>UTTypeConformsTo</key><array><string>public.data</string></array>
    <key>UTTypeDescription</key><string>MonApp Document</string>
    <key>UTTypeIdentifier</key><string>com.example.monext</string>
    <key>UTTypeTagSpecification</key>
    <dict>
      <key>public.filename-extension</key><array><string>monext</string></array>
      <key>public.mime-type</key><string>application/x-mon-app</string>
    </dict>
  </dict>
</array>
```
Équivalent `wails.json` (Wails génère l'Info.plist) :
```json
{ "info": { "fileAssociations": [
  { "ext": "monext", "name": "MonApp Document",
    "description": "Fichier MonApp", "iconName": "monextIcon", "role": "Editor" }
]}}
```

**H. NSIS (association de fichiers) :**
```nsis
WriteRegStr HKCR ".monext" "" "Example.MonApp"
WriteRegStr HKCR "Example.MonApp" "" "MonApp Document"
WriteRegStr HKCR "Example.MonApp\DefaultIcon" "" "$INSTDIR\MonApp.exe,0"
WriteRegStr HKCR "Example.MonApp\shell\open\command" "" '"$INSTDIR\MonApp.exe" "%1"'
System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, i 0, i 0)'  ; SHCNE_ASSOCCHANGED
```

**I. Go — détection de canal + version via ETag :**
```go
package main

var channel = "source" // injecté au build : -ldflags "-X main.channel=flathub"

func detectChannel() string {
    if channel != "source" { return channel }
    if _, err := os.Stat("/.flatpak-info"); err == nil { return "flathub" }
    if os.Getenv("SNAP") != "" { return "snap" }
    if os.Getenv("APPIMAGE") != "" { return "appimage" }
    return "source"
}

// N'appeler que si le canal N'EST PAS un paquet géré.
func checkUpdate(etag string) (tag, newEtag string, err error) {
    req, _ := http.NewRequest("GET",
        "https://api.github.com/repos/example/monapp/releases/latest", nil)
    req.Header.Set("User-Agent", "MonApp-updater")   // obligatoire
    req.Header.Set("Accept", "application/vnd.github+json")
    if etag != "" { req.Header.Set("If-None-Match", etag) }
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return "", etag, err }
    defer resp.Body.Close()
    if resp.StatusCode == http.StatusNotModified { return "", etag, nil }
    var r struct{ TagName string `json:"tag_name"` }
    json.NewDecoder(resp.Body).Decode(&r)
    return r.TagName, resp.Header.Get("ETag"), nil
}
```

**J. Options Wails (OnFileOpen + SingleInstanceLock) :**
```go
wails.Run(&options.App{
    Title:     "MonApp",
    OnStartup: app.startup,
    SingleInstanceLock: &options.SingleInstanceLock{
        UniqueId:               "e3984e08-28dc-4e3d-b70a-45e961589cdc",
        OnSecondInstanceLaunch: app.onSecondInstanceLaunch, // reçoit SecondInstanceData.Args
    },
    Mac: &mac.Options{
        OnFileOpen: func(filePaths []string) { app.openFiles(filePaths) },
    },
})
// Linux/Windows : lire os.Args[1:] au démarrage pour le chemin du fichier.
```

**K. Matrice GitHub Actions avec arm64 :**
```yaml
jobs:
  build:
    strategy:
      matrix:
        include:
          - os: ubuntu-24.04
            arch: amd64
          - os: ubuntu-24.04-arm   # GA pour dépôts publics depuis le 07/08/2025
            arch: arm64
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - run: sudo apt-get update && sudo apt-get install -y libwebkit2gtk-4.1-dev libgtk-3-dev
      - run: wails build -tags webkit2_41 -platform linux/${{ matrix.arch }}
```