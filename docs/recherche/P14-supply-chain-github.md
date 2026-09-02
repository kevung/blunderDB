# Sécuriser la chaîne d'approvisionnement d'un projet open source publiant via GitHub Actions — État 2026

## TL;DR
- **Trois actions à fort levier et faible coût dominent** : (1) passer `default_workflow_permissions` en `read` + blocs `permissions:` minimaux par job ; (2) ajouter `actions/attest-build-provenance` (désormais un simple wrapper d'`actions/attest`) pour signer une provenance SLSA Build L2 vérifiable via `gh attestation verify` ; (3) épingler TOUTES les actions par SHA (y compris `actions/*`) et laisser Renovate/Dependabot maintenir les pins. Ces trois éléments coûtent moins d'une journée de travail et neutralisent la classe d'attaques tj-actions (CVE-2025-30066, CVSS 8.6) / TeamPCP (CVE-2026-33634, CVSS 9.4).
- **La signature (cosign keyless / attestation) ne remplace PAS un certificat de signature de code.** SmartScreen (Windows) et Gatekeeper (macOS) ignorent totalement une attestation SLSA ou une signature Sigstore. En 2026, Homebrew a déjà retiré le flag `--no-quarantine` (depuis Homebrew 5.0.0, novembre 2025) et supprimera les casks échouant au Gatekeeper check au **1er septembre 2026** ; Apple Silicon refuse d'exécuter du code arm64 natif non signé. Pour macOS il faut réellement un compte Apple Developer (99 $/an) + notarisation, sinon distribuer via un tap non officiel.
- **Le vecteur d'attaque n°1 en 2025-2026 reste le job de release** : tag poussé sur un commit arbitraire, `pull_request_target`, empoisonnement du cache Actions, vol de secrets/OIDC en mémoire du runner. Isoler le job de release (déclencheur `tags:`, GitHub Environment avec required reviewers, secrets scopés à l'environnement) est aussi important que la signature.

## Key Findings

### 1. Provenance, attestation et signature
- `actions/attest-build-provenance` est, depuis la v4, un simple wrapper sur `actions/attest` ; GitHub recommande `actions/attest` pour les nouvelles implémentations. Il requiert `id-token: write` et `attestations: write`. Sur runners GitHub hébergés + dépôt public, il utilise l'instance Sigstore « public-good » et téléverse l'attestation à l'API GitHub. Coût : ~2-5 min de mise en place, quelques secondes de CI.
- **Niveau SLSA réellement atteignable** : générer une provenance = SLSA Build L1 ; avec les Artifact Attestations de GitHub sur runners hébergés = **SLSA v1.0 Build Level 2 par défaut** (confirmé par le GitHub Blog : « Generating build provenance puts you at SLSA Level 1, and by using GitHub Artifact Attestations on GitHub-hosted runners, you reach SLSA Level 2 by default » ; les GitHub Docs indiquent « Artifact attestations by itself provides SLSA v1.0 Build Level 2 »). Pour **SLSA Build L3**, il faut `slsa-framework/slsa-github-generator` (workflow réutilisable qui génère la provenance dans un job séparé avec son propre jeton OIDC que le build ne peut pas usurper).
- **SBOM** : `anchore/sbom-action` (syft) génère SPDX ou CycloneDX ; `actions/attest-sbom` signe l'attestation SBOM ; `docker/build-push-action` avec `sbom: true` + `provenance: mode=max` intègre provenance SLSA + SBOM directement dans le manifeste OCI poussé vers GHCR.
- **Sigstore/Rekor en 2026** : Rekor v2 (« Rekor on Tiles ») est passé GA en octobre 2025 ; backend tuilé (Trillian-Tessera), moins cher, mais Rekor v1 reste en mode maintenance pour la public-good instance. Cosign v3 (v3.0.1+) supporte Rekor v2. Les certificats Fulcio sont éphémères (~10 min) ; la confiance long terme repose sur l'entrée Rekor + timestamp, pas sur la validité du certificat au moment de la vérification.

