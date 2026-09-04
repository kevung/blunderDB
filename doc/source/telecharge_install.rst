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

Quel fichier choisir ?
----------------------

Chaque version publiée sur la `page des releases
<https://github.com/kevung/blunderDB/releases>`__ est accompagnée d'une
trentaine de fichiers. Le tableau ci-dessous indique lequel prendre selon
votre système ; remplacez ``x.y.z`` par le numéro de la version.

.. list-table::
   :header-rows: 1
   :widths: 24 34 42

   * - Système
     - Fichier
     - Installation
   * - Debian, Ubuntu, Linux Mint
     - ``blunderdb_x.y.z_amd64.deb``
     - ``sudo apt install ./blunderdb_x.y.z_amd64.deb``
   * - Fedora, openSUSE
     - ``blunderdb-x.y.z.x86_64.rpm``
     - ``sudo dnf install ./blunderdb-x.y.z.x86_64.rpm``
   * - Arch Linux, Manjaro
     - paquet AUR ``blunderdb-bin``
     - ``yay -S blunderdb-bin``
   * - Toute distribution Linux, avec Flatpak
     - ``blunderDB-x.y.z.flatpak``
     - ``flatpak install ./blunderDB-x.y.z.flatpak``
   * - Autre distribution Linux
     - ``blunderDB-linux-webkit2gtk-4.1-x.y.z.tar.gz``
     - ``tar xzf`` puis ``./blunderDB``
   * - Linux sur ARM 64 bits
     - ``blunderdb_x.y.z_arm64.deb``, ``blunderdb-x.y.z.aarch64.rpm`` ou
       ``blunderDB-linux-arm64-webkit2gtk-4.1-x.y.z.tar.gz``
     - Comme ci-dessus, avec le fichier ``arm64``/``aarch64``
   * - macOS (Intel et Apple Silicon)
     - ``blunderDB-macos-x.y.z.zip``
     - Décompresser, puis glisser ``blunderDB.app`` dans *Applications*
   * - Windows
     - ``blunderDB-windows-x.y.z.exe``
     - Exécuter directement, sans installation
   * - Serveur (mode ``serve``, Docker)
     - image ``ghcr.io/kevung/blunderdb-serve``
     - ``docker pull ghcr.io/kevung/blunderdb-serve:x.y.z``

Les fichiers ``amd64`` et ``x86_64`` sont pour un processeur Intel ou AMD,
c'est-à-dire la quasi-totalité des ordinateurs de bureau. Les fichiers
``arm64`` et ``aarch64`` sont pour un processeur ARM 64 bits : Raspberry Pi 4
et 5, Mac à puce Apple faisant tourner Linux, serveurs ARM. En cas de doute,
``uname -m`` répond : ``x86_64`` ou ``aarch64``. Le paquet AUR
``blunderdb-bin`` couvre les deux et choisit tout seul.

Les autres fichiers de la page sont les binaires Linux bruts (voir plus bas),
les manifestes winget et Homebrew (voir :ref:`install_winget_homebrew`), la
documentation en PDF dans neuf langues, et les empreintes ``.sha256``.

Vérifier un téléchargement
--------------------------

Chaque fichier publié est accompagné de son empreinte SHA-256, dans un fichier
du même nom suivi de ``.sha256``. Téléchargez les deux dans le même dossier et
vérifiez que le fichier reçu est bien celui qui a été publié :

.. code-block:: bash

   sha256sum -c blunderdb_x.y.z_amd64.deb.sha256        # Linux
   shasum -a 256 -c blunderDB-macos-x.y.z.zip.sha256   # macOS

.. code-block:: powershell

   Get-FileHash .\blunderDB-windows-x.y.z.exe -Algorithm SHA256   # Windows

Sous Linux et macOS, la commande affiche ``OK`` ; sous Windows, comparez la
valeur affichée avec le contenu du fichier ``.sha256``. Cette vérification est
la garantie disponible en l'absence de signature de code (voir
:numref:`annexe_windows_malware` et :numref:`annexe_mac_malware`).

Chaque fichier publié est également accompagné d'une attestation de
provenance (SLSA, via Sigstore), qui prouve qu'il a bien été produit par le
run de compilation officiel du dépôt GitHub, sans clé à gérer :

