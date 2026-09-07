# Diriger un tournoi

Diriger un tournoi de backgammon depuis blunderDB : inscrire, apparier, lancer, saisir les
résultats, tenir les tableaux, classer — et retrouver les matchs du tournoi rangés par ronde
et par joueur quand ils sont transcrits ou importés.

Moteur : **Nicomaque**, créé par **Nicolas Harmand**
([PileOfCells/backgammon-tournoi](https://github.com/PileOfCells/backgammon-tournoi)) —
bibliothèque Go sans dépendance, journal d'événements, état reconstruit par rejeu.

Cadrage : entretien du 2026-09-07, dix-neuf décisions ; décision durable dans
[ADR-0047](../../docs/adr/0047-directing-a-tournament-creates-its-matches-before-they-are-played.md)
(qui amende 0037) ; vocabulaire dans `CONTEXT.md`, section « Directing a tournament ».

| Document | Contenu |
|---|---|
| [decisions.md](decisions.md) | les dix-neuf décisions de l'entretien, dans l'ordre |
| [fonctionnel.md](fonctionnel.md) | les objets, le cycle de vie, la configuration, les inscriptions, diriger, le temps, les Slots, les sorties, vingt-huit flux |
| [ux.md](ux.md) | placement, six vues, fiche de résultat, clavier, budgets de clics par persona, correction de saisie |
| [integration.md](integration.md) | schéma 2.23.0, paquet `direction`, Wails, frontend, CLI, documentation, risques |
| [nicomaque.md](nicomaque.md) | les quatorze demandes au moteur, à ouvrir dans son dépôt |
| [plan.md](plan.md) | cinq lots, quarante-huit issues, ordre et parallélisme |

## Les deux contraintes de premier rang

Posées par l'utilisateur pendant l'entretien, elles priment sur le reste :

1. **La souris est première, le clavier accélère** ; la charge mentale d'un TD occasionnel
   et le coût d'entrée sont bas — de « Nouveau » à la première ronde lancée en moins de
   90 secondes, sans rien lire.
2. **Le crédit est visible** : Nicomaque, créé par Nicolas Harmand, dans l'aide, la
   documentation, un bouton info, et en pied des pages produites ; la documentation du
   moteur est publiée dans les neuf langues de blunderDB.

## Le jalon qui décide

Le moteur n'a jamais dirigé un vrai tournoi. Le **lot 1 se termine par un tournoi de club de
16 à 32 joueurs réellement dirigé**, en doublant sur papier. Ce que le directeur a voulu
faire et que la page ne proposait pas devient des issues, avant d'attaquer le lot 3.