### 2. Épinglage par SHA
- Il faut épingler **aussi** `actions/*` : l'incident tj-actions a montré que des tags mutables peuvent être réécrits. GitHub exclut cependant `actions/*`, `github/*` de certaines vérifications d'immutabilité par défaut.
- Depuis août 2025, la politique repo/org « Require actions to be pinned to a full-length commit SHA » (`sha_pinning_required`) peut faire **échouer** (et non plus seulement avertir) un workflow utilisant une action non épinglée, et bloquer des actions via préfixe `!`.
- Les **immutable releases** sont GA depuis le 28 octobre 2025 (assets et tag figés, attestations de release signées). Elles ne suppriment pas le besoin d'épingler par SHA mais rendent les pins fiables.
- Outils : Renovate (`helpers:pinGitHubActionDigests`), Dependabot (lit le commentaire de version après le SHA), `sethvargo/ratchet`, `stacklok/frizbee`, `mheap/pin-github-action`, zizmor (audit statique, détecte les impostor commits), StepSecurity Harden-Runner.

### 3. Permissions et isolation du job de release
- Passer le défaut org/repo à `read`. Ajouter `permissions:` explicite par workflow et par job (contents: read partout ; write uniquement là où nécessaire).
- Isoler la release : job déclenché par `on: push: tags:`, `environment:` avec required reviewers + deployment tag rules ; les secrets d'environnement ne sont accessibles qu'après validation des règles de protection.
- Incidents documentés à connaître : tj-actions/changed-files (CVE-2025-30066, mars 2025), reviewdog/action-setup (CVE-2025-30154), chaîne Coinbase, Shai-Hulud npm (sept 2025) et Shai-Hulud 2.0 (nov-déc 2025), TeamPCP/Trivy-Checkmarx (CVE-2026-33634, mars 2026), TanStack (mai 2026, première compromission npm avec provenance SLSA valide), Megalodon (mai 2026, 5 500+ repos).

### 4. Signature des tags et checksums
- `git tag -s` fonctionne avec GPG ou SSH (`gpg.format ssh` + `allowedSignersFile`). GitHub affiche « Verified » si la clé publique est enregistrée comme signing key. Clé publique publiable via `https://github.com/<user>.gpg` (GPG) et `.keys` (SSH).
- Pour signer `checksums.txt` : cosign keyless (bundle `.sigstore.json`) offre la meilleure ergonomie CI sans gestion de clé ; minisign est le plus simple pour l'utilisateur final ; GPG est le plus répandu mais lourd. GoReleaser supporte nativement `signs` (checksum), `docker_signs`, et la notarisation macOS.

### 5. Windows / macOS non signés
- SmartScreen : les certificats EV ne donnent plus de bypass instantané ; la réputation se construit par téléchargements. Une attestation/cosign n'a AUCUN effet.
- macOS : Gatekeeper attache l'attribut quarantine ; sans notarisation → « app is damaged » / « cannot be opened ». Depuis Sequoia (15), le clic-droit → Ouvrir est supprimé (passage par Réglages Système). Homebrew a retiré `--no-quarantine` et abandonne les casks échouant au Gatekeeper check au 1er septembre 2026.
- Options : certificat Authenticode OV (~100-200 $/an) / EV (~300-500 $/an, HSM obligatoire depuis juin 2023) ; Apple Developer 99 $/an + notarytool ; Azure Trusted Signing / Artifact Signing (~9,99 $/mois, US/CA/UE/UK) ; SignPath Foundation (gratuit OSS, certificat Sectigo au nom « SignPath Foundation »).
- Distribution qui contourne les avertissements : winget (bypass SmartScreen pour la source par défaut) ; Homebrew (mais bientôt signature obligatoire) ; Scoop (pas de signature requise) ; Chocolatey.

## Details

### 1. Attestations de provenance et signature

