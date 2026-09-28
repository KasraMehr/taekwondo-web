# League API

The league roster is the source of truth. Creating a league tournament imports active roster entries for the selected group as snapshots. `sync-roster` may be used until weigh-in or draw starts. Publishing replaces that tournament's point ledger, so retrying or republishing a corrected result cannot double-count points.

Default scoring is gold 10, silver 6, bronze 3, passed weigh-in 1 and each win 2. Player and team ranking uses total points, gold, silver, bronze, round difference, wins, then name. All paths start with `/api/v1/organizations/{org}` and require `Authorization: Bearer {token}`.

| Method | Path | Purpose |
|---|---|---|
| POST | `/leagues` | Create league, stages, weeks and initial groups |
| GET | `/leagues/{league}` | Full league roster and structure |
| PUT | `/leagues/{league}/settings` | Scoring and promotion rules |
| POST | `/leagues/{league}/groups` | Add a group |
| POST | `/leagues/{league}/teams` | Add a club as a league team |
| PUT | `/leagues/{league}/teams/{team}` | Move or deactivate a team |
| POST/PUT/DELETE | `/leagues/{league}/athletes...` | Manage the league roster |
| POST | `/leagues/{league}/stages/{stage}/weeks` | Add a week |
| PUT | `/leagues/{league}/weeks/{week}/lock` | Lock or unlock a week |
| POST | `/leagues/{league}/tournaments` | Create a tournament and import its group roster |
| POST | `/leagues/{league}/tournaments/{tournament}/sync-roster` | Re-sync before weigh-in/draw |
| POST | `/leagues/{league}/tournaments/{tournament}/publish` | Validate brackets and rebuild points |
| GET | `/leagues/{league}/standings?stageId={stage}` | Player/team table; omit stage for season totals |
| POST | `/leagues/{league}/stages/{stage}/complete` | Save promoted teams and activate next stage |

```json
POST /leagues
{
  "name": "لیگ استان ۱۴۰۵",
  "gender": "male",
  "ageCategory": "بزرگسالان",
  "seasonName": "۱۴۰۵",
  "stageCount": 2,
  "weeksPerStage": 4,
  "groupNames": ["گروه A", "گروه B"]
}
```

```json
POST /leagues/{league}/athletes
{
  "profileId": "ATHLETE_PROFILE_UUID",
  "teamId": "LEAGUE_TEAM_UUID",
  "weightCategory": "-54",
  "coachName": "مربی",
  "ranking": 1
}
```

```json
POST /leagues/{league}/tournaments
{
  "name": "هفته ۱ - گروه A",
  "date": "2026-10-01",
  "courts": 3,
  "stageId": "STAGE_UUID",
  "weekId": "WEEK_UUID",
  "groupId": "GROUP_UUID"
}
```

Use the normal tournament URLs for weigh-in, draw, numbering and results. When every playable match in every drawn weight is complete, call `publish`, then read `standings`.
