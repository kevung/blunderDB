Traductions (gettext, 8 langues : en, de, el, es, fi, it, ja, ru — le
français de source/ n'a pas de catalogue, c'est la référence)
==============================================================

En une commande, depuis la racine du dépôt :

    source .venv/bin/activate   # sphinx-build et sphinx-intl doivent être sur le PATH
    scripts/doc-po-update.sh    # régénère les huit catalogues, répare ce qui suit
    # ... traduire les msgstr vides (Lokalize, Poedit, ou à la main) ...
    scripts/doc-i18n-check.sh   # doit terminer sur "all translations complete"

N'appelez pas sphinx-build/sphinx-intl à la main : scripts/doc-po-update.sh
encode trois pièges déjà rencontrés, chacun documenté en tête du script
lui-même — chemin de sortie du gettext RELATIF (un chemin absolu réécrit
la référence source `#:` de chaque chaîne, dans les huit catalogues, et
noie le vrai diff sous des milliers de lignes) ; le drapeau `python-format`
que msgmerge appose dès qu'un texte français contient un `%` suivi d'une
espace (« 93,4 % (3735/4000) » ressemble à un directive `% (`) — la
normalisation le retire après coup, il revient à chaque régénération et ne
concerne aucune vraie chaîne de format Python de ce projet ; et jamais de
passage par `msgcat` pour reformater un catalogue, Babel et msgcat ne
coupant pas les lignes de la même façon (un passage réécrit tout le
fichier).

Piège supplémentaire, non lié aux scripts : `msgfmt` appelé sur plusieurs
catalogues à la fois échoue

    msgfmt -c -o /dev/null source/locale/en/LC_MESSAGES/*.po
    # source/locale/en/LC_MESSAGES/annexe_filtres.po:6: duplicate message definition
    # source/locale/en/LC_MESSAGES/annexe_db_scheme.po:7: ...this is the location of the first definition

`msgfmt` accepte plusieurs fichiers pour les CONCATÉNER en un seul
catalogue (cas d'usage : un .po éclaté en plusieurs morceaux) ; il traite
alors l'en-tête (`msgid ""`) de chacun comme une définition du même
message et refuse le second en doublon. Ce n'est pas un catalogue corrompu
— c'est la vérification elle-même qui doit tourner **fichier par fichier** :
c'est ce que fait `scripts/doc-po-update.sh` (boucle `for po in
doc/source/locale/*/LC_MESSAGES/*.po`) et ce que fait `scripts/doc-i18n-check.sh`
en interne. Ne réintroduisez pas un glob `msgfmt *.po` dans un script ou un
hook : il échouera systématiquement, même quand chaque catalogue pris
séparément est valide.

Éditer un .po
-------------

Modifiez le champ `msgstr` d'une entrée directement (Lokalize, Poedit, ou
un éditeur de texte). **N'éditez jamais un catalogue par une substitution
regex/sed globale** : un motif générique correspond aussi bien à l'entrée
visée qu'à sa voisine (les échappements `\"`, `\\`, les retours à la ligne
`\n` internes à une chaîne rendent un remplacement "à l'oeil" trompeur),
et une regex qui touche par erreur un texte japonais peut casser
l'échappement `\ ` (barre oblique inverse suivie d'une espace) que la
convention de ce projet exige aux deux frontières d'un balisage RST
(`` `` ``, `**`, `*`) collé à un caractère CJK sans espace — voir
CONTRIBUTING.md. Pour un remplacement programmatique (traduire un lot
d'entrées, par exemple), passez par `polib` (déjà installé dans le
virtualenv : c'est une dépendance de `sphinx-intl`, listé dans
doc/requirements.txt) : chargez le fichier, modifiez l'entrée par son
`msgid` exact (unique dans un catalogue), et n'appelez `POFile.save()`
qu'après avoir vérifié qu'il ne réencode PAS tout le fichier avec un style
de retour à la ligne différent de celui déjà commité — `polib` a son
propre style de découpe des lignes longues, distinct de celui de Babel,
et un `save()` sur le fichier entier produit le même effet qu'un passage
par `msgcat` (un diff de plusieurs milliers de lignes pour une poignée de
chaînes changées). Ne réécrivez que le nécessaire.
