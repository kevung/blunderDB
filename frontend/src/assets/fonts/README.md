# Polices embarquées

Tout ce dossier part dans chaque binaire (`//go:embed all:frontend/dist`), d'où
les sous-ensembles WOFF2 plutôt que les fontes complètes.

## NotoSansJP-Regular.woff2

Sous-ensemble de Noto Sans JP Regular 2.004 (licence : `OFL.txt`). La fonte
complète pèse 5,7 Mo en TTF ; le sous-ensemble couvre les caractères de
l'interface japonaise et les blocs kana/ponctuation, soit ~180 Ko.

Il contient :

- tous les caractères de `frontend/src/i18n/locales/ja.json` et
  `frontend/src/i18n/help/ja.js` (donc les kanji que l'interface affiche) ;
- Latin de base et Latin-1 (U+0000-00FF) ;
- ponctuation CJK (U+3000-303F), Hiragana (U+3040-309F), Katakana
  (U+30A0-30FF), formes pleine largeur (U+FF00-FFEF).

Les kanji absents de l'interface (noms de joueurs, commentaires saisis par
l'utilisateur…) tombent sur la police CJK de l'hôte, comme avant pour les
caractères hors de Noto Sans JP.

### Régénérer après une modification de `ja.json` ou `help/ja.js`

Depuis ce dossier, avec le TTF complet à portée (fonttools ≥ 4.62,
`pyftsubset` dans le PATH ; le TTF se télécharge sur
<https://fonts.google.com/noto/specimen/Noto+Sans+JP>) :

```sh
pyftsubset NotoSansJP-Regular.ttf \
  --output-file=NotoSansJP-Regular.woff2 --flavor=woff2 \
  --text-file=../../i18n/locales/ja.json --text-file=../../i18n/help/ja.js \
  --unicodes="U+0000-00FF,U+3000-303F,U+3040-309F,U+30A0-30FF,U+FF00-FFEF" \
  --layout-features='*' --no-hinting --desubroutinize
```

Puis vérifier que rien ne manque (la liste doit être vide ; un caractère que
la fonte complète ne possède pas non plus, comme `↺` U+21BA, est ignoré) :

```sh
python - <<'PY'
from fontTools.ttLib import TTFont
full = TTFont('NotoSansJP-Regular.ttf').getBestCmap()
sub = TTFont('NotoSansJP-Regular.woff2').getBestCmap()
missing = set()
for p in ('../../i18n/locales/ja.json', '../../i18n/help/ja.js'):
    for ch in open(p, encoding='utf-8').read():
        if ord(ch) in full and ord(ch) not in sub:
            missing.add(f'U+{ord(ch):04X} {ch}')
print('manquants :', sorted(missing) or 'aucun')
PY
```

## nunito-v16-latin-regular.woff2

Nunito Regular, sous-ensemble latin (inchangé, ~19 Ko).
