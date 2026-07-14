// Contenu d'aide français (traduction de en.js).
// Chaque valeur est le HTML interne verbatim de l'onglet correspondant de HelpModal.
export default {
    manual: `
                    <h3>Introduction</h3>
                    <p>
                        blunderDB est un logiciel permettant de créer des bases de données de positions de backgammon. Sa principale force est d'offrir un endroit unique où regrouper les positions qu'un joueur a rencontrées (en ligne,
                        en tournoi) et de pouvoir réétudier ces positions en les filtrant selon divers filtres combinables de façon arbitraire. blunderDB peut aussi être utilisé pour créer des catalogues
                        de positions de référence.
                    </p>
                    <p>Les positions sont stockées dans une base de données représentée par un fichier .db.</p>

                    <h3>Interactions principales</h3>
                    <p>Les principales interactions possibles avec blunderDB sont :</p>
                    <ul>
                        <li>ajouter une nouvelle position,</li>
                        <li>modifier une position existante,</li>
                        <li>copier le plateau sous forme d'image PNG dans le presse-papiers (<strong>Ctrl+X</strong>), ou le plateau avec son analyse (<strong>Ctrl+X, Ctrl+X</strong>),</li>
                        <li>supprimer une position existante,</li>
                        <li>rechercher une ou plusieurs positions,</li>
                        <li>importer des matchs depuis diverses sources (XG, GNUbg, BGBlitz, Jellyfish), y compris les commentaires des fichiers XG,</li>
                        <li>parcourir les coups d'un match importé,</li>
                        <li>organiser les positions en collections,</li>
                        <li>organiser les matchs en tournois.</li>
                    </ul>
                    <p>L'utilisateur peut librement étiqueter les positions et les annoter avec des commentaires.</p>

                    <h3>Description de l'interface</h3>
                    <p>L'interface de blunderDB est structurée de haut en bas de la manière suivante :</p>
                    <ul>
                        <li>[en haut] la barre d'outils, qui rassemble toutes les opérations principales pouvant être effectuées sur la base de données,</li>
                        <li>[au milieu] la zone d'affichage principale, qui permet d'afficher ou d'éditer des positions de backgammon,</li>
                        <li>[en bas] la barre d'état, qui intègre la ligne de commande et présente diverses informations sur la position courante.</li>
                    </ul>
                    <p>Des panneaux peuvent être affichés pour :</p>
                    <ul>
                        <li>afficher les données d'analyse associées à la position courante (de XG, GNUbg ou BGBlitz),</li>
                        <li>afficher, ajouter ou modifier des commentaires,</li>
                        <li>parcourir les matchs importés et naviguer dans leurs coups (panneau Match),</li>
                        <li>gérer des collections de positions (panneau Collection),</li>
                        <li>étudier des positions par répétition espacée (panneau Anki),</li>
                        <li>gérer des tournois (panneau Tournoi),</li>
                        <li>afficher des statistiques de performance (panneau Stats),</li>
                        <li>calculer les valeurs d'EPC pour les positions de sortie (panneau EPC),</li>
                        <li>consulter les filtres de recherche enregistrés (panneau Bibliothèque de filtres),</li>
                        <li>consulter l'historique des recherches (panneau Historique des recherches),</li>
                        <li>voir les journaux d'opérations (panneau Log).</li>
                    </ul>
                    <p>La zone d'affichage principale fournit à l'utilisateur :</p>
                    <ul>
                        <li>un plateau pour afficher ou éditer une position de backgammon,</li>
                        <li>le niveau et le propriétaire du videau,</li>
                        <li>le pip count de chaque joueur,</li>
                        <li>le score de chaque joueur,</li>
                        <li>les dés à jouer. Si aucune valeur n'est affichée sur les dés, la position des dés indique quel joueur a le trait et que la position est une décision de videau.</li>
                    </ul>
                    <p>La barre d'état affiche de gauche à droite :</p>
                    <ul>
                        <li>la ligne de commande (appuyez sur <strong>Espace</strong> pour l'ouvrir),</li>
                        <li>un message d'information lié à la dernière opération effectuée,</li>
                        <li>l'indice de la position courante, suivi du nombre total de positions (ou des informations de coup/partie lors de la navigation dans un match).</li>
                    </ul>
                    <p>Dans le cas de positions issues d'une recherche de l'utilisateur, le nombre de positions indiqué dans la barre d'état correspond au nombre de positions filtrées.</p>

                    <h3>Parcourir les positions</h3>
                    <p>Par défaut, blunderDB vous permet de :</p>
                    <ul>
                        <li>faire défiler les différentes positions de la bibliothèque courante,</li>
                        <li>afficher les informations d'analyse associées à une position,</li>
                        <li>afficher, ajouter et modifier des commentaires sur une position.</li>
                    </ul>

                    <h3>Éditer des positions</h3>
                    <p>
                        Appuyer sur la touche <strong>Tab</strong> ouvre le panneau de recherche et permet d'éditer une position sur le plateau afin de l'ajouter à la base de données ou de définir une structure de position à rechercher.
                        La répartition des pions, le videau, le score et le trait peuvent être modifiés à l'aide de la souris.
                    </p>

                    <h3>Ligne de commande</h3>
                    <p>
                        La ligne de commande, intégrée dans la barre d'état, permet d'exécuter toutes les fonctionnalités de blunderDB : opérations sur la base de données, navigation dans les positions, affichage des analyses et
                        des commentaires, recherche de positions avec filtres... Après vous être familiarisé avec l'interface, il est recommandé d'utiliser progressivement la ligne de commande, qui permet une utilisation puissante et
                        fluide de blunderDB, en particulier pour les fonctionnalités de recherche de positions.
                    </p>
                    <p>
                        Pour ouvrir la ligne de commande, appuyez sur la touche <strong>Espace</strong>. Une invite apparaît dans la barre d'état. Tapez votre commande et appuyez sur <strong>Entrée</strong> pour l'exécuter. Appuyez sur
                        <strong>Échap</strong>
                        pour annuler. L'historique des commandes et les résultats sont consignés dans le panneau <strong>Log</strong>.
                    </p>
                    <p>
                        blunderDB exécute les requêtes envoyées par l'utilisateur dès lors qu'elles sont valides et modifie immédiatement l'état de la base de données si nécessaire. Aucune action d'enregistrement explicite
                        n'est requise de la part de l'utilisateur.
                    </p>
                    <p>
                        Pour affiner une recherche au sein de positions préalablement filtrées, utilisez la commande <strong>ss</strong> suivie de filtres (par ex. <strong>ss nc</strong>). Cela restreint la recherche aux
                        seules positions actuellement affichées, ce qui permet de réduire progressivement les résultats. Le panneau de recherche (<strong>Ctrl+F</strong>) propose aussi une case « Rechercher dans les résultats courants »
                        pour la même fonctionnalité.
                    </p>

                    <h3>Calculatrice EPC</h3>
                    <p>La calculatrice EPC (Effective Pip Count) calcule le pip count effectif des positions de sortie. Elle utilise la base de données de sortie unilatérale à 6 points de GNUbg pour obtenir des valeurs d'EPC exactes.</p>
                    <p>
                        Pour ouvrir le panneau EPC, appuyez sur <strong>Ctrl+E</strong>, cliquez sur l'onglet EPC du panneau inférieur ou tapez <strong>epc</strong> dans la ligne de commande. Le plateau est initialisé avec une configuration standard
                        de sortie (15 pions).
                    </p>
                    <p>
                        Vous pouvez librement ajouter ou retirer des pions sur les flèches du jan intérieur à l'aide de la souris. Les valeurs d'EPC s'affichent en temps réel dans le panneau EPC dédié, indiquant pour chaque joueur :
                    </p>
                    <ul>
                        <li><strong>EPC</strong> : le nombre moyen de pips nécessaires pour sortir tous les pions,</li>
                        <li><strong>Pip Count</strong> : le pip count brut,</li>
                        <li><strong>Wastage</strong> : la différence entre l'EPC et le pip count,</li>
                        <li><strong>Avg Rolls</strong> : nombre moyen de jets pour sortir tous les pions,</li>
                        <li><strong>Std Dev</strong> : écart-type du nombre de jets.</li>
                    </ul>
                    <p>Lorsque les deux joueurs ont des pions dans leur jan intérieur, une section de comparaison affiche les différences d'EPC et de pip count.</p>
                    <p>Pour fermer le panneau EPC, appuyez de nouveau sur <strong>Ctrl+E</strong> ou passez à un autre onglet.</p>

                    <h3>Navigation dans les matchs</h3>
                    <p>
                        blunderDB permet de parcourir les coups des matchs importés. Ouvrez le panneau Match avec <strong>Ctrl+Tab</strong> et double-cliquez sur un match (ou appuyez sur <strong>Entrée</strong>)
                        pour charger ses positions.
                    </p>
                    <p>
                        Lors de la navigation dans un match, la dernière position visitée est automatiquement enregistrée et restaurée. Utilisez les touches <strong>Gauche</strong>/<strong>Droite</strong> pour vous déplacer entre les positions, et
                        <strong>PageUp</strong>/<strong>PageDown</strong> pour sauter d'une partie à l'autre.
                    </p>
                    <p>
                        Le panneau d'analyse (<strong>Ctrl+L</strong>) affiche l'analyse de chaque coup, le coup joué étant mis en évidence. Appuyez sur <strong>d</strong> pour basculer entre l'analyse de pions et l'analyse de videau.
                    </p>

                    <h3>Collections</h3>
                    <p>
                        Les collections permettent d'organiser les positions en groupes personnalisés. Ouvrez le panneau Collection avec <strong>Ctrl+B</strong>, puis double-cliquez sur une collection pour parcourir ses positions.
                        Les collections et les positions qu'elles contiennent peuvent être réordonnées par glisser-déposer.
                    </p>

                    <h3>Anki (répétition espacée)</h3>
                    <p>Le panneau Anki (<strong>Ctrl+K</strong>) offre la répétition espacée pour étudier des positions de backgammon à l'aide de l'algorithme FSRS.</p>
                    <p>
                        <strong>Créer des paquets :</strong> Cliquez sur <em>Nouveau paquet</em> pour créer un paquet à partir d'une collection ou des résultats de recherche courants. Les paquets basés sur une recherche se synchronisent automatiquement lorsque l'onglet Anki
                        est activé.
                    </p>
                    <p>
                        <strong>Réviser :</strong> Sélectionnez un paquet et cliquez sur <em>Étudier</em> (ou double-cliquez sur un paquet) pour commencer à réviser les cartes à échéance. Chaque carte affiche la position correspondante sur le
                        plateau. Notez votre rappel avec les touches <strong>1</strong> (À revoir), <strong>2</strong> (Difficile), <strong>3</strong> (Correct) ou <strong>4</strong> (Facile). Appuyez sur <strong>Échap</strong> pour arrêter
                        et revenir à la liste des paquets.
                    </p>
                    <p>
                        <strong>Arrêter/Reprendre :</strong> Vous pouvez arrêter une session de révision à tout moment en appuyant sur <strong>Échap</strong>. Le bouton devient <em>Reprendre</em> et affiche votre progression. Cliquez dessus pour
                        continuer là où vous vous êtes arrêté.
                    </p>
                    <p>
                        <strong>Gestion des paquets :</strong> Utilisez les boutons d'action pour renommer, synchroniser, réinitialiser ou supprimer des paquets. Les paramètres FSRS (rétention cible, intervalle maximal, fuzz) peuvent être configurés par paquet
                        dans les Réglages (icône d'engrenage).
                    </p>

                    <h3>Tournois</h3>
                    <p>Les tournois permettent de regrouper les matchs par événement. Ouvrez le panneau Tournoi avec <strong>Ctrl+Y</strong> pour gérer les tournois et leur affecter des matchs.</p>

                    <h3>Stats</h3>
                    <p>
                        Le panneau Stats (<strong>Ctrl+D</strong>) affiche des statistiques de performance (PR et coût en MWC) calculées à partir de toutes les positions importées. Utilisez la barre de filtres pour restreindre l'analyse par
                        joueur, tournoi, plage de dates, type de décision ou longueur de match. Cliquez sur n'importe quel indicateur pour explorer en détail les positions correspondantes.
                    </p>
`,
    shortcuts: `
                    <h3>Base de données</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + N</td>
                                <td>Nouvelle base de données</td>
                            </tr>

                            <tr>
                                <td>Ctrl + O</td>
                                <td>Ouvrir une base de données</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + I</td>
                                <td>Importer une base de données</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + S</td>
                                <td>Exporter une base de données</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Q</td>
                                <td>Quitter blunderDB</td>
                            </tr>

                            <tr>
                                <td>Ctrl + M</td>
                                <td>Éditer les métadonnées</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Position</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + I</td>
                                <td>Importer une position ou un match</td>
                            </tr>

                            <tr>
                                <td>Ctrl + C</td>
                                <td>Copier la position (copie aussi dans le presse-papiers du plateau)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X</td>
                                <td>Copier l'image du plateau dans le presse-papiers (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X, Ctrl + X</td>
                                <td>Copier l'image du plateau + analyse dans le presse-papiers (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + V</td>
                                <td>Coller la position (dans le panneau de recherche : coller sur le plateau)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + S</td>
                                <td>Enregistrer la position</td>
                            </tr>

                            <tr>
                                <td>Ctrl + U</td>
                                <td>Mettre à jour la position</td>
                            </tr>

                            <tr>
                                <td>Del</td>
                                <td>Supprimer la position</td>
                            </tr>

                            <tr>
                                <td>Backspace</td>
                                <td>Réinitialiser le plateau, le videau, le score et les dés</td>
                            </tr>

                            <tr>
                                <td>Ctrl + G</td>
                                <td>Afficher les métadonnées de la position</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Navigation</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + R</td>
                                <td>Recharger toutes les positions</td>
                            </tr>

                            <tr>
                                <td>PageUp, h</td>
                                <td>Première position / Partie précédente (navigation dans un match)</td>
                            </tr>

                            <tr>
                                <td>Left, k</td>
                                <td>Position précédente</td>
                            </tr>

                            <tr>
                                <td>Right, j</td>
                                <td>Position suivante</td>
                            </tr>

                            <tr>
                                <td>PageDown, l</td>
                                <td>Dernière position / Partie suivante (navigation dans un match)</td>
                            </tr>

                            <tr>
                                <td>r</td>
                                <td>Charger une position aléatoire</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Affichage</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + ArrowLeft</td>
                                <td>Orienter le plateau vers la gauche</td>
                            </tr>

                            <tr>
                                <td>Ctrl + ArrowRight</td>
                                <td>Orienter le plateau vers la droite</td>
                            </tr>

                            <tr>
                                <td>p</td>
                                <td>Afficher/masquer le pipcount</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Actions</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Tab</td>
                                <td>Ouvrir le panneau de recherche (éditeur de position)</td>
                            </tr>

                            <tr>
                                <td>Space</td>
                                <td>Ouvrir la ligne de commande</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Outils</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + L</td>
                                <td>Afficher l'analyse</td>
                            </tr>

                            <tr>
                                <td>Ctrl + P</td>
                                <td>Écrire des commentaires</td>
                            </tr>

                            <tr>
                                <td>Ctrl + K</td>
                                <td>Afficher le panneau Anki</td>
                            </tr>

                            <tr>
                                <td>Ctrl + F</td>
                                <td>Panneau de recherche</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Tab</td>
                                <td>Panneau Match</td>
                            </tr>

                            <tr>
                                <td>Ctrl + B</td>
                                <td>Panneau Collection</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Y</td>
                                <td>Panneau Tournois</td>
                            </tr>

                            <tr>
                                <td>Ctrl + D</td>
                                <td>Panneau Stats</td>
                            </tr>

                            <tr>
                                <td>Ctrl + E</td>
                                <td>Panneau EPC</td>
                            </tr>

                            <tr>
                                <td>?</td>
                                <td>Ouvrir l'aide</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Ligne de commande</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Up</td>
                                <td>Parcourir l'historique des commandes vers le haut</td>
                            </tr>
                            <tr>
                                <td>Down</td>
                                <td>Parcourir l'historique des commandes vers le bas</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panneau d'analyse</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Sélectionner/désélectionner un coup (afficher/masquer les flèches)</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Sélectionner le coup précédent (quand un coup est sélectionné)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Sélectionner le coup suivant (quand un coup est sélectionné)</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Basculer entre l'analyse de pions et de videau (navigation dans un match uniquement)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Désélectionner le coup. Si aucun coup n'est sélectionné, fermer le panneau.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panneau Historique des recherches</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Sélectionner/désélectionner une recherche (afficher la position)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Exécuter la recherche et fermer le panneau</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Sélectionner la recherche précédente (plus récente, au-dessus)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Sélectionner la recherche suivante (plus ancienne, en dessous)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Désélectionner la recherche. Si aucune recherche n'est sélectionnée, fermer le panneau.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panneau Bibliothèque de filtres</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Sélectionner/désélectionner un filtre (afficher la position)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Exécuter la recherche du filtre</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Sélectionner le filtre précédent (quand un filtre est sélectionné)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Sélectionner le filtre suivant (quand un filtre est sélectionné)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Désélectionner le filtre. Si aucun filtre n'est sélectionné, fermer le panneau.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panneau Match</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Sélectionner/désélectionner un match</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Charger les positions du match</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Sélectionner le match précédent</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Sélectionner le match suivant</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Supprimer le match sélectionné</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Désélectionner le match. Si aucun match n'est sélectionné, fermer le panneau.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panneau Collection</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Sélectionner une collection (afficher ses positions)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Ouvrir la collection et parcourir ses positions</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Retirer la position courante de la collection active</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Désélectionner la collection. Si aucune collection n'est sélectionnée, fermer le panneau.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panneau Anki</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Raccourci</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>1</td>
                                <td>Noter : À revoir (révision ratée, à réafficher bientôt)</td>
                            </tr>
                            <tr>
                                <td>2</td>
                                <td>Noter : Difficile (rappel difficile)</td>
                            </tr>
                            <tr>
                                <td>3</td>
                                <td>Noter : Correct (rappel correct)</td>
                            </tr>
                            <tr>
                                <td>4</td>
                                <td>Noter : Facile (rappel sans effort)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Arrêter la révision et revenir à la liste des paquets (reprise possible plus tard)</td>
                            </tr>
                        </tbody>
                    </table>
`,
    commands: `
                    <h3>Base de données</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Commande</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>new, ne, n</td>
                                <td>Créer une nouvelle base de données</td>
                            </tr>
                            <tr>
                                <td>open, op, o</td>
                                <td>Ouvrir une base de données existante</td>
                            </tr>
                            <tr>
                                <td>import_db, idb</td>
                                <td>Importer et fusionner une autre base de données</td>
                            </tr>
                            <tr>
                                <td>export_db, edb</td>
                                <td>Exporter la sélection courante vers une nouvelle base de données</td>
                            </tr>
                            <tr>
                                <td>quit, q</td>
                                <td>Quitter blunderDB</td>
                            </tr>
                            <tr>
                                <td>meta</td>
                                <td>Éditer les métadonnées</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Position</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Commande</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>import, i</td>
                                <td>Importer une position ou un match</td>
                            </tr>
                            <tr>
                                <td>write, wr, w</td>
                                <td>Enregistrer une position</td>
                            </tr>
                            <tr>
                                <td>write!, wr!, w!</td>
                                <td>Mettre à jour une position</td>
                            </tr>
                            <tr>
                                <td>delete, del, d</td>
                                <td>Supprimer une position</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Navigation</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Commande</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>[number]</td>
                                <td>Aller à une position précise par son indice</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Outils</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Commande</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>list, l</td>
                                <td>Afficher l'analyse</td>
                            </tr>
                            <tr>
                                <td>comment, co</td>
                                <td>Écrire des commentaires</td>
                            </tr>
                            <tr>
                                <td>filter, fl</td>
                                <td>Afficher la bibliothèque de filtres</td>
                            </tr>
                            <tr>
                                <td>history, hi</td>
                                <td>Afficher l'historique des recherches</td>
                            </tr>
                            <tr>
                                <td>match, ma</td>
                                <td>Afficher le panneau Match</td>
                            </tr>
                            <tr>
                                <td>collection, coll</td>
                                <td>Afficher le panneau des collections</td>
                            </tr>
                            <tr>
                                <td>epc</td>
                                <td>Calculatrice EPC (Effective Pip Count)</td>
                            </tr>
                            <tr>
                                <td>m</td>
                                <td>Naviguer dans le dernier match visité</td>
                            </tr>
                            <tr>
                                <td>help, he, h</td>
                                <td>Ouvrir l'aide</td>
                            </tr>
                            <tr>
                                <td>met</td>
                                <td>Ouvrir la table de match equity Kazaross-XG2</td>
                            </tr>
                            <tr>
                                <td>tp2</td>
                                <td>Take point avec videau à 2 (Live et Last)</td>
                            </tr>
                            <tr>
                                <td>tp2_live</td>
                                <td>Take point avec videau à 2 dans les longues courses</td>
                            </tr>
                            <tr>
                                <td>tp2_last</td>
                                <td>Take point avec videau à 2 dans les positions de dernier jet</td>
                            </tr>
                            <tr>
                                <td>tp4</td>
                                <td>Take point avec videau à 4 (Live et Last)</td>
                            </tr>
                            <tr>
                                <td>tp4_live</td>
                                <td>Take point avec videau à 4 dans les longues courses</td>
                            </tr>
                            <tr>
                                <td>tp4_last</td>
                                <td>Take point avec videau à 4 dans les positions de dernier jet</td>
                            </tr>
                            <tr>
                                <td>gv1</td>
                                <td>Valeurs de gammon avec videau à 1</td>
                            </tr>
                            <tr>
                                <td>gv2</td>
                                <td>Valeurs de gammon avec videau à 2</td>
                            </tr>
                            <tr>
                                <td>gv4</td>
                                <td>Valeurs de gammon avec videau à 4</td>
                            </tr>
                            <tr>
                                <td>#tag1 tag2 ...</td>
                                <td>Étiqueter la position</td>
                            </tr>
                            <tr>
                                <td>s</td>
                                <td>Rechercher des positions avec filtres</td>
                            </tr>
                            <tr>
                                <td>ss</td>
                                <td>Rechercher dans les résultats courants avec filtres</td>
                            </tr>
                            <tr>
                                <td>e</td>
                                <td>Recharger toutes les positions</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Filtres</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Filtre</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>cube, cub, cu, c</td>
                                <td>Inclure le videau</td>
                            </tr>
                            <tr>
                                <td>score, sco, sc, s</td>
                                <td>Inclure le score</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Inclure le type de décision</td>
                            </tr>
                            <tr>
                                <td>D</td>
                                <td>Inclure le jet de dés</td>
                            </tr>
                            <tr>
                                <td>nc</td>
                                <td>Inclure les positions sans contact</td>
                            </tr>
                            <tr>
                                <td>M</td>
                                <td>Inclure les positions miroir</td>
                            </tr>
                            <tr>
                                <td>i</td>
                                <td>Uniquement les positions importées seules, non apportées par l'import d'un match</td>
                            </tr>
                            <tr>
                                <td>x</td>
                                <td
                                    >Exclure les positions qui contiennent <em>n'importe quel</em> pion de la structure dessinée dans l'onglet « Except » (par ex. dessiner des pions sur 1, 3 et 5 ne conserve que les positions qui n'en ont aucun
                                    d'entre eux). Basculez « At least » / « Except » au-dessus des filtres pour dessiner les pions exclus sur le plateau (indiqués par un repère rouge). Le décompte par flèche n'est pas limité (3 sur une flèche
                                    exclut 3 ou plus à cet endroit — une flèche faite sans pion en réserve), et deux clics rapides sur une flèche la marquent comme devant être vide (une case hachurée rouge, de n'importe quelle couleur) ; un simple clic sur cette
                                    flèche la débloque. Sur une flèche partagée, « Except » l'emporte sur « At least » lorsque les deux se contredisent.</td
                                >
                            </tr>
                            <tr>
                                <td>p>x</td>
                                <td>Pip count &gt; x</td>
                            </tr>
                            <tr>
                                <td>p&lt;x</td>
                                <td>Pip count &lt; x</td>
                            </tr>
                            <tr>
                                <td>px,y</td>
                                <td>Pip count entre x et y</td>
                            </tr>
                            <tr>
                                <td>P>x</td>
                                <td>Pip count absolu du joueur &gt; x</td>
                            </tr>
                            <tr>
                                <td>P&lt;x</td>
                                <td>Pip count absolu du joueur &lt; x</td>
                            </tr>
                            <tr>
                                <td>Px,y</td>
                                <td>Pip count absolu du joueur entre x et y</td>
                            </tr>
                            <tr>
                                <td>e>x</td>
                                <td>Equity &gt; x (en millipoints)</td>
                            </tr>
                            <tr>
                                <td>e&lt;x</td>
                                <td>Equity &lt; x (en millipoints)</td>
                            </tr>
                            <tr>
                                <td>ex,y</td>
                                <td>Equity entre x et y (en millipoints)</td>
                            </tr>
                            <tr>
                                <td>E>x</td>
                                <td>Erreur de coup du joueur 1 &gt; x (en millipoints)</td>
                            </tr>
                            <tr>
                                <td>E&lt;x</td>
                                <td>Erreur de coup du joueur 1 &lt; x (en millipoints)</td>
                            </tr>
                            <tr>
                                <td>Ex,y</td>
                                <td>Erreur de coup du joueur 1 entre x et y (en millipoints)</td>
                            </tr>
                            <tr>
                                <td>w>x</td>
                                <td>Taux de victoire &gt; x</td>
                            </tr>
                            <tr>
                                <td>w&lt;x</td>
                                <td>Taux de victoire &lt; x</td>
                            </tr>
                            <tr>
                                <td>wx,y</td>
                                <td>Taux de victoire entre x et y</td>
                            </tr>
                            <tr>
                                <td>g>x</td>
                                <td>Taux de gammon &gt; x</td>
                            </tr>
                            <tr>
                                <td>g&lt;x</td>
                                <td>Taux de gammon &lt; x</td>
                            </tr>
                            <tr>
                                <td>gx,y</td>
                                <td>Taux de gammon entre x et y</td>
                            </tr>
                            <tr>
                                <td>b>x</td>
                                <td>Taux de backgammon &gt; x</td>
                            </tr>
                            <tr>
                                <td>b&lt;x</td>
                                <td>Taux de backgammon &lt; x</td>
                            </tr>
                            <tr>
                                <td>bx,y</td>
                                <td>Taux de backgammon entre x et y</td>
                            </tr>
                            <tr>
                                <td>W>x</td>
                                <td>Taux de victoire de l'adversaire &gt; x</td>
                            </tr>
                            <tr>
                                <td>W&lt;x</td>
                                <td>Taux de victoire de l'adversaire &lt; x</td>
                            </tr>
                            <tr>
                                <td>Wx,y</td>
                                <td>Taux de victoire de l'adversaire entre x et y</td>
                            </tr>
                            <tr>
                                <td>G>x</td>
                                <td>Taux de gammon de l'adversaire &gt; x</td>
                            </tr>
                            <tr>
                                <td>G&lt;x</td>
                                <td>Taux de gammon de l'adversaire &lt; x</td>
                            </tr>
                            <tr>
                                <td>Gx,y</td>
                                <td>Taux de gammon de l'adversaire entre x et y</td>
                            </tr>
                            <tr>
                                <td>B>x</td>
                                <td>Taux de backgammon de l'adversaire &gt; x</td>
                            </tr>
                            <tr>
                                <td>B&lt;x</td>
                                <td>Taux de backgammon de l'adversaire &lt; x</td>
                            </tr>
                            <tr>
                                <td>Bx,y</td>
                                <td>Taux de backgammon de l'adversaire entre x et y</td>
                            </tr>
                            <tr>
                                <td>o>x</td>
                                <td>Pions sortis du joueur &gt; x</td>
                            </tr>
                            <tr>
                                <td>o&lt;x</td>
                                <td>Pions sortis du joueur &lt; x</td>
                            </tr>
                            <tr>
                                <td>ox,y</td>
                                <td>Pions sortis du joueur entre x et y</td>
                            </tr>
                            <tr>
                                <td>O>x</td>
                                <td>Pions sortis de l'adversaire &gt; x</td>
                            </tr>
                            <tr>
                                <td>O&lt;x</td>
                                <td>Pions sortis de l'adversaire &lt; x</td>
                            </tr>
                            <tr>
                                <td>Ox,y</td>
                                <td>Pions sortis de l'adversaire entre x et y</td>
                            </tr>
                            <tr>
                                <td>k>x</td>
                                <td>Pions arrière du joueur &gt; x</td>
                            </tr>
                            <tr>
                                <td>k&lt;x</td>
                                <td>Pions arrière du joueur &lt; x</td>
                            </tr>
                            <tr>
                                <td>kx,y</td>
                                <td>Pions arrière du joueur entre x et y</td>
                            </tr>
                            <tr>
                                <td>K>x</td>
                                <td>Pions arrière de l'adversaire &gt; x</td>
                            </tr>
                            <tr>
                                <td>K&lt;x</td>
                                <td>Pions arrière de l'adversaire &lt; x</td>
                            </tr>
                            <tr>
                                <td>Kx,y</td>
                                <td>Pions arrière de l'adversaire entre x et y</td>
                            </tr>
                            <tr>
                                <td>z>x</td>
                                <td>Pions du joueur dans la zone &gt; x</td>
                            </tr>
                            <tr>
                                <td>z&lt;x</td>
                                <td>Pions du joueur dans la zone &lt; x</td>
                            </tr>
                            <tr>
                                <td>zx,y</td>
                                <td>Pions du joueur dans la zone entre x et y</td>
                            </tr>
                            <tr>
                                <td>Z>x</td>
                                <td>Pions de l'adversaire dans la zone &gt; x</td>
                            </tr>
                            <tr>
                                <td>Z&lt;x</td>
                                <td>Pions de l'adversaire dans la zone &lt; x</td>
                            </tr>
                            <tr>
                                <td>Zx,y</td>
                                <td>Pions de l'adversaire dans la zone entre x et y</td>
                            </tr>
                            <tr>
                                <td>bo>x</td>
                                <td>Blot du joueur dans l'outfield &gt; x</td>
                            </tr>
                            <tr>
                                <td>bo&lt;x</td>
                                <td>Blot du joueur dans l'outfield &lt; x</td>
                            </tr>
                            <tr>
                                <td>box,y</td>
                                <td>Blot du joueur dans l'outfield entre x et y</td>
                            </tr>
                            <tr>
                                <td>BO>x</td>
                                <td>Blot de l'adversaire dans l'outfield &gt; x</td>
                            </tr>
                            <tr>
                                <td>BO&lt;x</td>
                                <td>Blot de l'adversaire dans l'outfield &lt; x</td>
                            </tr>
                            <tr>
                                <td>BOx,y</td>
                                <td>Blot de l'adversaire dans l'outfield entre x et y</td>
                            </tr>
                            <tr>
                                <td>bj&gt;x</td>
                                <td>Blot Jan du joueur &gt; x</td>
                            </tr>
                            <tr>
                                <td>bj&lt;x</td>
                                <td>Blot Jan du joueur &lt; x</td>
                            </tr>
                            <tr>
                                <td>bjx,y</td>
                                <td>Blot Jan du joueur entre x et y</td>
                            </tr>
                            <tr>
                                <td>BJ&gt;x</td>
                                <td>Blot Jan de l'adversaire &gt; x</td>
                            </tr>
                            <tr>
                                <td>BJ&lt;x</td>
                                <td>Blot Jan de l'adversaire &lt; x</td>
                            </tr>
                            <tr>
                                <td>BJx,y</td>
                                <td>Blot Jan de l'adversaire entre x et y</td>
                            </tr>

                            <tr>
                                <td>t"word1;word2;..."</td>
                                <td>Rechercher du texte</td>
                            </tr>
                            <tr>
                                <td>m"pattern1;pattern2;..."</td>
                                <td>Meilleurs coups contenant au moins un des motifs indiqués</td>
                            </tr>
                            <tr>
                                <td>m"ND;DT;DP;..."</td>
                                <td>Meilleure décision de videau parmi No Double/Take, Double/Take, Double/Pass</td>
                            </tr>
                            <tr>
                                <td>T&gt;x</td>
                                <td>Date de création &gt; x (année/mois/jour)</td>
                            </tr>
                            <tr>
                                <td>T&lt;x</td>
                                <td>Date de création &lt; x (année/mois/jour)</td>
                            </tr>
                            <tr>
                                <td>Tx,y</td>
                                <td>Date de création entre x et y</td>
                            </tr>
                            <tr>
                                <td>max</td>
                                <td>Rechercher dans le match d'ID x (par ex. ma3)</td>
                            </tr>
                            <tr>
                                <td>max,y</td>
                                <td>Rechercher dans les matchs d'ID x à y (par ex. ma2,5)</td>
                            </tr>
                            <tr>
                                <td>tnx</td>
                                <td>Rechercher dans le tournoi d'ID x (par ex. tn1)</td>
                            </tr>
                            <tr>
                                <td>tnx,y</td>
                                <td>Rechercher dans les tournois d'ID x à y (par ex. tn1,3)</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Divers</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Commande</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>clear, cl</td>
                                <td>Effacer l'historique des commandes</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_0_to_1_1</td>
                                <td>Migrer la base de données de la version 1.0 vers la 1.1</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_1_to_1_2</td>
                                <td>Migrer la base de données de la version 1.1 vers la 1.2</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_2_to_1_3</td>
                                <td>Migrer la base de données de la version 1.2 vers la 1.3</td>
                            </tr>
                        </tbody>
                    </table>
`,
    about: `
                    <h3>Version</h3>
                    <p>Version de l'application : {appVersion}</p>
                    <p>Version de la base de données : {dbVersion}</p>

                    <h3>Auteur</h3>
                    <p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
                    <p>Vous pouvez aussi me retrouver sur Heroes sous le pseudo <strong>postmanpat</strong>.</p>
                    <p>
                        J'ai développé blunderDB au départ pour mon usage personnel, afin de détecter des schémas dans mes erreurs. Mais il est très agréable de recevoir des retours, surtout lorsqu'on a passé beaucoup d'heures
                        sur la conception, le code, le débogage... Alors n'hésitez pas à m'écrire pour partager vos retours.
                    </p>
                    <p>Voici plusieurs façons de me contacter :</p>
                    <ul>
                        <li>Discutez avec moi si nous nous croisons en tournoi,</li>
                        <li>Envoyez-moi un e-mail,</li>
                    </ul>
                    <h3>Licence</h3>
                    <p>
                        blunderDB est distribué sous licence MIT. Cela signifie que vous êtes libre d'utiliser, copier, modifier, fusionner, publier, distribuer, sous-licencier et/ou vendre des copies du logiciel, à condition
                        que la notice de copyright originale et cette notice d'autorisation soient incluses dans toutes les copies ou parties substantielles du logiciel.
                    </p>
                    <h3>Remerciements</h3>
                    <p>Je dédie ce petit logiciel à ma compagne <strong>Anne-Claire</strong> et à notre chère fille <strong>Perrine</strong>. Je tiens à remercier tout particulièrement quelques amis :</p>
                    <ul>
                        <li>
                            <strong>Tristan Remille</strong>, pour m'avoir initié au backgammon avec joie et bienveillance ; pour m'avoir montré la Voie dans la compréhension de ce jeu merveilleux ; pour continuer à
                            me soutenir malgré mes piètres tentatives de mieux jouer.
                        </li>
                        <li><strong>Nicolas Harmand</strong>, un joyeux compagnon depuis plus d'une décennie dans de grandes aventures, et un fantastique partenaire de jeu depuis qu'il a attrapé le virus du backgammon.</li>
                    </ul>
                    <p>La table de match equity Kazaross-XG2 (MET) est attribuée à <strong>Neil Kazaross</strong>.</p>
                    <p>Les tables de take points et de valeurs de gammon sont tirées du livre <em>The Theory of Backgammon</em> de <strong>Dirk Schiemann</strong>.</p>
                    <p>
                        La base de données de sortie unilatérale à 6 points utilisée pour le calcul de l'EPC (Effective Pip Count) a été générée avec <strong>GNU Backgammon</strong> (GNUbg). GNUbg est un logiciel de backgammon libre et open source
                        distribué sous la Licence Publique Générale GNU.
                    </p>
`
};
