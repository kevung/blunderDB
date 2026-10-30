-- Banc V0.2 : couverture des fenêtres par les Repères (move.roll_tick_ms / move.tick_ms).
-- Lecture seule, sur n'importe quelle base blunderDB :
--     sqlite3 -readonly -header -markdown BASE.db < 02-couverture.sql
-- N'écrit que dans la base temporaire de la connexion (vues TEMP).
--
-- Vocabulaire (ADR-0082 règles 1-2). Un coup « à jet » (dice_1 > 0 : coup de pions,
-- danse, coup non consigné) porte deux Repères, r = jet et t = action. Un double, une
-- prise, une passe n'ont que t. L'Action précédente est celle qui précède dans la partie
-- (game_number, move_number), quel que soit son camp ; le premier jet d'une partie n'en a pas.
-- Les matchs sans aucun Repère sont ignorés.

-- Paramètres : demi-largeurs de la fenêtre dense, en ms (ADR-0109 règle 1 : [-1 ; +2] s),
-- et seuils des parts (3 s et 10 s).
CREATE TEMP TABLE IF NOT EXISTS param(avant_ms INT, apres_ms INT, court_ms INT, long_ms INT);
DELETE FROM param;
INSERT INTO param VALUES (1000, 2000, 3000, 10000);

DROP VIEW IF EXISTS temp.act;
CREATE TEMP VIEW act AS
SELECT g.match_id AS match_id, g.game_number AS gn, mv.move_number AS mn,
       (mv.dice_1 > 0) AS a_jet,
       mv.roll_tick_ms AS r, mv.tick_ms AS t,
       LAG(mv.tick_ms) OVER (PARTITION BY mv.game_id ORDER BY mv.move_number) AS prev_t,
       ROW_NUMBER()    OVER (PARTITION BY mv.game_id ORDER BY mv.move_number) AS rang
FROM move mv JOIN game g ON g.id = mv.game_id
WHERE g.match_id IN (SELECT g2.match_id FROM move m2 JOIN game g2 ON g2.id = m2.game_id
                     WHERE m2.roll_tick_ms IS NOT NULL OR m2.tick_ms IS NOT NULL);

-- Une ligne par mesure d'écart : kind 'ja' = jet -> action (t - r),
-- 'aj' = action précédente -> jet (r - prev_t). Clé 'TOUS' = toutes les parties réunies.
DROP VIEW IF EXISTS temp.ecart;
CREATE TEMP VIEW ecart AS
SELECT CAST(match_id AS TEXT) AS cle, 'ja' AS kind, t - r AS ms FROM act WHERE a_jet AND r IS NOT NULL AND t IS NOT NULL
UNION ALL SELECT 'TOUS', 'ja', t - r FROM act WHERE a_jet AND r IS NOT NULL AND t IS NOT NULL
UNION ALL SELECT CAST(match_id AS TEXT), 'aj', r - prev_t FROM act WHERE a_jet AND r IS NOT NULL AND prev_t IS NOT NULL
UNION ALL SELECT 'TOUS', 'aj', r - prev_t FROM act WHERE a_jet AND r IS NOT NULL AND prev_t IS NOT NULL;

DROP VIEW IF EXISTS temp.ecart_rang;
CREATE TEMP VIEW ecart_rang AS
SELECT cle, kind, ms,
       ROW_NUMBER() OVER (PARTITION BY cle, kind ORDER BY ms) AS k,
       COUNT(*)     OVER (PARTITION BY cle, kind)             AS n
FROM ecart WHERE ms >= 0;

-- 1. Distribution des écarts en secondes (à rebours exclus), rang le plus proche (nearest-rank).
SELECT cle AS "match", CASE kind WHEN 'ja' THEN 'jet -> action' ELSE 'action -> jet suivant' END AS ecart,
       MAX(n) AS n,
       ROUND(MAX(CASE WHEN k = (n + 1) / 2 THEN ms END) / 1000.0, 1)           AS mediane_s,
       ROUND(MAX(CASE WHEN k = (n * 10 + 99) / 100 THEN ms END) / 1000.0, 1)   AS p10_s,
       ROUND(MAX(CASE WHEN k = (n * 90 + 99) / 100 THEN ms END) / 1000.0, 1)   AS p90_s,
       ROUND(MAX(ms) / 1000.0, 1)                                              AS max_s
FROM ecart_rang
GROUP BY cle, kind
ORDER BY cle = 'TOUS', CAST(cle AS INT), kind DESC;

-- 2. Part des jets (hors premier de partie) selon le délai après l'action précédente.
-- Dénominateur : tous les jets non premiers d'une partie ; « inconnu » = Repère manquant d'un
-- côté ou de l'autre ; « à rebours » = jet avant l'action précédente.
WITH j AS (
  SELECT CAST(match_id AS TEXT) AS cle, r - prev_t AS d, r, prev_t FROM act WHERE a_jet AND rang > 1
  UNION ALL SELECT 'TOUS', r - prev_t, r, prev_t FROM act WHERE a_jet AND rang > 1)
