# Grill Nicomaque — journal des décisions (2026-09-07)

D1. Frontière produit : un tournoi dirigé est une SOURCE de Matchs. Le Tournament existant,
    créé avant ses matchs ; chaque match Nicomaque est un emplacement que remplit une
    transcription ou un import. Emplacements vides admis (tournoi de club sans enregistrement) ;
    cas plein = BMAB (transcription obligatoire). Valeur pour le TD : regrouper les matchs par
    ronde / joueur. → ADR dans la lignée de 0044 : « diriger un tournoi n'est pas un second
    produit, c'est créer les Matchs avant qu'ils soient joués ».
D2. Une seule entité : le Tournament existant porte optionnellement une Direction (le journal
    Nicomaque). Termes Tournament + Direction écrits dans CONTEXT.md (section « Directing a
    tournament »). Collisions évitées : Journal (Entraînement), Replay (Transcription).
    Numéro d'ADR pressenti : 0047 (0046 = seuils de bibliothèque).
D3. Participant = inscription propre à une Direction ; lien au Player = égalité de nom choisie
    par le TD (autocomplétion sur les Players, PR → cote d'entrée). Rattachement d'un match à un
    emplacement : création depuis l'emplacement, ou geste explicite (proposition sur noms égaux),
    jamais inféré. Pas de fiche « personne ». Annuaire = vue dérivée (tous les Participants de la
    base, dédoublonnés par nom), export/import CSV, « reprendre les inscrits de <tournoi> ».
D4. Déploiement : bureau + page HTML statique autonome (render.Page : arbres SVG, vies, matchs
    en cours, classement) réécrite à chaque événement, pour second écran / partage / impression
    (feuille d'appariements). ADR-0039 intact. Téléphone des joueurs / route serve : hors
    périmètre, à décider après un vrai tournoi, ADR propre.
D5. Périmètre : TOUT RESTE_A_FAIRE.md est spécifié (avec sa règle, tranchée pendant le grill) et
    planifié. Ordre des lots = tracer bullets : lot 1 = suisse 2 vies continu + tableau à
    exemptions, 16-32 joueurs, avec tables (indispo/réservées/changement), forfait d'un match,
    longueurs par tour. Lots suivants : micro-rondes, pauses, retardataires, têtes de série
    (option de tirage, désactivée par défaut — l'étude les écarte), prix en %, export journal
    lisible. Écran joueur exclu (D4). Les cinq formats sont exposés dès le lot 1 (config = liste
    de phases, rendu générique).
D6. Autorité : le moteur propose, le TD décide, la Direction enregistre, rien n'est refusé sauf
    l'impossible (joueur occupé/inconnu). Action manuelle toujours disponible (lancer un match
    entre deux Participants libres, longueur et table au choix) ; résultat incohérent accepté
    avec avertissement ; Warnings affichés en permanence, jamais bloquants ; réparation d'un
    tableau = annulation + relance, et Nicomaque doit la PROPOSER (son point 4).
D7. Cycle de vie d'une Direction : EN PRÉPARATION (aucun match lancé ; configuration réécrite en
    place, inscriptions libres) → EN COURS (dès le premier match : configuration modifiable par
    événement seulement ; Nicomaque : EvConfigChanged général remplaçant EvLengthChanged —
    bascule, phase ajoutée APRÈS la courante, tables, prix — avec recompute + warnings ; la phase
    courante et les passées ne changent jamais de type) → TERMINÉE (EvFinished, lecture seule,
    réouvrable par événement pour corriger). Supprimer la Direction ≠ supprimer le Tournament :
    les matchs perdent leur emplacement, pas leur tournoi.
D8. Résultat = événement de la Direction dit par le TD, jamais dérivé silencieusement ; pré-rempli
    depuis le Match rattaché quand le score final est connu (un geste) ; écart Match/résultat
    affiché comme avertissement sur l'emplacement. Transcription créée depuis un emplacement :
    en-tête hérité (noms, longueur, tournoi, ronde, date), emplacement réservé dès le brouillon.
CONTRAINTE UX (utilisateur, 2026-09-07) : « la représentation et l'UX seront ultra importants »
    → passe UX = budgets KLM par flux et persona, maquettes de la zone principale, prototypes
    jetables pour vies / arbre / file des propositions.
D9. Langue : Nicomaque émet des CODES structurés (libellés {kind:round,n}, {kind:final}… ;
    notes {kind:alive,lives} ; warnings {code, params}) et garde un rendu français pour console
    et démo ; parseRound disparaît (ronde = champ). Journal versionné (`version` sur Event).
    Une seule issue Nicomaque, AVANT tout tournoi réel. blunderDB traduit dans ses 9 catalogues.
D10. Persistance : table append-only une ligne par événement (tournament_id, seq, kind, time,
    payload JSON) écrite dans la transaction du geste ; état jamais stocké, rejoué à l'ouverture ;
    match.<slot> = identifiant Nicomaque (M12) unique par tournoi. Schéma 2.23.0 × 3 copies (migration Postgres 024),
    demo.db. Logique sur Database, exposée à Wails. CLI `tournament` non interactif, lot tardif :
    verify / standings / page / export. Serveur : rien. Export de base : la Direction voyage
    avec son Tournament (issuance.Carried étendu explicitement).
D11. Temps : bande d'horloge permanente (heure, écoulé, joués/en cours/restants, min/pt observé
    vs planifié, fin estimée par Forecast rafraîchie à chaque événement et chaque minute, matchs
    lents ≥ 1,5× attendu, seuil réglable). Pauses = plages dans la config ; Propose propose quand
    même avec avertissement « finirait pendant la pause ». Micro-rondes : défaut = Propose à la
    demande du TD ; option de phase batch_minutes=X : blunderDB appelle Propose à l'échéance,
    file d'attente + compte à rebours visibles ; règle Nicomaque : à l'échéance, joueurs libres
    du même groupe appariés au hasard, le reste attend.
