# Déploiement de `blunderdb serve` derrière un proxy authentifiant

## Modèle de menace, en trois phrases

`blunderdb serve` (ADR-0005) n'effectue **aucune authentification** : il fait
confiance à l'en-tête `X-Tenant-ID` exactement tel qu'il le reçoit, et toute
requête qui l'atteint directement peut lire ou écrire les données de
**n'importe quel** tenant en le nommant. Toute la sécurité du déploiement
tient donc dans le reverse-proxy placé devant lui, qui doit authentifier
l'appelant, retirer tout `X-Tenant-ID` envoyé par le client puis injecter
l'entier correspondant au compte authentifié (jamais un nom — voir
l'amendement du 05/09/2026 de l'ADR-0005). PostgreSQL Row-Level Security
(`--rls` / `BLUNDERDB_RLS=true`) est une défense en profondeur *à l'intérieur*
de cette frontière, pas un substitut : elle protège contre un bug de handler,
pas contre un proxy mal configuré.

## À ne jamais faire

**Ne jamais exposer le démon nu** : pas de `ports:` sur `blunderdb-serve` dans
`docker-compose.yml`, pas de règle de pare-feu qui l'atteint directement, pas
de tunnel qui contourne le proxy. `docker-compose.yml` place `blunderdb-serve`
et `postgres` sur un réseau Docker `internal: true` — seul Caddy y a accès, et
Caddy est le seul service dont un port est publié sur l'hôte.

## Ce que fournit ce dossier

- `docker-compose.yml` — Caddy (authentification HTTP Basic de démonstration)
  + `blunderdb-serve` + PostgreSQL avec RLS activée.
- `Caddyfile` — la configuration de Caddy : authentifie, puis mappe le compte
  authentifié vers l'entier du tenant, puis l'injecte dans `X-Tenant-ID` après
  avoir explicitement effacé toute valeur reçue du client.
- `nginx-tenant-proxy.conf` — le même schéma (garde + injection), en snippet
  nginx pour qui a déjà un nginx en place plutôt que Caddy.
- `.env.example` — variable `POSTGRES_PASSWORD` à définir avant de lancer.

L'authentification HTTP Basic ci-dessus est une **démonstration**, pas une
recommandation : en production, remplacez-la par `forward_auth` vers votre
fournisseur d'identité réel (OIDC, SSO d'entreprise…), qui authentifie puis
transmet l'identité au même endroit du Caddyfile.

Le scénario complet, du conteneur vide à un démon qui répond (avec et sans
en-tête), est documenté dans `doc/source/mode_headless.rst`, section
« Déploiement derrière un proxy authentifiant ».
