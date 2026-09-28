# Accounts and access control

Roles are `athlete` (شاگرد), `coach` (مربی), `referee` (داور), `admin` (هیئت برگزاری), `manager` (مدیریت) and the immutable organization `owner`.

An owner or an admin with `members.manage` can create another account. An admin may only grant permissions that the admin already has. Only the owner can create or promote a `manager`.

## Create a scoped admin

```http
POST /api/v1/organizations/{org}/members/accounts
Authorization: Bearer OWNER_TOKEN
Content-Type: application/json
```

```json
{
  "name": "مسئول TA زمین A",
  "email": "court-a@example.com",
  "password": "temporary-password",
  "role": "admin",
  "permissions": ["members.manage", "tournaments.read", "matches.operate", "sheets.ta"],
  "scope": {
    "tournamentIds": ["TOURNAMENT_UUID"],
    "courts": ["A"],
    "stations": ["ta"]
  }
}
```

Empty `tournamentIds` or `courts` means all tournaments or courts. Supplying explicit values restricts both API reads and match operations. Court letters map as `A=1`, `B=2`, and so on.

## Useful URLs

| Method | URL | Use |
|---|---|---|
| GET | `/organizations/{org}/access-options` | Roles, permissions and supported scopes |
| GET | `/organizations/{org}/members` | Members and their effective configuration |
| POST | `/organizations/{org}/members/accounts` | Create a login and membership |
| PUT | `/organizations/{org}/members/{userId}` | Change role, permissions or scope |
| DELETE | `/organizations/{org}/members/{userId}` | Disable membership |
| POST | `/organizations/{org}/members/invitations` | Create a 72-hour invitation token |
| POST | `/invitations/accept` | Accept an invitation while logged in |
| GET | `/organizations/{org}/tournaments/{id}/access` | Current user's tournament access |
| GET | `/organizations/{org}/tournaments/{id}/sheets/ta?court=A` | TA sheet data for an allowed court |

Permission keys include `members.manage`, `audit.read`, `clubs.manage`, `athletes.read`, `athletes.manage`, `tournaments.read`, `tournaments.manage`, `draw.manage`, `weigh_in.manage`, `matches.operate`, `matches.override`, `sheets.ta`, `leagues.read`, and `leagues.manage`.
