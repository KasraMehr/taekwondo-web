ALTER TABLE memberships DROP CONSTRAINT memberships_role_check;
UPDATE memberships SET role='admin' WHERE role='organizer';
ALTER TABLE memberships ADD CONSTRAINT memberships_role_check
  CHECK(role IN ('owner','manager','admin','organizer','referee','coach','athlete'));
ALTER TABLE memberships
  ADD COLUMN permissions JSONB NOT NULL DEFAULT '[]' CHECK(jsonb_typeof(permissions)='array'),
  ADD COLUMN permissions_custom BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN scope JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(scope)='object'),
  ADD COLUMN active BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN created_by UUID REFERENCES users,
  ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE user_invitations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations ON DELETE CASCADE,
  email TEXT NOT NULL,
  role TEXT NOT NULL CHECK(role IN ('manager','admin','referee','coach','athlete')),
  permissions JSONB NOT NULL DEFAULT '[]' CHECK(jsonb_typeof(permissions)='array'),
  scope JSONB NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(scope)='object'),
  token_hash TEXT NOT NULL UNIQUE,
  invited_by UUID NOT NULL REFERENCES users,
  expires_at TIMESTAMPTZ NOT NULL,
  accepted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX active_invitation_per_org_email ON user_invitations(organization_id,email)
  WHERE accepted_at IS NULL;
