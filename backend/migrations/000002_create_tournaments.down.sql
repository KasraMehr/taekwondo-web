DROP INDEX IF EXISTS idx_tournaments_elimination_matches_gin;
DROP INDEX IF EXISTS idx_tournaments_matches_gin;
DROP INDEX IF EXISTS idx_tournaments_athletes_gin;

DROP INDEX IF EXISTS idx_tournaments_team_tournament_id;
DROP INDEX IF EXISTS idx_tournaments_week_id;
INDEX IF EXISTS idx_tournaments_league_id_id;

DROP INDEX IF EXISTS idx_tournaments_format;
DROP INDEX IF EXISTS idx_tournaments_gender_age_category;
DROP INDEX IF EXISTS idx_tournaments_event_date;

DROP TABLE IF EXISTS tournaments;
