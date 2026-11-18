ALTER TABLE athletes ADD COLUMN photo_data TEXT;
ALTER TABLE tournaments ADD COLUMN weigh_in_at TIMESTAMPTZ;
CREATE INDEX tournament_entries_profile ON tournament_entries(profile_id) WHERE profile_id IS NOT NULL;