SELECT cle AS "match", COUNT(*) AS jets,
  SUM(r IS NOT NULL AND prev_t IS NOT NULL AND d >= 0 AND d < (SELECT court_ms FROM param)) AS inf_3s,
  SUM(r IS NOT NULL AND prev_t IS NOT NULL AND d >= (SELECT court_ms FROM param) AND d <= (SELECT long_ms FROM param)) AS de_3_a_10s,
  SUM(r IS NOT NULL AND prev_t IS NOT NULL AND d > (SELECT long_ms FROM param)) AS sup_10s,
  SUM(r IS NOT NULL AND prev_t IS NOT NULL AND d < 0) AS a_rebours,
  SUM(r IS NULL OR prev_t IS NULL) AS inconnu,
  ROUND(100.0 * SUM(r IS NOT NULL AND prev_t IS NOT NULL AND d >= 0 AND d < (SELECT court_ms FROM param)) / NULLIF(SUM(r IS NOT NULL AND prev_t IS NOT NULL AND d >= 0), 0), 1) AS pct_inf_3s_mesures,
  ROUND(100.0 * SUM(r IS NOT NULL AND prev_t IS NOT NULL AND d >= 0 AND d <= (SELECT long_ms FROM param)) / NULLIF(SUM(r IS NOT NULL AND prev_t IS NOT NULL AND d >= 0), 0), 1) AS pct_inf_10s_mesures
FROM j GROUP BY cle ORDER BY cle = 'TOUS', CAST(cle AS INT);

-- 3. Repères manquants et à rebours. Un coup à jet attend deux Repères ; tout coup attend t.
-- jet_sans_action / action_sans_jet / aucun : coups à jet ; cube_sans_t : doubles, prises, passes.
-- rebours_ja : t < r dans un même coup ; rebours_aj : r < t de l'Action précédente.
WITH c AS (
  SELECT CAST(match_id AS TEXT) AS cle, * FROM act
  UNION ALL SELECT 'TOUS', * FROM act)
SELECT cle AS "match", COUNT(*) AS actions,
  SUM(a_jet) AS coups_a_jet,
  SUM(a_jet AND r IS NOT NULL AND t IS NOT NULL) AS deux_reperes,
  SUM(a_jet AND r IS NOT NULL AND t IS NULL) AS jet_sans_action,
  SUM(a_jet AND r IS NULL AND t IS NOT NULL) AS action_sans_jet,
  SUM(a_jet AND r IS NULL AND t IS NULL) AS aucun,
  SUM(NOT a_jet) AS cubes,
  SUM(NOT a_jet AND t IS NULL) AS cube_sans_t,
  SUM(a_jet AND r IS NOT NULL AND t IS NOT NULL AND t < r) AS rebours_ja,
  SUM(a_jet AND r IS NOT NULL AND prev_t IS NOT NULL AND r < prev_t) AS rebours_aj
FROM c GROUP BY cle ORDER BY cle = 'TOUS', CAST(cle AS INT);

-- 4. Les Repères comme fenêtres. « Fenêtre du jet » = de l'action précédente au jet (le jet se
-- tape après que les dés se sont posés : l'événement est dans cet intervalle) ; « fenêtre de
-- l'action » = du jet à l'action. Largeur = écart, en secondes ; parts d'écarts <= 3 s, <= 10 s.
SELECT cle AS "match", CASE kind WHEN 'aj' THEN 'fenetre du jet' ELSE 'fenetre de l''action' END AS fenetre,
       COUNT(*) AS n,
       ROUND(AVG(ms) / 1000.0, 1) AS moyenne_s,
       ROUND(100.0 * SUM(ms <= (SELECT court_ms FROM param)) / COUNT(*), 1) AS pct_inf_3s,
       ROUND(100.0 * SUM(ms <= (SELECT long_ms FROM param)) / COUNT(*), 1) AS pct_inf_10s,
       ROUND(100.0 * SUM(ms > 30000) / COUNT(*), 1) AS pct_sup_30s
FROM ecart WHERE ms >= 0 GROUP BY cle, kind ORDER BY cle = 'TOUS', CAST(cle AS INT), kind DESC;

-- 5. Couverture de l'étendue annotée par des fenêtres denses [x - avant ; x + apres] posées
-- sur chaque Repère. Toutes les fenêtres ont la même largeur : l'union des fenêtres triées par
-- instant vaut la somme des min(largeur, écart au Repère précédent). Étendue = du premier
-- Repère - avant au dernier + apres. Un Repère par ligne, par match ; un trou de plus de
-- 10 minutes entre deux Repères coupe l'étendue (vidéo laissée sans annotation).
DROP VIEW IF EXISTS temp.repere;
CREATE TEMP VIEW repere AS
SELECT DISTINCT match_id, x FROM (
  SELECT match_id, r AS x FROM act WHERE r IS NOT NULL
  UNION ALL SELECT match_id, t FROM act WHERE t IS NOT NULL);

WITH s AS (
  SELECT match_id, x,
         x - LAG(x) OVER (PARTITION BY match_id ORDER BY x) AS dx
  FROM repere),
m AS (
  SELECT match_id, COUNT(*) AS reperes,
         SUM(CASE WHEN dx IS NULL OR dx > 600000 THEN (SELECT avant_ms + apres_ms FROM param)
                  ELSE MIN(dx, (SELECT avant_ms + apres_ms FROM param)) END) AS union_ms,
         SUM(CASE WHEN dx IS NULL OR dx > 600000 THEN (SELECT avant_ms + apres_ms FROM param)
                  ELSE dx END) AS etendue_ms
  FROM s GROUP BY match_id)
SELECT * FROM (
  SELECT CAST(match_id AS TEXT) AS "match", reperes, ROUND(etendue_ms / 60000.0, 1) AS etendue_min,
         ROUND(union_ms / 60000.0, 1) AS fenetres_min, ROUND(100.0 * union_ms / etendue_ms, 1) AS couverture_pct
  FROM m
  UNION ALL
  SELECT 'TOUS', SUM(reperes), ROUND(SUM(etendue_ms) / 60000.0, 1), ROUND(SUM(union_ms) / 60000.0, 1),
         ROUND(100.0 * SUM(union_ms) / SUM(etendue_ms), 1) FROM m)
ORDER BY "match" = 'TOUS', CAST("match" AS INT);
