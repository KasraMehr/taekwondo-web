ALTER TABLE tournaments ADD COLUMN settings JSONB;
ALTER TABLE tournament_entries ADD CONSTRAINT tournament_entry_profile_unique UNIQUE(tournament_id,profile_id);
