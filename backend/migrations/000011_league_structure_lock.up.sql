ALTER TABLE leagues ADD COLUMN structure_locked boolean NOT NULL DEFAULT false;

-- Existing league entries may have inherited a global club/coach rather than the league roster.
UPDATE tournament_entries e SET data=jsonb_set(jsonb_set(e.data,'{club}',to_jsonb(la.club_name)),'{coach}',to_jsonb(la.coach_name))
FROM tournaments t, league_athletes la
WHERE t.id=e.tournament_id AND la.league_id=t.league_id
AND (e.athlete_id=la.id OR e.data->>'profileId'=la.profile_id::text);

UPDATE leagues l SET structure_locked=true WHERE EXISTS (
 SELECT 1 FROM tournaments t JOIN tournament_matches m ON m.tournament_id=t.id
 WHERE t.league_id=l.id AND (m.data->>'status'='ongoing' OR
 (m.data->>'status'='completed' AND m.data->>'athlete1Id' IS NOT NULL AND m.data->>'athlete2Id' IS NOT NULL))
);

-- A permanent latch: clearing results or deleting a tournament cannot unlock a season.
CREATE FUNCTION lock_started_league() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.data->>'status'='ongoing' OR (NEW.data->>'status'='completed'
 AND NEW.data->>'athlete1Id' IS NOT NULL AND NEW.data->>'athlete2Id' IS NOT NULL) THEN
  UPDATE leagues SET structure_locked=true, revision=revision+1, updated_at=now()
  WHERE id=(SELECT league_id FROM tournaments WHERE id=NEW.tournament_id) AND NOT structure_locked;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER league_started AFTER INSERT OR UPDATE OF data ON tournament_matches
FOR EACH ROW EXECUTE FUNCTION lock_started_league();
