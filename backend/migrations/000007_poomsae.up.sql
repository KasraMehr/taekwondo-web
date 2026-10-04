CREATE UNIQUE INDEX athletes_organization_id_id ON athletes(organization_id,id);
CREATE TABLE poomsae_events (
 id UUID PRIMARY KEY, organization_id UUID NOT NULL REFERENCES organizations,
 data JSONB NOT NULL CHECK(jsonb_typeof(data)='object'),
 revision BIGINT NOT NULL DEFAULT 1 CHECK(revision>0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(organization_id,id)
);
CREATE TABLE poomsae_divisions (
 id UUID PRIMARY KEY, organization_id UUID NOT NULL, event_id UUID NOT NULL,
 data JSONB NOT NULL CHECK(jsonb_typeof(data)='object'),
 revision BIGINT NOT NULL DEFAULT 1 CHECK(revision>0),
 UNIQUE(organization_id,event_id,id), UNIQUE(organization_id,id),
 FOREIGN KEY(organization_id,event_id) REFERENCES poomsae_events(organization_id,id)
);
CREATE INDEX poomsae_divisions_event ON poomsae_divisions(organization_id,event_id);
CREATE TABLE poomsae_entries (
 id UUID PRIMARY KEY, organization_id UUID NOT NULL, division_id UUID NOT NULL,
 athlete_id UUID NOT NULL,
 data JSONB NOT NULL CHECK(jsonb_typeof(data)='object'),
 UNIQUE(division_id,athlete_id), UNIQUE(division_id,id),
 FOREIGN KEY(organization_id,division_id) REFERENCES poomsae_divisions(organization_id,id),
 FOREIGN KEY(organization_id,athlete_id) REFERENCES athletes(organization_id,id)
);
CREATE TABLE poomsae_draws (
 id UUID PRIMARY KEY, division_id UUID NOT NULL REFERENCES poomsae_divisions,
 version INTEGER NOT NULL CHECK(version>0), active BOOLEAN NOT NULL,
 data JSONB NOT NULL CHECK(jsonb_typeof(data)='object'), UNIQUE(division_id,version)
);
CREATE UNIQUE INDEX poomsae_active_draw ON poomsae_draws(division_id) WHERE active;
CREATE TABLE poomsae_change_log (
 id BIGSERIAL PRIMARY KEY, organization_id UUID NOT NULL, event_id UUID NOT NULL,
 division_id UUID, actor_id UUID NOT NULL REFERENCES users,
 action TEXT NOT NULL, reason TEXT NOT NULL DEFAULT '',
 before_data JSONB, after_data JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,event_id) REFERENCES poomsae_events(organization_id,id),
 FOREIGN KEY(organization_id,event_id,division_id) REFERENCES poomsae_divisions(organization_id,event_id,id)
);
CREATE INDEX poomsae_changes_resource ON poomsae_change_log(organization_id,event_id,division_id,id);
