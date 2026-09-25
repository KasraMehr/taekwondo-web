ALTER TABLE leagues
  ADD COLUMN gender TEXT CHECK (gender IN ('male','female')),
  ADD COLUMN age_category TEXT CHECK (age_category IN ('خردسالان','نونهالان','نوجوانان','امیدها','بزرگسالان')),
  ADD COLUMN status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','active','completed','archived')),
  ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE league_groups
  ADD COLUMN position INTEGER;
WITH numbered AS (
  SELECT id, row_number() OVER (PARTITION BY league_id ORDER BY name,id) AS n FROM league_groups
)
UPDATE league_groups g SET position=n.n FROM numbered n WHERE n.id=g.id;
ALTER TABLE league_groups ALTER COLUMN position SET NOT NULL;
ALTER TABLE league_groups ADD CHECK(position > 0);
CREATE UNIQUE INDEX league_group_position ON league_groups(league_id, position);

CREATE TABLE league_stages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  league_id UUID NOT NULL REFERENCES leagues ON DELETE CASCADE,
  name TEXT NOT NULL CHECK(length(trim(name)) > 0),
  position INTEGER NOT NULL CHECK(position > 0),
  status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','active','completed')),
  promoted_team_ids UUID[] NOT NULL DEFAULT '{}',
  UNIQUE(league_id, position), UNIQUE(league_id, id)
);
INSERT INTO league_stages(league_id,name,position,status)
SELECT id,'مرحله 1',1,'active' FROM leagues;

ALTER TABLE league_weeks
  ADD COLUMN stage_id UUID,
  ADD COLUMN date_range TEXT,
  ADD COLUMN locked BOOLEAN NOT NULL DEFAULT false;
UPDATE league_weeks w SET stage_id=s.id
FROM league_stages s WHERE s.league_id=w.league_id AND s.position=1;
ALTER TABLE league_weeks ALTER COLUMN stage_id SET NOT NULL;
ALTER TABLE league_weeks
  ADD CONSTRAINT league_week_stage_scope FOREIGN KEY(league_id, stage_id)
  REFERENCES league_stages(league_id, id) ON DELETE CASCADE;
ALTER TABLE league_weeks DROP CONSTRAINT IF EXISTS league_weeks_league_id_position_key;
CREATE UNIQUE INDEX league_week_position ON league_weeks(league_id, stage_id, position);

ALTER TABLE league_teams
  ADD COLUMN active BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE league_athletes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  league_id UUID NOT NULL REFERENCES leagues ON DELETE CASCADE,
  profile_id UUID NOT NULL REFERENCES athletes,
  team_id UUID REFERENCES league_teams ON DELETE SET NULL,
  group_id UUID,
  club_name TEXT NOT NULL DEFAULT '',
  coach_name TEXT NOT NULL DEFAULT '',
  weight_category TEXT NOT NULL,
  ranking INTEGER CHECK(ranking IS NULL OR ranking > 0),
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(league_id, profile_id), UNIQUE(league_id, id),
  FOREIGN KEY(league_id, team_id) REFERENCES league_teams(league_id, id),
  FOREIGN KEY(league_id, group_id) REFERENCES league_groups(league_id, id)
);
CREATE INDEX league_athletes_roster ON league_athletes(league_id, group_id, active);

ALTER TABLE tournaments
  ADD CONSTRAINT tournament_league_scope FOREIGN KEY(league_id) REFERENCES leagues(id) ON DELETE SET NULL,
  ADD CONSTRAINT tournament_week_scope FOREIGN KEY(league_id, week_id) REFERENCES league_weeks(league_id, id),
  ADD CONSTRAINT tournament_group_scope FOREIGN KEY(league_id, group_id) REFERENCES league_groups(league_id, id);
CREATE UNIQUE INDEX tournament_per_league_week_group
  ON tournaments(league_id, week_id, COALESCE(group_id, '00000000-0000-0000-0000-000000000000'::uuid))
  WHERE league_id IS NOT NULL AND week_id IS NOT NULL;

CREATE TABLE league_point_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  league_id UUID NOT NULL REFERENCES leagues ON DELETE CASCADE,
  stage_id UUID NOT NULL,
  tournament_id UUID NOT NULL REFERENCES tournaments ON DELETE CASCADE,
  tournament_entry_id UUID NOT NULL,
  profile_id UUID REFERENCES athletes,
  team_id UUID REFERENCES league_teams ON DELETE SET NULL,
  athlete_name TEXT NOT NULL,
  club_name TEXT NOT NULL,
  weight_category TEXT NOT NULL,
  event_type TEXT NOT NULL CHECK(event_type IN ('weigh_in','win','gold','silver','bronze','match_stats')),
  event_key TEXT NOT NULL,
  points INTEGER NOT NULL DEFAULT 0,
  gold INTEGER NOT NULL DEFAULT 0,
  silver INTEGER NOT NULL DEFAULT 0,
  bronze INTEGER NOT NULL DEFAULT 0,
  round_diff INTEGER NOT NULL DEFAULT 0,
  wins_2_0 INTEGER NOT NULL DEFAULT 0,
  wins_2_1 INTEGER NOT NULL DEFAULT 0,
  losses_0_2 INTEGER NOT NULL DEFAULT 0,
  losses_1_2 INTEGER NOT NULL DEFAULT 0,
  wins INTEGER NOT NULL DEFAULT 0,
  losses INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  FOREIGN KEY(league_id, stage_id) REFERENCES league_stages(league_id, id),
  UNIQUE(tournament_id, event_key)
);
CREATE INDEX league_points_standings ON league_point_events(league_id, stage_id, team_id);

CREATE TABLE league_tournament_publications (
  tournament_id UUID PRIMARY KEY REFERENCES tournaments ON DELETE CASCADE,
  league_id UUID NOT NULL REFERENCES leagues ON DELETE CASCADE,
  tournament_revision BIGINT NOT NULL,
  published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_by UUID REFERENCES users
);
