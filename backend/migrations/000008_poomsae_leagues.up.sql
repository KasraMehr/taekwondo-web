CREATE TABLE poomsae_leagues (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations,
  data JSONB NOT NULL CHECK (jsonb_typeof(data) = 'object'),
  revision BIGINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (organization_id, id)
);
CREATE INDEX poomsae_leagues_organization ON poomsae_leagues(organization_id, created_at DESC);