D12. Retardataire après tirage : prend une place BYE d'une section non commencée à ce tour
    (EvPlayerAdded + slot), sinon inscrit et entre à la prochaine phase/section qui l'admet ;
    jamais de retirage. Forfait d'un match : EvResult forfeit=true sans retrait, offert à la
    saisie. Retrait différé : EvPlayerWithdrawn after_current=true (finit son match en cours).
    Abandon en poule : retrait, non joués perdus par forfait, joués conservés.
D13. Tables : liste numérotée en config (nombre + exceptions : indisponible, réservée à une
    section/phase), modifiable en cours par événement. assignTables saute indispo/réservées ;
    proposition sans table = « en attente de table », lançable avec numéro manuel. EvTableChanged.
    Vue : grille des tables (match, durée, libre/indispo).
D14. Dotation : droit d'entrée, retenue (montant ou %), une structure par section (principal,
    consolante, dernière chance) en % du reste, arrondi à l'unité, reste au premier ; montant
    fixe possible. Pool recalculé à chaque inscription/retrait. Classement général = Nicomaque ;
    chaque section de tableau a son classement propre (tour atteint) sur lequel sa dotation se
    répartit. Validation FFBG hors périmètre, la doc le dit.
D15. Cohabitation : la vue tournoi = zone principale quand l'onglet Tournoi est actif ET une
    Direction est ouverte ; tout autre onglet ramène le plateau sans fermer. Une seule Direction
    ouverte à la fois, choisie dans la liste des tournois ; ouvrir un tournoi sans Direction
    propose d'en créer une. La Direction vit en arrière-plan (horloge, échéances) ; badge sur
    l'onglet Tournoi = propositions en attente + ligne dans la barre d'état. Raccourci
    Ctrl+Maj+D si libre / commande `direct` (à vérifier dans raccourcis.rst). Le panneau
    Tournoi existant gagne la Direction, pas d'onglet supplémentaire.
D16. Personas TD : Marc (club, 16-24, seul, interrompu : 1-2 clics par geste courant) ; Sophie
    (grand tournoi BMAB, 48-64, deux jours, assistant, retransmission, transcription obligatoire,
    dotation, projection : rigueur, corrections tracées, imprimable) ; Yanis (remplaçant, ne
    connaît pas le format : l'interface dit toujours quoi faire ensuite, une erreur se répare) ;
    Léa (TD-joueuse : tout en deux minutes entre ses matchs, reprise « où en étais-je »).
CRÉDIT (exigence utilisateur) : « Nicomaque, créé par Nicolas Harmand » + lien vers le dépôt,
    dans l'aide, la doc (À propos + page tournoi) et un bouton info dans la gestion d'un
    tournoi. Issue Nicomaque : GitHub Pages avec documents introductifs et spécification.
    Le site Nicomaque (GitHub Pages) est en NEUF langues, comme blunderDB ; même chaîne
    Sphinx + gettext recommandée.
D17. Squelette UX : une page « Direction » (propositions | grille des tables | file d'attente)
    par défaut en cours ; vues Joueurs, Arbres, Classement, Historique, Réglages ; en préparation
    ouverture sur Réglages puis Joueurs ; panneau ancré inchangé ; bouton ⓘ crédit.
    Résultat : VAINQUEUR seul exigé ; score libre (les deux ou rien) ; remarque facultative
    (« tombé au temps », abandon…) TRÈS discrète (dépliée sur ⋯), champ texte de l'événement
    de résultat. Ligne de commande : `result <table> <vainqueur> [score]`.
D18. Souris première, clavier = raccourcis ; charge mentale minimale pour un TD occasionnel.
    Chaque geste = bouton visible libellé ; pas de touche-mode. Clavier réduit à l'idiome
    existant (j/k/Entrée dans une liste, Échap, Tab) + chiffres = fiche de la table N ; ligne de
    commande pour les habitués (result/start/withdraw/table/direct). Raccourcis dans les
    infobulles. Yanis ne lit rien avant de commencer.
CONTRAINTE (utilisateur) : le COÛT D'ENTRÉE doit être bas — premier tournoi créé et première
    ronde lancée sans lire ; configurations nommées, défauts qui tiennent ; hypothèse : une
    Direction d'exemple dans demo.db.
D19. Hypothèses UX 1-8 acceptées. Exigence : CORRIGER UNE ERREUR DE SAISIE sur place (F28) —
    dernier résultat affiché avec [Corriger] à 1 clic, Ctrl+Z = correction du dernier événement,
    case du match fini rouvrable, Participant corrigeable sans changer de slug.
