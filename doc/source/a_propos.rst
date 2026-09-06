.. _a_propos:

À propos
========

.. _contacts:

Contacts
--------

Auteur: Kévin Unger <blunderdb@proton.me>.
Vous pouvez aussi me trouver sur Heroes sous le pseudo postmanpat.

J'ai développé blunderDB initialement pour mon usage personnel afin de
pouvoir détecter des motifs dans mes erreurs. Mais il est très agréable
d'avoir un retour surtout quand on a dépensé un paquet d'heures de
conception, codage, débuggage... Aussi n'hésitez pas à m'écrire pour
faire part de votre retour d'expérience. Tous les retours (constructifs)
sont bienvenus.

Voici plusieurs manières de discuter:

* rejoindre le serveur Discord de blunderDB: https://discord.gg/DA5PpzM9En

* m'écrire un mail à blunderdb@proton.me,

* discuter avec moi, si on se retrouve dans un tournoi,

* sur Github,

  * ouvrir un ticket: https://github.com/kevung/blunderDB/issues

  * pour des corrections de bugs ou des propositions d'amélioration,
    créer une pull request.

Les tickets ouverts sont regroupés par `jalon
<https://github.com/kevung/blunderDB/milestones>`__ sur GitHub. Les idées se
discutent dans les `Discussions — catégorie Idées
<https://github.com/kevung/blunderDB/discussions/categories/ideas>`__, et
chaque version publiée est annoncée dans la `catégorie Annonces
<https://github.com/kevung/blunderDB/discussions/categories/announcements>`__.

Faire un don
------------

Si vous appréciez blunderDB et que vous voulez soutenir les développements passés et futurs, vous pouvez

* me payer un verre si on a le plaisir de se rencontrer!

* faire un petit don par PayPal à l'adresse blunderdb@proton.me

Remerciements
-------------

Je dédie ce petit logiciel à ma compagne Anne-Claire et notre tendre
fille Perrine. Je tiens à remercier tout particulièrement quelques amis:

* *Tristan Remille*, de m'avoir initié au backgammon avec joie et
  bienveillance; de montrer la Voie dans la compréhension de ce
  merveilleux jeu; de continuer à m'encourager malgré mes piètres
  tentatives de mieux jouer.

* *Nicolas Harmand*, joyeux camarade depuis maintenant plus d'une dizaine
  d'années dans de chouettes aventures, et un fantastique partenaire de jeu
  depuis qu'il a choppé le virus du backgammon.

Crédits
-------

blunderDB embarque du code, des données et des polices d'autres personnes.
L'inventaire complet, avec le texte des licences, est le fichier
``THIRD_PARTY.md`` livré avec chaque paquet (``/usr/share/doc/blunderdb/``)
et présent à la racine du `dépôt
<https://github.com/kevung/blunderDB/blob/main/THIRD_PARTY.md>`__.
L'essentiel :

* le réseau de neurones ``strehl-prob5-512-512-256-128`` est l'œuvre
  d'*Alexander Strehl* (`alexstrehl/backgammon-ai-engine
  <https://github.com/alexstrehl/backgammon-ai-engine>`__, licence MIT) ; la
  recherche, le modèle de videau et la table d'équité de match qui l'entourent
  forment la configuration propre de `gammonNet
  <https://github.com/kevung/gammonNet>`__ (MIT) ;

* la table d'équité de match Kazaross-XG2 est l'œuvre de *Neil Kazaross* ;

* les tables de take points et de valeurs de gammon sont tirées de *The Theory
  of Backgammon* de *Dirk Schiemann* ;

* les tables de bearoff unilatérale (6 points, 15 pions, pour l'EPC) et
  bilatérale (6 points, 6 pions, pour les verdicts de videau en course) sont
  calculées par blunderDB lui-même, par un portage de l'outil ``makebearoff``
  de `GNU Backgammon <https://www.gnu.org/software/gnubg/>`__ ; le résultat
  est identique octet pour octet à celui de gnubg, dont l'empreinte SHA-256
  sert de référence. GNUbg est un logiciel libre sous licence GPL ;

* les fichiers de match sont lus par `xgparser
  <https://github.com/kevung/xgparser>`__, `gnubgparser
  <https://github.com/kevung/gnubgparser>`__ et `bgfparser
  <https://github.com/kevung/bgfparser>`__ (MIT) ;

* côté Go : `modernc.org/sqlite <https://gitlab.com/cznic/sqlite>`__
  (BSD-3-Clause), `pgx <https://github.com/jackc/pgx>`__, `Wails
  <https://wails.io>`__ et `go-fsrs
  <https://github.com/open-spaced-repetition/go-fsrs>`__ (MIT) ;

* côté interface : `Svelte <https://svelte.dev>`__, `two.js
  <https://two.js.org>`__, `Chart.js <https://www.chartjs.org>`__ et
  `driver.js <https://driverjs.com>`__ (MIT) ;

* les polices `Nunito <https://github.com/googlefonts/nunito>`__ et `Noto Sans
  JP <https://fonts.google.com/noto/specimen/Noto+Sans+JP>`__ (SIL Open Font
  License 1.1).