**Fonctionnement d'`actions/attest-build-provenance`.** L'action lie un sujet (artefact + digest SHA-256) à un prédicat de provenance SLSA au format in-toto, et génère une signature vérifiable via un certificat Sigstore/Fulcio éphémère. Sur dépôt public + runner GitHub hébergé, elle utilise l'instance Sigstore public-good ; sur dépôt privé/internal (Enterprise Cloud), l'instance Sigstore privée de GitHub. L'attestation est téléversée à l'API attestations de GitHub. **Prérequis de permissions** : `id-token: write` (émettre le jeton OIDC pour Fulcio) et `attestations: write` (persister l'attestation). Les attestations sont disponibles gratuitement pour les dépôts **publics** sur tous les plans actuels (pas les plans legacy Bronze/Silver/Gold).

**Migration importante 2026** : depuis la v4, `actions/attest-build-provenance` n'est qu'un wrapper d'`actions/attest` ; GitHub encourage les nouvelles implémentations à utiliser directement `actions/attest`. Les attestations générées utilisent le format bundle Sigstore v0.3.1 (vs v0.2.1) et exigent `gh` CLI ≥ 2.49.0 pour la vérification. L'ancien `github-early-access/generate-build-provenance` est déprécié.

**Niveaux SLSA sur runners GitHub hébergés.** Générer une provenance = **SLSA Build L1**. Les Artifact Attestations de GitHub sur runners hébergés = **SLSA v1.0 Build Level 2 par défaut** (le build tourne sur une plateforme hébergée managée, et la provenance est signée par la plateforme, pas par le script de build). Le passage à **L3** exige que la génération de provenance soit non-falsifiable et isolée du build : c'est ce qu'apporte `slsa-framework/slsa-github-generator`, qui exécute la génération dans un workflow réutilisable séparé, avec son propre jeton OIDC, que le code de build compromis ne peut pas usurper. Un prérequis L3 explicite (GitHub Blog / runwaylab) : « Have an isolated build environment: the build job must not use the Actions cache, as this can be poisoned by an attacker. » Pour un mainteneur solo, `actions/attest-build-provenance` (L2) est le meilleur rapport bénéfice/effort ; `slsa-github-generator` (L3) ajoute de la complexité pour un gain marginal à ce stade.

**Exemple de job de release avec attestation, SBOM et push GHCR :**
```yaml
name: release
on:
  push:
    tags: ['v*']

permissions:
  contents: read   # défaut restrictif au niveau workflow

jobs:
  release:
    runs-on: ubuntu-24.04
    environment: release          # gate secrets + required reviewers
    permissions:
      contents: write             # créer la release + uploader les assets
      packages: write             # push GHCR
      id-token: write             # OIDC pour cosign keyless + attestation
      attestations: write         # persister les attestations
    steps:
      - uses: actions/checkout@<SHA>       # v4.x.x — re-résoudre le SHA
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@<SHA>       # v5.x.x
        with: { go-version: '1.23' }
      - uses: sigstore/cosign-installer@<SHA>   # v3.x.x
      - uses: anchore/sbom-action/download-syft@<SHA> # v0.x.x
      - uses: docker/login-action@<SHA>    # v3.x.x
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: goreleaser/goreleaser-action@<SHA>  # v6.x.x
        with: { version: latest, args: release --clean }
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      # Attestation de provenance des binaires produits par GoReleaser
      - uses: actions/attest-build-provenance@<SHA>  # v2.x.x (wrapper d'actions/attest)
        with:
          subject-path: 'dist/*.tar.gz, dist/*.zip, dist/checksums.txt'
```

**Image conteneur : provenance SLSA + SBOM dans le manifeste OCI.**
```yaml
      - uses: docker/setup-buildx-action@<SHA>   # v3.x.x
      - id: build
        uses: docker/build-push-action@<SHA>     # v6.x.x
        with:
          context: .
          push: true                # OBLIGATOIRE pour que provenance/SBOM soient poussés
          tags: ghcr.io/OWNER/IMAGE:${{ github.ref_name }}
          provenance: mode=max      # min par défaut ; max = provenance SLSA détaillée
          sbom: true                # génère le SBOM et l'embarque
      # Attestation GitHub liée à l'image (par digest) + SBOM signé
      - uses: actions/attest-build-provenance@<SHA>
        with:
          subject-name: ghcr.io/OWNER/IMAGE
          subject-digest: ${{ steps.build.outputs.digest }}
          push-to-registry: true
```

**SBOM (approfondissement).** Deux approches complémentaires : (a) SBOM des binaires/archives via `anchore/sbom-action` (moteur syft), formats SPDX-JSON ou CycloneDX, puis `actions/attest-sbom` pour attester ; (b) SBOM + provenance de l'image conteneur via `docker/build-push-action` avec `provenance: mode=max` et `sbom: true` — attention, un bug connu fait que sans `push: true` les attestations sont générées mais non poussées. SPDX (norme ISO, orienté licences/conformité) vs CycloneDX (orienté sécurité/VEX) : pour un projet OSS, SPDX est le choix le plus consensuel pour la conformité, CycloneDX pour l'analyse de vulnérabilités.

**Attestation SBOM (binaires) :**
```yaml
      - uses: anchore/sbom-action@<SHA>
        with:
          format: spdx-json
          output-file: sbom.spdx.json
      - uses: actions/attest-sbom@<SHA>
        with:
          subject-path: 'dist/mybinary_linux_amd64'
          sbom-path: 'sbom.spdx.json'
```

**Signature keyless cosign.** Cosign v3 signe blobs et images ; en mode keyless il demande un certificat Fulcio éphémère lié à l'identité OIDC (`https://token.actions.githubusercontent.com` en CI) et enregistre l'entrée dans Rekor. Les certificats Fulcio expirent en ~10 min : la vérification long terme ne s'appuie PAS sur la validité du certificat mais sur l'entrée horodatée dans Rekor (transparence). **Statut Sigstore 2026** : Rekor v2 GA (octobre 2025), backend tuilé Tessera ; cosign ≥ v3.0.1 requis ; Rekor v1 en maintenance pour la public-good instance. Toujours signer une image par son digest (`@sha256:...`), jamais par un tag mutable.

**Configuration GoReleaser (signature keyless checksums + image) :**
```yaml
# .goreleaser.yaml
checksum:
  name_template: 'checksums.txt'

signs:
  - cmd: cosign
    signature: '${artifact}.sigstore.json'
    args: ['sign-blob', '--bundle=${signature}', '${artifact}', '--yes']
    artifacts: checksum          # signe uniquement checksums.txt (keyless)

docker_signs:
  - cmd: cosign
    args: ['sign', '${artifact}', '--yes']
    artifacts: images            # keyless : pas de --key
```

**Vérification côté utilisateur (à documenter dans README/SECURITY.md).**
```bash
# 1. Intégrité classique
sha256sum -c checksums.txt

# 2. Attestation GitHub d'un binaire (en ligne)
gh attestation verify ./mybinary \
  --repo OWNER/REPO \
  --signer-workflow OWNER/REPO/.github/workflows/release.yml \
  --predicate-type https://slsa.dev/provenance/v1

# 3. Vérification HORS-LIGNE
gh attestation trusted-root > trusted_root.jsonl
gh attestation verify ./mybinary \
  --bundle attestation.jsonl \
  --custom-trusted-root trusted_root.jsonl

# 4. Attestation d'une image GHCR
gh attestation verify oci://ghcr.io/OWNER/IMAGE:TAG --repo OWNER/REPO

# 5. Cosign : image (identité = workflow OIDC)
cosign verify ghcr.io/OWNER/IMAGE@sha256:... \
  --certificate-identity-regexp "^https://github.com/OWNER/REPO/.github/workflows/release.yml@.*" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

# 6. Cosign : checksums.txt (bundle)
cosign verify-blob checksums.txt \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp "^https://github.com/OWNER/REPO/.*" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

# 7. Cosign : attestation SBOM d'une image
cosign verify-attestation ghcr.io/OWNER/IMAGE@sha256:... --type spdxjson \
  --certificate-identity-regexp "^https://github.com/OWNER/REPO/.*" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```
*Note : des bugs de vérification offline ont été signalés sur `gh` v2.60.1 fin 2024 ; utiliser une version récente du CLI.*

### 2. Épinglage par SHA des actions

**Faut-il épingler `actions/*` ?** Oui. Un tag Git est mutable (déplaçable, supprimable). L'incident tj-actions a réécrit les tags v1→v45 vers le commit malveillant `0e58ed8671d6b60d0890c21b07f8835ace038e67`. Wiz note explicitement : « Customers who were using a hash-pinned version… would not be impacted. » Épingler par SHA 40 caractères (avec le tag en commentaire) rend le code immuable. **Piège documenté** : l'épinglage par SHA ne prouve pas la provenance du commit — GitHub partageant un pool d'objets entre un dépôt et ses forks, `owner/action@<sha>` peut résoudre un commit n'existant que sur un fork d'attaquant (« impostor commit »). zizmor détecte cette classe. Argument contre l'épinglage de `actions/*` : lourdeur de maintenance ; contre-argument : une politique uniforme est plus auditable, et Dependabot/Renovate automatisent les mises à jour.

**Bon vs mauvais :**
```yaml
# Mauvais : mutable
- uses: actions/checkout@v4
# Bon : immuable, avec version en commentaire (lue par Dependabot/Renovate)
- uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8 # v5.0.0
```

**Configuration Renovate (maintien des pins) :**
```json
{
  "extends": ["config:recommended", "helpers:pinGitHubActionDigests"]
}
```
**Ou Dependabot :**
```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule: { interval: "weekly" }
    # Dependabot lit le commentaire "# vX.Y.Z" et ouvre une PR mettant à jour le SHA
```

**Politique `sha_pinning_required`.** Depuis août 2025, la politique « Require actions to be pinned to a full-length commit SHA » existe au niveau repo ET org, et fait désormais échouer (pas seulement avertir) tout workflow utilisant une action non épinglée ; elle vérifie l'arbre de dépendances complet (sous-actions incluses) et supporte le blocage d'actions via préfixe `!`. Se règle dans Settings → Actions → General.

**Immutable releases / immutable actions.** GA depuis le 28 octobre 2025 : une fois une release publiée immuable, ses assets et son tag ne peuvent plus être modifiés/supprimés ; elle reçoit des attestations de release signées. Cela rend les tags fiables MAIS ne remplace pas l'épinglage SHA (il n'existe pas de politique native exigeant que les actions consommées utilisent des immutable releases). La feuille de route GitHub Actions 2026 (post du 26 mars 2026) évoque un déplacement « vers » les immutable releases comme direction future (« On the publishing side, we're moving away from mutable references and towards immutable releases »), sans en faire un défaut obligatoire.

**Recommandation mainteneur solo** : Renovate avec preset `helpers:pinGitHubActionDigests` (ou Dependabot qui lit désormais les commentaires de version) + un passage initial de `sethvargo/ratchet pin` ou `mheap/pin-github-action`, + zizmor en CI pour l'audit. Activer la politique repo `sha_pinning_required`.

### 3. Permissions et isolation des jobs de release

**Défaut de permissions.** GitHub a fait migrer le défaut du `GITHUB_TOKEN` de read/write vers read only ; il faut vérifier que le repo est bien sur `read` (Settings → Actions → General → Workflow permissions). Une campagne org courante fin 2025 a poussé les mainteneurs à définir des `permissions:` explicites. Dès qu'on déclare un bloc `permissions:`, toute permission non listée passe à `none`.

**Permissions minimales par étape** :

| Étape | Permission minimale |
|---|---|
| build (checkout + compilation) | `contents: read` |
| création de release + upload d'assets | `contents: write` |
| attestation de provenance/SBOM | `id-token: write`, `attestations: write` |
| push image GHCR | `packages: write` |
| signature cosign keyless | `id-token: write` |

**Isolation du job de release.** Un tag peut être poussé sur **n'importe quel commit** (y compris hors branche par défaut). Mitigations :
- Déclencher la release uniquement sur `on: push: tags: ['v*']`.
- **Tag ruleset « Restrict creations »** : seuls les utilisateurs/équipes/apps avec bypass peuvent créer des tags `v*` (les tag protection rules legacy sont dépréciées en GHES 3.16+, migrées vers rulesets). Docs GitHub : « Restrict creations — If selected, only users with bypass permissions can create branches or tags whose name matches the pattern. »
- **GitHub Environment** : `environment: release` sur le job. Docs GitHub : « Secrets stored in an environment are only available to workflow jobs that reference the environment. If the environment requires approval, a job cannot access environment secrets until one of the required reviewers approves it. » On peut aussi restreindre par « deployment branches and tags » (option « Selected branches and tags » → règle de type Tag `v*`). Jusqu'à 6 required reviewers, wait timer jusqu'à 30 jours, option « prevent self-review ». Sur plan gratuit, les required reviewers/wait timer ne sont dispo que pour les dépôts publics.
- Optionnel : vérifier dans le job que le tag pointe sur un commit de la branche par défaut.

**Incidents documentés et leçons** :
- **tj-actions/changed-files** (CVE-2025-30066, CVSS 8.6, 14 mars 2025 ; advisory GHSA-mrrh-fwg8-r2c3) : PAT de bot compromis → réécriture des tags v1-v45 vers le commit malveillant `0e58ed8…` → dump des secrets du runner dans les logs. Plus de **23 000 dépôts** concernés (StepSecurity/Wiz). Leçon : épingler par SHA, restreindre les permissions du token, ne pas exposer de secrets dans les logs.
- **reviewdog/action-setup** (CVE-2025-30154) : compromission en amont ayant potentiellement permis tj-actions.
- **Shai-Hulud** (npm, 16 sept 2025) : ver auto-propageant ; le nombre de paquets infectés a dépassé **500** (Truesec, MàJ 18 sept 2025), première vague ~40 paquets dont `@ctrl/tinycolor` (2M+ téléchargements/sem). Moisson de secrets CI/CD, création de repos `Shai-Hulud`, workflow `shai-hulud-workflow.yml`.
- **Shai-Hulud 2.0** (« The Second Coming », identifié le 24 nov 2025) : Datadog Security Labs a compté **796 paquets npm uniques backdoorés, sur 1 092 versions uniques** (>20M téléchargements/sem) ; Unit 42 (Palo Alto) : « over 25,000 malicious repositories across about 350 unique users », exécution en phase **pre-install**, backdoor via self-hosted runner, dead-man's switch. Répliques 2026 (« Mini Shai-Hulud », « The Third Coming »).
- **TeamPCP / CVE-2026-33634** (mars 2026, CVSS 9.4) : rotation de credentials incomplète après une brèche → force-push des 35 tags de `Checkmarx/kics-github-action` vers des impostor commits, stealer lisant `/proc/<pid>/mem` du Runner.Worker. Le badge « Immutable » s'affichait normalement.
- **TanStack** (mai 2026) : `pull_request_target` + empoisonnement du cache Actions → vol du jeton OIDC en mémoire → 84 versions npm malveillantes. **Première compromission npm portant une provenance SLSA valide** — leçon clé : la provenance atteste du « où/comment » du build, PAS de l'innocuité du code.
- **Megalodon** (18 mai 2026) : 5 500+ repos, injection de workflows `workflow_dispatch` dormants (exemptés des règles anti-récursion de GitHub).

**Durcissements GitHub 2026 (confirmés, shippés)** : `actions/checkout` a un défaut plus sûr contre le checkout de code de fork non fiable sur `pull_request_target` (18 juin 2026) ; cache Actions en lecture seule pour les triggers non fiables (26 juin 2026) ; contrôle « qui/quoi déclenche les workflows » (juin 2026) ; cooldown Dependabot de 3 jours par défaut (14 juillet 2026 — « Version updates through Dependabot now wait until a release has been available for at least three days before opening a pull request » ; les mises à jour de sécurité restent immédiates). En préparation (roadmap mars 2026, projections) : « workflow dependency locking » (section `dependencies:` façon go.mod/go.sum, preview visée 3-6 mois), politiques d'exécution centralisées via rulesets, secrets scopés, pare-feu réseau natif des runners.

### 4. Signature des tags et des checksums

**Tags signés.** `git tag -s` avec GPG (`gpg.format openpgp`) ou SSH. Configuration SSH signing :
```bash
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub
git config --global gpg.ssh.allowedSignersFile ~/.config/git/allowed_signers
git config --global tag.gpgsign true
```
Un tag/commit signé s'affiche « Verified » sur GitHub si la clé publique est enregistrée comme **signing key** dans le profil (distinct des authentication keys). Publier la clé : `https://github.com/<user>.gpg` (GPG), `https://github.com/<user>.keys` (clés SSH), keys.openpgp.org, fichier dans le dépôt, site web. Vérification locale SSH : nécessite un fichier `allowed_signers` (sinon `error: gpg.ssh.allowedSignersFile needs to be configured`).

**Signer checksums.txt.** Comparatif :
- **cosign keyless** : aucun secret à gérer (identité OIDC en CI), bundle `.sigstore.json` ; l'utilisateur vérifie avec `cosign verify-blob`. Meilleur pour un mainteneur solo en CI. Inconvénient : l'utilisateur doit installer cosign et connaître l'identité attendue.
- **minisign** : clé unique, signatures courtes, très simple pour l'utilisateur ; mais gestion de la clé privée (secret GitHub). Vérification : `minisign -Vm checksums.txt -P <clé publique>`.
- **GPG** : le plus universel/attendu par les distributions Linux, mais UX lourde et gestion de clé. Vérification : `gpg --verify checksums.txt.asc checksums.txt`.

GoReleaser supporte les trois via `signs:` (voir bloc plus haut), plus `docker_signs:` pour les images, et la notarisation macOS.

**Keyless vs clé privée en secret.** Pour un mainteneur solo, le keyless (cosign/attestation) évite le risque n°1 : la fuite/rotation d'une clé privée stockée en secret GitHub. Le compromis : dépendance à l'infra Sigstore et à l'identité OIDC. Une clé longue durée en secret n'a de sens que pour minisign/GPG si l'on tient à une vérification sans dépendance réseau à Sigstore.

**Documentation** : mettre les commandes de vérification dans le README et un `SECURITY.md`, avec l'identité de signataire exacte (chemin du workflow) et l'empreinte de clé publique.

### 5. Applications non signées Windows et macOS

**Windows / SmartScreen.** SmartScreen valide la signature Authenticode ET la réputation. Un binaire non signé (ou signé par un nouveau certificat) démarre à réputation zéro et déclenche « Windows protected your PC ». Les certificats **EV ne donnent plus de bypass instantané** de SmartScreen (Microsoft Learn : « EV certificates no longer bypass SmartScreen »). La réputation ne se transfère pas d'une version à l'autre sauf si les deux sont signées avec la même identité d'éditeur. Une attestation SLSA ou une signature cosign n'a **aucun effet** sur SmartScreen.

**macOS / Gatekeeper.** Tout fichier téléchargé reçoit l'attribut `com.apple.quarantine` ; sans signature Developer ID + notarisation, Gatekeeper bloque (« app is damaged and can't be opened »). Depuis macOS Sequoia (15), le contournement par clic-droit → Ouvrir est supprimé : l'utilisateur doit passer par Réglages Système → Confidentialité et sécurité. Sur Apple Silicon, le code arm64 natif non signé ne s'exécute pas du tout (Homebrew : « Macs with Apple silicon also don't permit native arm64 code to execute unless a valid signature is attached »). **Homebrew** : le flag `--no-quarantine` est déjà **déprécié/retiré depuis Homebrew 5.0.0 (novembre 2025)** ; et le support des casks échouant au Gatekeeper check prend **fin le 1er septembre 2026** (« we are ending support for all casks that fail Gatekeeper checks on September 1st, 2026 ») — les apps non notarisées seront retirées de homebrew-cask.

**La signature de provenance ≠ certificat de signature de code.** À dire clairement dans la doc : une attestation/cosign prouve l'origine et l'intégrité du build pour un utilisateur averti (vérifiable en ligne de commande), mais ne supprime aucun avertissement OS. Les deux sont complémentaires, pas substituables.

**Notarisation macOS en CI (extrait) :**
```yaml
  # sur runner macos-latest, après signature codesign
  - run: |
      xcrun notarytool submit app.zip \
        --key "$KEY_PATH" --key-id "$KEY_ID" --issuer "$ISSUER" --wait
      xcrun stapler staple "MyApp.app"   # ou le .dmg
```

**Coûts/options 2026** :
- Authenticode OV : ~100-200 $/an ; EV : ~300-500 $/an, stockage matériel HSM/token obligatoire depuis juin 2023.
- Apple Developer Program : 99 $/an ; notarisation via `xcrun notarytool submit --wait` + `xcrun stapler staple` (GoReleaser gère via anchore/quill cross-platform ou codesign/xcrun natif ; certificat « Developer ID Application » pour DMG).
- Azure Trusted Signing / Artifact Signing : à partir de ~9,99 $/mois, signe uniquement Windows, éligibilité entreprises et indépendants en US/Canada/UE/UK ; SmartScreen construit la réputation normalement (fonctionnellement équivalent à un OV depuis la fin du bypass EV).
- SignPath Foundation : signature OV gratuite pour projets OSS qualifiés (certificat Sectigo, éditeur affiché « SignPath Foundation »).

**Canaux de distribution** :
- **winget** : le client winget bypasse SmartScreen pour les sources par défaut ; la soumission à winget-pkgs recommande fortement des installeurs signés (les installeurs non signés sont bloqués par Defender SmartScreen lors du téléchargement, ce qui fait échouer `winget install`).
- **Homebrew** : casks bientôt soumis à signature/notarisation obligatoire (1er sept 2026) ; les **formules** (CLI, compilées depuis la source) sont moins affectées ; option de tap non officiel pour apps non signées.
- **Scoop** : pas d'exigence de signature, effort minimal, excellent pour les CLI Windows.
- **Chocolatey** : signature recommandée, modération, approbation plus rapide si signé.
- **Microsoft Store** : pour MSI/EXE, Microsoft ne re-signe pas ; certificat chaînant à une CA du Microsoft Trusted Root Program requis.

## Recommandations (ordonnées par rapport bénéfice/effort)

1. **Verrouiller les permissions (~30 min, maintenance quasi nulle).** Défaut org/repo `read` + blocs `permissions:` par job. Bénéfice : neutralise l'essentiel de l'impact d'une action compromise. Piège : casser un workflow qui écrivait implicitement — tester avant.
2. **Épingler toutes les actions par SHA + Renovate/Dependabot (1-2 h initiales, maintenance automatisée).** Activer la politique `sha_pinning_required`. Bénéfice : bloque la classe tj-actions/TeamPCP. Piège : impostor commits (auditer avec zizmor).
3. **Ajouter la provenance `actions/attest-build-provenance` (2-4 h, +quelques s CI).** Bénéfice : SLSA Build L2, vérifiable par `gh attestation verify`. Piège : bien mettre `id-token: write` + `attestations: write` ; documenter la commande de vérification.
4. **Isoler le job de release (2-4 h).** Déclencheur `tags:`, `environment:` avec required reviewers, tag ruleset « Restrict creations ». Bénéfice : protège les secrets même si un tag est poussé sur un commit arbitraire.
5. **Signer image + checksums avec cosign keyless (2-3 h via GoReleaser `docker_signs`/`signs`).** Bénéfice : signature sans clé, transparence Rekor. Piège : signer par digest, documenter l'identité OIDC.
6. **SBOM (1-2 h).** `docker/build-push-action` `sbom: true` + `provenance: mode=max` (avec `push: true`) pour l'image ; `anchore/sbom-action` + `actions/attest-sbom` pour les binaires. Bénéfice : conformité procurement, réponse CVE.
7. **Signer les tags Git (~30 min).** SSH signing (plus simple que GPG), clé publique sur le profil GitHub.
8. **macOS : Apple Developer 99 $/an + notarytool (4-8 h initiales, ~1 h/an).** Devenu quasi incontournable vu le retrait de `--no-quarantine` (Homebrew 5.0.0) et la fin des casks non conformes (1er sept 2026), plus Apple Silicon. Alternative gratuite : tap non officiel + instructions `xattr -dr com.apple.quarantine`.
9. **Windows : Azure Trusted Signing (~9,99 $/mois) si éligible, sinon SignPath Foundation (gratuit OSS).** Bénéfice : suppression progressive des avertissements SmartScreen. Distribuer via winget/Scoop pour contourner SmartScreen.

**Seuils de décision** : si le projet cible surtout des développeurs (CLI, conteneurs), les priorités 1-7 suffisent — provenance + cosign + doc de vérification. Si l'audience inclut des utilisateurs grand public sur Windows/macOS (apps GUI), la signature de code OS (8-9) devient indispensable — et pour macOS, la deadline Homebrew du 1er septembre 2026 en fait une échéance concrète. Passer à SLSA L3 (`slsa-github-generator`) uniquement si un consommateur l'exige contractuellement.

## Caveats
- **La feuille de route GitHub Actions 2026 (workflow lockfiles, politiques d'exécution centralisées, secrets scopés, pare-feu runner) est constituée de projections** annoncées le 26 mars 2026 (verbes « nous introduisons/construisons »), avec des cibles de 3-9 mois — à traiter comme des plans, non comme des fonctionnalités livrées. Seuls les éléments à date de changelog confirmée (cooldown Dependabot 14 juillet 2026, defaults `actions/checkout` 18 juin 2026, cache lecture seule 26 juin 2026, contrôle des déclencheurs juin 2026) sont shippés.
- **La provenance/attestation n'est PAS une garantie d'innocuité du code** : l'attaque TanStack (mai 2026) a produit des paquets npm malveillants avec provenance SLSA valide. La provenance atteste le « où/comment », pas le « quoi ».
- **Certaines dates/chiffres d'incidents 2026** (TeamPCP, TanStack, Megalodon, Mini Shai-Hulud) proviennent de blogs de recherche et de postmortems ; les CVE (CVE-2026-33634) et advisories GitHub corroborent les principaux. Les chiffres Shai-Hulud varient selon les sources et les vagues (≥500 paquets en sept 2025 selon Truesec ; 796 paquets / 1 092 versions pour la 2.0 selon Datadog).
- **Les immutable releases ne remplacent pas l'épinglage SHA** ; l'incident TeamPCP a montré que le badge « Immutable » peut s'afficher tout en pointant du code force-pushé après compromission de credentials.
- **Vérifier les versions/SHA exacts** au moment de la mise en place : tous les `@<SHA>` des exemples doivent être re-résolus contre le dépôt canonique de chaque action (et non un fork).
- Le retrait de `--no-quarantine` de Homebrew (5.0.0, nov 2025) et la deadline du 1er sept 2026 sont deux événements distincts — le flag n'existe plus déjà, la purge des casks non conformes vient ensuite.