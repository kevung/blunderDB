# blunderDB
Backgammon position and match database software

## Roadmap

import position/match in sqlite db (0.txt; 1.xg)
  via files (txt, xg)
  via clipboard
export position/match
  via files
  via clipboard (xgid, raw)
tag position/matchs (cataegories)
notes commentaries
filter by selector:
  size blunder
  points made
  double decision
  take decision
  score
  date
  tag

gui
  filter
  tag
  categories
  preview position

unique executable

possibility to send positions to commun database

## Database representation

Tables
  match: id, game_id, player1_score, player2_score
  position: id, checker distribution, points, cube, score,
    prev pos (foreign key), next pos (foreign key)

## Inspirations

Zotero capabilities


