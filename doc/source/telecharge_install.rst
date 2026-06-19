.. _telecharge_install:

Téléchargement et installation
==============================

blunderDB se présente comme un unique exécutable ne nécessitant aucune installation.

La dernière version de blunderDB est disponible en licence MIT:

* pour Windows: |latest_windows_exe|

* pour Linux: |latest_linux_exe|

* pour Mac: |latest_mac_exe|

.. only:: html

   Pour une consultation hors ligne, vous pouvez également télécharger cette documentation au format PDF  : |latest_fr_pdf|

.. note:: blunderDB utilise Webview2 pour le rendu de l'interface graphique. Il
   y a de fortes chances que Webview2 soit déjà présent nativement sur votre
   système d'exploitation. Si ce n'est pas le cas, la première exécution de
   blunderDB proposera de le télécharger et de l'installer. Aucune manipulation
   de la part de l'utilisateur n'est attendue.

Installation sous Linux
-----------------------

Plusieurs formats sont proposés pour Linux. Les paquets et archives ci-dessous
**rendent blunderDB exécutable automatiquement** : contrairement au binaire brut
téléchargé via un navigateur, ils évitent d'avoir à lancer ``chmod +x`` à chaque
téléchargement ou mise à jour. Ils créent également une entrée dans le menu des
applications.

Paquets natifs (.deb / .rpm)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Méthode recommandée sur Debian, Ubuntu et Linux Mint (``.deb``) ainsi que sur
Fedora et openSUSE (``.rpm``). Le gestionnaire de paquets installe
automatiquement la dépendance webkit2gtk appropriée. Remplacez ``x.y.z`` par la
version téléchargée :

.. code-block:: bash

   sudo apt install ./blunderdb_x.y.z_amd64.deb     # Debian / Ubuntu / Mint
   sudo dnf install ./blunderdb-x.y.z.x86_64.rpm    # Fedora / openSUSE

Arch Linux (AUR)
~~~~~~~~~~~~~~~~~

Le paquet ``blunderdb-bin`` est disponible sur l'AUR et mis à jour
automatiquement par les assistants AUR :

.. code-block:: bash

   yay -S blunderdb-bin      # ou : paru -S blunderdb-bin

Archive .tar.gz
~~~~~~~~~~~~~~~~

Pour les autres distributions. L'extraction d'une archive conserve le bit
exécutable, aucun ``chmod`` n'est donc nécessaire :

.. code-block:: bash

   tar xzf blunderDB-linux-webkit2gtk-4.1-x.y.z.tar.gz
   cd blunderDB-linux-webkit2gtk-4.1-x.y.z
   ./blunderdb

Binaire brut (méthode avancée)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Le binaire brut |latest_linux_exe| reste disponible. Comme un navigateur retire
le bit exécutable au téléchargement, il faut le rétablir avant la première
exécution :

.. code-block:: bash

   chmod +x ./blunderDB-linux-x.y.z

.. note:: Deux variantes Linux sont publiées selon la version de la
   bibliothèque webkit2gtk. Si vous obtenez l'erreur
   ``libwebkit2gtk-4.0.so.37: cannot open shared object file``, votre
   distribution utilise webkit2gtk-4.1 : utilisez le paquet ``.deb``/``.rpm``,
   le paquet AUR, ou téléchargez la version dédiée
   |latest_linux_webkit2gtk41_exe|. Les paquets natifs choisissent
   automatiquement la bonne dépendance.

Avertissements Windows et Mac
-----------------------------

.. warning:: Sous Windows, il est possible que ce dernier émette des réticences
   à exécuter blunderDB. Voir :numref:`annexe_windows_malware` pour comprendre
   pourquoi et contourner les éventuels blocages.

.. warning:: Sous Mac, il est possible que ce dernier émette des réticences
   à exécuter blunderDB. Voir :numref:`annexe_mac_malware` pour comprendre
   pourquoi et contourner les éventuels blocages.
