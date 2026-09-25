CREATE TABLE users (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), email TEXT NOT NULL UNIQUE,
 name TEXT NOT NULL CHECK (length(trim(name)) > 0), password_hash TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE sessions (
 token_hash TEXT PRIMARY KEY, user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
 expires_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sessions_expiry ON sessions(expires_at);
CREATE TABLE organizations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name TEXT NOT NULL CHECK(length(trim(name)) > 0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE memberships (
 organization_id UUID NOT NULL REFERENCES organizations ON DELETE CASCADE,
 user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
 role TEXT NOT NULL CHECK(role IN ('owner','organizer','referee','coach','athlete')),
 PRIMARY KEY(organization_id,user_id)
);
CREATE TABLE clubs (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), organization_id UUID NOT NULL REFERENCES organizations,
 name TEXT NOT NULL CHECK(length(trim(name)) > 0), coach_id UUID REFERENCES users,
 UNIQUE(organization_id,name), UNIQUE(organization_id,id)
);
ALTER TABLE athletes ADD COLUMN user_id UUID REFERENCES users;
ALTER TABLE athletes ADD COLUMN organization_id UUID REFERENCES organizations;
ALTER TABLE athletes ADD COLUMN club_id UUID;
ALTER TABLE athletes ADD CONSTRAINT athlete_club_scope FOREIGN KEY(organization_id,club_id) REFERENCES clubs(organization_id,id);
CREATE UNIQUE INDEX athlete_account_per_org ON athletes(organization_id,user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX athlete_national_per_org ON athletes(organization_id,national_code) WHERE national_code IS NOT NULL AND national_code <> '';
ALTER TABLE tournaments ADD COLUMN organization_id UUID REFERENCES organizations;
ALTER TABLE tournaments ADD COLUMN revision BIGINT NOT NULL DEFAULT 1;
ALTER TABLE tournaments ADD COLUMN group_id UUID;
CREATE INDEX tournaments_organization ON tournaments(organization_id);
CREATE TABLE tournament_entries (
 tournament_id UUID NOT NULL REFERENCES tournaments ON DELETE CASCADE,
 athlete_id UUID NOT NULL, profile_id UUID REFERENCES athletes,
 data JSONB NOT NULL CHECK(jsonb_typeof(data) = 'object'),
 PRIMARY KEY(tournament_id,athlete_id)
);
CREATE TABLE tournament_matches (
 tournament_id UUID NOT NULL REFERENCES tournaments ON DELETE CASCADE,
 id UUID NOT NULL, data JSONB NOT NULL CHECK(jsonb_typeof(data) = 'object'),
 PRIMARY KEY(tournament_id,id)
);
INSERT INTO tournament_entries(tournament_id,athlete_id,data)
 SELECT t.id,(a->>'id')::uuid,a FROM tournaments t CROSS JOIN LATERAL jsonb_array_elements(t.athletes) a;
INSERT INTO tournament_matches(tournament_id,id,data)
 SELECT t.id,(m->>'id')::uuid,m FROM tournaments t CROSS JOIN LATERAL jsonb_array_elements(t.matches) m;
CREATE TABLE registrations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), tournament_id UUID NOT NULL REFERENCES tournaments ON DELETE CASCADE,
 athlete_id UUID NOT NULL REFERENCES athletes, requested_by UUID NOT NULL REFERENCES users,
 weight_category TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(tournament_id,athlete_id)
);
CREATE TABLE leagues (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), organization_id UUID NOT NULL REFERENCES organizations,
 name TEXT NOT NULL CHECK(length(trim(name)) > 0), config JSONB NOT NULL DEFAULT '{}',
 revision BIGINT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(organization_id,id)
);
CREATE TABLE league_groups (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), league_id UUID NOT NULL REFERENCES leagues ON DELETE CASCADE,
 name TEXT NOT NULL CHECK(length(trim(name)) > 0), UNIQUE(league_id,name), UNIQUE(league_id,id)
);
CREATE TABLE league_weeks (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), league_id UUID NOT NULL REFERENCES leagues ON DELETE CASCADE,
 name TEXT NOT NULL CHECK(length(trim(name)) > 0), position INTEGER NOT NULL CHECK(position>0),
 UNIQUE(league_id,position), UNIQUE(league_id,id)
);
CREATE TABLE league_teams (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), league_id UUID NOT NULL REFERENCES leagues ON DELETE CASCADE,
 club_id UUID NOT NULL REFERENCES clubs, group_id UUID,
 UNIQUE(league_id,club_id), UNIQUE(league_id,id),
 FOREIGN KEY(league_id,group_id) REFERENCES league_groups(league_id,id)
);
CREATE TABLE encounters (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), league_id UUID NOT NULL REFERENCES leagues ON DELETE CASCADE,
 week_id UUID NOT NULL, team1_id UUID NOT NULL, team2_id UUID NOT NULL,
 status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','completed')),
 bouts JSONB NOT NULL DEFAULT '[]', lineup JSONB NOT NULL DEFAULT '{}', revision BIGINT NOT NULL DEFAULT 1,
 CHECK(team1_id <> team2_id), UNIQUE(week_id,team1_id,team2_id),
 FOREIGN KEY(league_id,week_id) REFERENCES league_weeks(league_id,id),
 FOREIGN KEY(league_id,team1_id) REFERENCES league_teams(league_id,id),
 FOREIGN KEY(league_id,team2_id) REFERENCES league_teams(league_id,id)
);
CREATE TABLE audit_events (
 id BIGSERIAL PRIMARY KEY, organization_id UUID REFERENCES organizations, actor_id UUID REFERENCES users,
 action TEXT NOT NULL, resource TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
