# Documentation de la vision, à republier à l'activation

La documentation publiée ne décrit que ce qui est disponible. Le jour où `VISION_AVAILABLE`
(`frontend/src/vision/available.js`) passe à vrai, remettre ce texte, puis les huit `.po`
(`doc/CLAUDE.md`) et `make help`.

## `doc/source/manuel_matchs.rst`

Avant le paragraphe « Des repères se déduisent les **durées** de décision » :

```rst
.. _transcription_vision:

**La vue du plateau et la ligne de temps.** Le bouton de calibration de la barre
vidéo désigne les quatre coins du plateau dans l'image (points 1, 12, 13 puis
24) ; les calibrations sont nommées et retenues pour la vidéo, et le bouton ◌
affiche des pions fantômes pour vérifier le calage. Quand le stream affiche un
overlay de score ou de position, le bouton ▭ de la même barre en trace la zone :
deux clics sur deux coins opposés, puis ▭✕ pour l'oublier. Une zone trop petite
pour être surveillée est refusée (« Calibration échouée »). L'analyse de la
vidéo suit alors le plateau et cette zone ; elle n'écrit rien dans le
brouillon.

Sous la vidéo, une **ligne de temps** place les événements que l'analyse a
relevés : jet, repos et reprise des dés, début et fin de partie, plateau perdu ou
retrouvé, overlay modifié. Un clic sur un marqueur amène la vidéo à cet instant,
lecture ou pause conservée ; son info-bulle donne l'événement et son état. La
**forme** dit la certitude : un cercle pour un événement sûr, un losange pour un
événement à vérifier, un triangle pour un événement incertain ; la couleur ne
fait que la doubler. *g* saute au marqueur suivant et *MAJ-G* au précédent, tous
états confondus.

Le premier dé saisi d'une action pose le repère du jet à l'instant où les dés se
sont immobilisés sur la table, quand ce repos est sûr et date d'au plus une
minute ; sinon, ou si le plateau est hors de vue, il garde l'instant courant.

Deux indicateurs signalent ce que l'analyse ne peut garantir. **Vue perdue**
apparaît dans la barre de calibration tant que le plateau calibré n'est plus
dans l'image. Le pictogramme ◐ au-dessus de la vidéo signale, pendant trente
secondes, que l'overlay a changé sans qu'un événement de la partie l'explique :
la zone tracée ne correspond peut-être plus à la partie et se retrace par ▭.
```

## `doc/source/raccourcis.rst`

Dans la table du panneau de transcription, entre `[ / ] (vidéo attachée)` et `o (vidéo attachée)` :

```rst
   "g / MAJ-G (vidéo attachée)", "Sauter au marqueur suivant / précédent de la ligne de temps sous la vidéo, lecture ou pause conservée (:ref:`détail <transcription_vision>`)."
```
