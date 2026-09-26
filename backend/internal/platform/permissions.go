package platform

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	permMembersManage     = "members.manage"
	permAuditRead         = "audit.read"
	permClubsManage       = "clubs.manage"
	permAthletesRead      = "athletes.read"
	permAthletesManage    = "athletes.manage"
	permTournamentsRead   = "tournaments.read"
	permTournamentsManage = "tournaments.manage"
	permDrawManage        = "draw.manage"
	permWeighInManage     = "weigh_in.manage"
	permMatchesOperate    = "matches.operate"
	permMatchesOverride   = "matches.override"
	permSheetsTA          = "sheets.ta"
	permLeaguesRead       = "leagues.read"
	permLeaguesManage     = "leagues.manage"
)

var validPermissions = map[string]bool{
	permMembersManage: true, permAuditRead: true, permClubsManage: true,
	permAthletesRead: true, permAthletesManage: true, permTournamentsRead: true,
	permTournamentsManage: true, permDrawManage: true, permWeighInManage: true,
	permMatchesOperate: true, permMatchesOverride: true, permSheetsTA: true, permLeaguesRead: true, permLeaguesManage: true,
}

var rolePermissions = map[string][]string{
	"manager":   {"*"},
	"admin":     {permAuditRead, permClubsManage, permAthletesRead, permAthletesManage, permTournamentsRead, permTournamentsManage, permDrawManage, permWeighInManage, permMatchesOperate, permMatchesOverride, permSheetsTA, permLeaguesRead, permLeaguesManage},
	"organizer": {permAuditRead, permClubsManage, permAthletesRead, permAthletesManage, permTournamentsRead, permTournamentsManage, permDrawManage, permWeighInManage, permMatchesOperate, permMatchesOverride, permSheetsTA, permLeaguesRead, permLeaguesManage},
	"referee":   {permTournamentsRead, permMatchesOperate, permSheetsTA, permLeaguesRead},
	"coach":     {permAthletesRead, permTournamentsRead},
	"athlete":   {permAthletesRead, permTournamentsRead},
}

type AccessScope struct {
	TournamentIDs []string `json:"tournamentIds,omitempty"`
	Courts        []string `json:"courts,omitempty"`
	Stations      []string `json:"stations,omitempty"`
}

func normalizePermissions(values []string) ([]string, error) {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if !validPermissions[v] {
			return nil, bad("invalid permission: " + v)
		}
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out, nil
}

func (r *Request) loadAccess(permissionBytes, scopeBytes []byte, custom bool) error {
	var explicit []string
	if err := json.Unmarshal(permissionBytes, &explicit); err != nil {
		return err
	}
	if err := json.Unmarshal(scopeBytes, &r.Scope); err != nil {
		return err
	}
	if r.Role == "owner" {
		r.Permissions = map[string]bool{"*": true}
		return nil
	}
	if !custom {
		explicit = rolePermissions[r.Role]
	}
	r.Permissions = map[string]bool{}
	for _, p := range explicit {
		r.Permissions[p] = true
	}
	return nil
}
func (r *Request) hasPermission(p string) bool { return r.Permissions["*"] || r.Permissions[p] }
func (r *Request) permit(p string) error {
	if !r.hasPermission(p) {
		return forbidden()
	}
	return nil
}

func containsFold(values []string, want string) bool {
	for _, v := range values {
		if strings.EqualFold(strings.TrimSpace(v), want) {
			return true
		}
	}
	return false
}
func courtLabel(court int) string {
	if court >= 1 && court <= 26 {
		return string(rune('A' + court - 1))
	}
	return strconv.Itoa(court)
}
func (r *Request) permitsTournament(id string) bool {
	return len(r.Scope.TournamentIDs) == 0 || containsFold(r.Scope.TournamentIDs, id)
}
func (r *Request) permitsCourt(court int) bool {
	return len(r.Scope.Courts) == 0 || containsFold(r.Scope.Courts, courtLabel(court)) || containsFold(r.Scope.Courts, strconv.Itoa(court))
}
func (r *Request) authorizeTournament(permission, id string) error {
	if err := r.permit(permission); err != nil {
		return err
	}
	if !r.permitsTournament(id) {
		return forbidden()
	}
	return nil
}
func (r *Request) authorizeMatch(permission, tournamentID, matchID string) error {
	if err := r.authorizeTournament(permission, tournamentID); err != nil {
		return err
	}
	var court int
	err := r.Tx.QueryRow(r.Context(), `SELECT (tm.data->>'court')::int FROM tournament_matches tm JOIN tournaments t ON t.id=tm.tournament_id WHERE tm.tournament_id=$1 AND tm.id=$2 AND t.organization_id=$3`, tournamentID, matchID, r.OrganizationID).Scan(&court)
	if err != nil {
		return err
	}
	if !r.permitsCourt(court) {
		return forbidden()
	}
	return nil
}
func (r *Request) authorizeStation(station string) error {
	if !containsFold(r.Scope.Stations, station) && len(r.Scope.Stations) > 0 {
		return forbidden()
	}
	return nil
}
func (r *Request) canGrant(permissions []string) error {
	if r.Role == "owner" || r.Role == "manager" || r.Permissions["*"] {
		return nil
	}
	for _, p := range permissions {
		if !r.hasPermission(p) {
			return forbidden()
		}
	}
	return nil
}

func permissionOptions() []map[string]string {
	labels := map[string]string{permMembersManage: "مدیریت کاربران", permAuditRead: "مشاهده گزارش تغییرات", permClubsManage: "مدیریت باشگاه‌ها", permAthletesRead: "مشاهده ورزشکاران", permAthletesManage: "مدیریت ورزشکاران", permTournamentsRead: "مشاهده تورنومنت", permTournamentsManage: "مدیریت تورنومنت", permDrawManage: "قرعه‌کشی", permWeighInManage: "وزن‌کشی", permMatchesOperate: "اجرای بازی و ثبت نتیجه", permMatchesOverride: "اصلاح نتیجه", permSheetsTA: "برگه TA", permLeaguesRead: "مشاهده لیگ", permLeaguesManage: "مدیریت لیگ"}
	out := make([]map[string]string, 0, len(labels))
	for key, label := range labels {
		out = append(out, map[string]string{"value": key, "label": label})
	}
	return out
}
func validateRole(role string) error {
	switch role {
	case "manager", "admin", "referee", "coach", "athlete":
		return nil
	default:
		return fmt.Errorf("%w", bad("invalid membership role"))
	}
}