.. code-block:: bash

   gh attestation verify blunderdb_x.y.z_amd64.deb --repo kevung/blunderDB

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

Flatpak
~~~~~~~

Le bundle ``blunderDB-x.y.z.flatpak`` fonctionne sur toute distribution où
Flatpak est installé. Il repose sur l'environnement GNOME 47, que Flatpak
télécharge depuis Flathub s'il manque :

.. code-block:: bash

   flatpak install ./blunderDB-x.y.z.flatpak
   flatpak run io.github.kevung.blunderDB

blunderDB n'est pas encore publié sur Flathub : le bundle s'installe depuis le
fichier téléchargé, et se met à jour en installant celui de la version
suivante. L'application a accès au dossier personnel pour ouvrir et
enregistrer les bases. La ligne de commande (:ref:`cli`) s'obtient par
``flatpak run io.github.kevung.blunderDB <commande>``.

Archive .tar.gz
~~~~~~~~~~~~~~~~

Pour les autres distributions. L'extraction d'une archive conserve le bit
exécutable, aucun ``chmod`` n'est donc nécessaire :

.. code-block:: bash

   tar xzf blunderDB-linux-webkit2gtk-4.1-x.y.z.tar.gz
   cd blunderDB-linux-webkit2gtk-4.1-x.y.z
   ./blunderDB

.. note:: L'exécutable de l'archive s'appelle ``blunderDB``, avec les
   majuscules. Les paquets ``.deb``, ``.rpm``, AUR et Flatpak installent en
   plus le lien ``blunderdb`` en minuscules, le nom que la documentation de la
   ligne de commande (:ref:`cli`) utilise ; l'archive contient le même lien
   à côté de l'exécutable. Pour l'avoir dans votre ``PATH`` :
   ``ln -s "$PWD/blunderDB" ~/.local/bin/blunderdb``.

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

Image Docker (mode serveur)
---------------------------

Le mode serveur (:ref:`headless`) n'a pas besoin de l'application de bureau :
chaque version publie l'image ``ghcr.io/kevung/blunderdb-serve`` sur le
registre GitHub, pour ``linux/amd64`` et ``linux/arm64``, sous l'étiquette de
la version et sous ``latest`` :

.. code-block:: bash

   docker pull ghcr.io/kevung/blunderdb-serve:x.y.z

Le lancement, la configuration et l'avertissement sur le mandataire
d'authentification sont décrits dans :ref:`headless_docker_image`.

.. _install_winget_homebrew:

Gestionnaires de paquets Windows et Mac (à venir)
-------------------------------------------------

Sous Linux, blunderDB s'installe déjà par le gestionnaire de paquets (AUR,
``.deb``, ``.rpm``). L'équivalent pour Windows (`winget
<https://learn.microsoft.com/windows/package-manager/>`__) et pour Mac
(`Homebrew <https://brew.sh/>`__) est en préparation : **les manifestes sont
fournis avec chaque release** (``blunderDB-winget-manifests-x.y.z.zip`` et
``blunderdb-x.y.z.rb`` sur la page des releases), mais ils ne sont pas encore
soumis aux dépôts publics. Tant que ce n'est pas le cas, les commandes
ci-dessous ne fonctionnent pas ; elles décrivent l'installation prévue :

.. code-block:: powershell

   winget install KevinUnger.blunderDB        # Windows, une fois publié

.. code-block:: bash

   brew tap kevung/tap                        # Mac, une fois le tap créé
   brew install --cask blunderdb

Ni le paquet winget ni le cask ne changent la nature de l'exécutable : il
reste non signé, et les avertissements décrits ci-dessous s'appliquent tels
quels. Sur Mac, ``brew install --cask --no-quarantine blunderdb`` évite le
blocage de Gatekeeper au premier lancement.

Avertissements Windows et Mac
-----------------------------

.. warning:: Sous Windows, il est possible que ce dernier émette des réticences
   à exécuter blunderDB. Voir :numref:`annexe_windows_malware` pour comprendre
   pourquoi et contourner les éventuels blocages.

.. warning:: Sous Mac, il est possible que ce dernier émette des réticences
   à exécuter blunderDB. Voir :numref:`annexe_mac_malware` pour comprendre
   pourquoi et contourner les éventuels blocages.
