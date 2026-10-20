# L'affichage d'un tournoi est une lecture servie au réseau du club, et rien d'autre

Statut : acceptée. Élargit la règle 1 de l'ADR-0039.
Voir aussi : ADR-0005 (le démon n'authentifie personne), ADR-0039 (périmètre du front web),
ADR-0057 (lectures conditionnelles de la Direction).

## Contexte

Un soir de club, le directeur fait tourner la Direction sur son portable ; les joueurs veulent
voir l'arbre, les appariements, les résultats et le classement sur leur téléphone sans aller
lire l'écran. La page d'affichage existe déjà (le moteur l'écrit dans un dossier choisi) mais
un fichier sur le disque du directeur n'atteint pas un téléphone. L'ADR-0039 ferme le front web
à trois verbes, et « tournois » figure dans sa liste de refus.

## Décision

1. **Quatrième verbe, en lecture seule : afficher un tournoi.** Le front web montre la page
   d'affichage d'une Direction (`directions.pageHtml`), la même que celle que le bureau écrit
   dans son dossier : arbre, appariements, résultats, classement. Aucun geste, aucune saisie.
   Le front n'a pas de second dessinateur de tournoi, comme il n'a pas de second dessinateur
   de plateau (ADR-0039 règle 3).
2. **La page du moteur est isolée** dans un cadre `sandbox` sans aucun droit : elle porte des
   noms saisis, et rien de ce qu'elle contient ne doit pouvoir exécuter quoi que ce soit ni
   atteindre la page ou le démon.
3. **Rafraîchie par lecture conditionnelle**, toutes les dix secondes : l'ETag d'ADR-0057
   rend un sondage inchangé presque gratuit (304 sans corps), et la minute repliée dans
   l'étiquette fait avancer horloge et pauses sans écriture. `/v1/events` est écarté : un flux
   ouvert par téléphone pèse plus qu'un 304, et une page de salle peut se permettre dix
   secondes.
4. **Aucune route `/v1` n'est créée** (ADR-0039 règle 4) : `directions.list` et
   `directions.pageHtml` existent pour tous les clients.
5. **Le bureau sert cet affichage sur le réseau local, sur demande.** Depuis la Direction, un
   geste lance un serveur HTTP sur toutes les interfaces (port 8770 par défaut) et montre le QR
   code de son adresse. Il est **éteint par défaut** et s'éteint à la fermeture. Ce serveur
   n'expose que :
   - la page web (`/app/`), qui ne contient aucune donnée ;
   - `POST /v1/directions.pageHtml`, et **seulement pour le tournoi choisi** ; toute autre
     route ou tout autre tournoi répond 404 sans atteindre le moteur.
   Le tenant est posé par le bureau, jamais lu dans la requête. Rien d'autre du fichier ouvert
   (positions, matchs, autres tournois, gestes de Direction, `/ops`, `/mcp`) n'est joignable.
6. **Aucune authentification n'est ajoutée** (ADR-0005). Le panneau le dit en permanence :
   quiconque est sur le réseau local peut lire la page — noms, appariements, résultats. C'est
   acceptable parce que ce qu'elle montre est ce qu'un affichage mural montre à la salle, et
   parce qu'elle ne permet aucune écriture.

## Conséquences

- La liste de refus de l'ADR-0039 perd « tournois » pour la seule lecture de la page
  d'affichage ; la saisie d'un résultat depuis le web reste refusée.
- Le QR code est produit en Go pur (`github.com/skip2/go-qrcode`, MIT), sans appel réseau.
- Le QR pointe vers une IPv4 de réseau local, au choix du directeur parmi celles de la machine
  (interfaces virtuelles — conteneurs, VPN — en dernier) ; sans aucune, le panneau le dit et
  ne dessine pas de QR vers 127.0.0.1.
- L'affichage est lié au fichier ouvert : ouvrir, créer ou fermer un fichier l'arrête, car le
  même identifiant désignerait un autre tournoi.
- La page sert dans la langue du catalogue de la Direction, pas dans celle du téléphone.

## Garde

`internal/gui/tournamentdisplay_test.go` (liste blanche, tenant posé par l'hôte, éteint par
défaut), `frontend/src/__tests__/directionNetworkDisplay.render.test.js`.
