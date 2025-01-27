.. _index:

blunderDB
=========

blunderDB est un logiciel pour constituer des bases de données de positions de
backgammon. Sa force principale est de constituer un lieu unique où aggréger
les positions qu'un joueur a pu rencontrer (en ligne, en tournoi) et de pouvoir
réétudier ces positions en les filtrant selon différents filtres combinables
arbitrairement. blunderDB peut également être utilisé pour constituer des
catalogues de positions de référence.

La présente documentation est structurée de la manière suivante:

* la section **téléchargement et installation** explique comment se procurer et
  lancer blunderDB.

* le **manuel** décrit le fonctionnement général de blunderDB.

* le **guide utilisateur** est une introduction pratique pour utiliser
  rapidement blunderDB.

* la liste des **commandes** ainsi que la liste des **raccourcis**
  clavier permettent une utilisation efficace de blunderDB.

* la **FAQ** fournit quelques réponses aux interrogations les plus fréquentes.

Historique des versions
=======================

.. csv-table::
   :header: "Version", "Date", "Cause et/ou nature des évolutions"
   :widths: 5, 7, 20
   :align: center

   0.1.0, 31/12/2024, "Création version beta."
   0.2.0, 06/01/2025, "Résolutions de bugs divers. 
   
   Ajout de tables de matchs/TP/GV.

   Ajout de filtres de recherche (coups, décision de videau, date).
   
   Ajout de métadonnées sur les positions.

   Fonction d'import/export entre instances de blunderDB.

   Ajout de fonction de métadonnées sur les bases de données.
   
   Introduction des numéros de versions (base de données et application)."
   0.3.0, 27/01/2025, "Résolutions de bugs divers. 

   Sauvegarde automatiquement le dimensionnement de la fenêtre.

   Importe les éventuels commentaires depuis XG."

Sommaire
========

.. toctree::
   :maxdepth: 2
   :numbered:

   telecharge_install
   manuel
   guide_utilisateur
   cmd_mode
   raccourcis
   faq
   annexe_windows_securite

.. youtube:: HkY4iXjxMeI
   :width: 100%

.. raw:: html

   <div style="margin-top: 20px;"></div>

.. _contacts:

Contacts
========

Auteur: Kévin Unger <blunderdb@proton.me>.
Vous pouvez aussi me trouver sur Heroes sous le pseudo postmanpat.

J'ai développé blunderDB initialement pour mon usage personnel afin de
pouvoir détecter des motifs dans mes erreurs. Mais il est très agréable
d'avoir un retour surtout quand on a dépensé un paquet d'heures de
conception, codage, débuggage... Aussi n'hésitez pas à m'écrire pour
faire part de votre retour d'expérience. Tous les retours (constructifs)
sont bienvenus.

Voici plusieurs manières de réagir:

* discuter avec moi, si on se retrouve dans un tournoi,

* m'écrire un mail à blunderdb@proton.me,

* sur Github,

  * ouvrir un ticket: https://github.com/kevung/blunderDB/issues

  * pour des corrections de bugs ou des propositions d'amélioration,
    créer une pull request.

Faire un don
============

Si vous appréciez blunderDB et que vous voulez soutenir les développements passés et futurs, vous pouvez

* me payer un verre si on a le plaisir de se rencontrer!

* faire un petit don par PayPal à l'adresse blunderdb@proton.me

Remerciements
=============

Je dédie ce petit logiciel à ma compagne Anne-Claire et notre tendre
fille Perrine. Je tiens à remercier tout particulièrement quelques amis:

* *Tristan Remille*, de m'avoir initié au backgammon avec joie et
  bienveillance; de montrer la Voie dans la compréhension de ce
  merveilleux jeu; de continuer à m'encourager avec gentillesse et
  patience face à mes piètres tentatives de mieux jouer.

* *Nicolas Harmand*, joyeux camarade depuis maintenant plus d'une dizaine
  d'années dans de chouettes aventures, et un fantastique partenaire de jeu
  depuis qu'il a choppé le virus du backgammon.
