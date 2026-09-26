package platform

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
	"time"
)

func listJSON(r *Request, q string, args ...any) (any, error) {
	rows, err := r.Tx.Query(r.Context(), q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		result = append(result, json.RawMessage(b))
	}
	return result, rows.Err()
}
func oneJSON(r *Request, q string, args ...any) (any, error) {
	var b []byte
	err := r.Tx.QueryRow(r.Context(), q, args...).Scan(&b)
	return json.RawMessage(b), err
}
func (a *API) organizationRoutes(api *gin.RouterGroup) {
	api.POST("/invitations/accept", a.route(true, false, func(r *Request) (any, error) {
		in, err := bind[struct {
			Token string `json:"token"`
		}](r)
		if err != nil {
			return nil, err
		}
		var invitationID, orgID, role string
		var permissions, scope []byte
		err = r.Tx.QueryRow(r.Context(), `SELECT i.id::text,i.organization_id::text,i.role,i.permissions,i.scope FROM user_invitations i JOIN users u ON lower(u.email)=lower(i.email) WHERE i.token_hash=$1 AND i.accepted_at IS NULL AND i.expires_at>now() AND u.id=$2 FOR UPDATE`, tokenHash(strings.TrimSpace(in.Token)), r.UserID).Scan(&invitationID, &orgID, &role, &permissions, &scope)
		if err != nil {
			return nil, err
		}
		_, err = r.Tx.Exec(r.Context(), `INSERT INTO memberships(organization_id,user_id,role,permissions,permissions_custom,scope,created_by) SELECT organization_id,$2,role,permissions,true,scope,invited_by FROM user_invitations WHERE id=$1 ON CONFLICT(organization_id,user_id) DO UPDATE SET role=excluded.role,permissions=excluded.permissions,permissions_custom=true,scope=excluded.scope,active=true,updated_at=now()`, invitationID, r.UserID)
		if err != nil {
			return nil, err
		}
		_, err = r.Tx.Exec(r.Context(), `UPDATE user_invitations SET accepted_at=now() WHERE id=$1`, invitationID)
		if err != nil {
			return nil, err
		}
		return gin.H{"organizationId": orgID, "role": role}, nil
	}))
	api.POST("/organizations", a.route(true, false, func(r *Request) (any, error) {
		in, err := bind[struct {
			Name string `json:"name"`
		}](r)
		if err != nil {
			return nil, err
		}
		in.Name = strings.TrimSpace(in.Name)
		if in.Name == "" || len(in.Name) > 150 {
			return nil, bad("organization name is required (max 150 bytes)")
		}
		var id string
		err = r.Tx.QueryRow(r.Context(), `INSERT INTO organizations(name) VALUES($1) RETURNING id::text`, in.Name).Scan(&id)
		if err != nil {
			return nil, err
		}
		_, err = r.Tx.Exec(r.Context(), `INSERT INTO memberships VALUES($1,$2,'owner')`, id, r.UserID)
		return gin.H{"id": id, "name": in.Name, "role": "owner"}, err
	}))
	api.GET("/organizations", a.route(true, false, func(r *Request) (any, error) {
		return listJSON(r, `SELECT jsonb_build_object('id',o.id,'name',o.name,'role',m.role) FROM organizations o JOIN memberships m ON m.organization_id=o.id WHERE m.user_id=$1 ORDER BY o.name`, r.UserID)
	}))
	api.GET("/organizations/:org/members", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permMembersManage); err != nil {
			return nil, err
		}
		return listJSON(r, `SELECT jsonb_build_object('userId',u.id,'name',u.name,'email',u.email,'role',m.role,'permissions',m.permissions,'permissionsCustom',m.permissions_custom,'scope',m.scope,'active',m.active) FROM memberships m JOIN users u ON u.id=m.user_id WHERE m.organization_id=$1 ORDER BY u.name`, r.OrganizationID)
	}))
	api.GET("/organizations/:org/access-options", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permMembersManage); err != nil {
			return nil, err
		}
		return gin.H{"roles": []gin.H{{"value": "athlete", "label": "شاگرد"}, {"value": "coach", "label": "مربی"}, {"value": "referee", "label": "داور"}, {"value": "admin", "label": "هیئت برگزاری / ادمین"}, {"value": "manager", "label": "مدیریت"}}, "permissions": permissionOptions(), "scope": gin.H{"courts": "A تا Z یا شماره زمین", "stations": []string{"ta", "weigh_in", "draw", "results"}}}, nil
	}))
	api.POST("/organizations/:org/members", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permMembersManage); err != nil {
			return nil, err
		}
		in, err := bind[struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}](r)
		if err != nil {
			return nil, err
		}
		if err = validateRole(in.Role); err != nil {
			return nil, err
		}
		if in.Role == "manager" && r.Role != "owner" {
			return nil, forbidden()
		}
		var id string
		if err = r.Tx.QueryRow(r.Context(), `SELECT id::text FROM users WHERE email=$1`, strings.ToLower(strings.TrimSpace(in.Email))).Scan(&id); err != nil {
			return nil, err
		}
		if id == r.UserID {
			return nil, bad("owner cannot change their own role")
		}
		tag, err := r.Tx.Exec(r.Context(), `INSERT INTO memberships VALUES($1,$2,$3) ON CONFLICT(organization_id,user_id) DO UPDATE SET role=excluded.role WHERE memberships.role<>'owner'`, r.OrganizationID, id, in.Role)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, forbidden()
		}
		return gin.H{"userId": id, "role": in.Role}, nil
	}))
	type accountInput struct {
		Name        string      `json:"name"`
		Email       string      `json:"email"`
		Password    string      `json:"password"`
		Role        string      `json:"role"`
		Permissions []string    `json:"permissions"`
		Scope       AccessScope `json:"scope"`
	}
	api.POST("/organizations/:org/members/accounts", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permMembersManage); err != nil {
			return nil, err
		}
		in, err := bind[accountInput](r)
		if err != nil {
			return nil, err
		}
		in.Name = strings.TrimSpace(in.Name)
		in.Email = strings.ToLower(strings.TrimSpace(in.Email))
		parsedEmail, parseErr := mail.ParseAddress(in.Email)
		if in.Name == "" || parseErr != nil || parsedEmail.Address != in.Email || len(in.Password) < 10 || len(in.Password) > 72 {
			return nil, bad("name, email and password of 10–72 bytes are required")
		}
		if err = validateRole(in.Role); err != nil {
			return nil, err
		}
		if in.Role == "manager" && r.Role != "owner" {
			return nil, forbidden()
		}
		permissions, err := normalizePermissions(in.Permissions)
		if err != nil {
			return nil, err
		}
		if err = r.canGrant(permissions); err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		var userID string
		err = r.Tx.QueryRow(r.Context(), `INSERT INTO users(email,name,password_hash) VALUES($1,$2,$3) RETURNING id::text`, in.Email, in.Name, string(hash)).Scan(&userID)
		if err != nil {
			return nil, err
		}
		permissionJSON, _ := json.Marshal(permissions)
		scopeJSON, _ := json.Marshal(in.Scope)
		_, err = r.Tx.Exec(r.Context(), `INSERT INTO memberships(organization_id,user_id,role,permissions,permissions_custom,scope,created_by) VALUES($1,$2,$3,$4,$5,$6,$7)`, r.OrganizationID, userID, in.Role, permissionJSON, in.Permissions != nil, scopeJSON, r.UserID)
		if err != nil {
			return nil, err
		}
		return gin.H{"userId": userID, "name": in.Name, "email": in.Email, "role": in.Role, "permissions": permissions, "scope": in.Scope}, nil
	}))
	api.PUT("/organizations/:org/members/:member", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permMembersManage); err != nil {
			return nil, err
		}
		in, err := bind[accountInput](r)
		if err != nil {
			return nil, err
		}
		if err = validateRole(in.Role); err != nil {
			return nil, err
		}
		if in.Role == "manager" && r.Role != "owner" {
			return nil, forbidden()
		}
		permissions, err := normalizePermissions(in.Permissions)
		if err != nil {
			return nil, err
		}
		if err = r.canGrant(permissions); err != nil {
			return nil, err
		}
		permissionJSON, _ := json.Marshal(permissions)
		scopeJSON, _ := json.Marshal(in.Scope)
		tag, err := r.Tx.Exec(r.Context(), `UPDATE memberships SET role=$4,permissions=$5,permissions_custom=true,scope=$6,active=true,updated_at=now() WHERE organization_id=$1 AND user_id=$2 AND role<>'owner' AND ($3='owner' OR role<>'manager')`, r.OrganizationID, r.C.Param("member"), r.Role, in.Role, permissionJSON, scopeJSON)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, forbidden()
		}
		return gin.H{"userId": r.C.Param("member"), "role": in.Role, "permissions": permissions, "scope": in.Scope}, nil
	}))
	api.DELETE("/organizations/:org/members/:member", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permMembersManage); err != nil {
			return nil, err
		}
		tag, err := r.Tx.Exec(r.Context(), `UPDATE memberships SET active=false,updated_at=now() WHERE organization_id=$1 AND user_id=$2 AND role<>'owner' AND ($3='owner' OR role<>'manager')`, r.OrganizationID, r.C.Param("member"), r.Role)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, forbidden()
		}
		return nil, nil
	}))
	api.POST("/organizations/:org/members/invitations", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permMembersManage); err != nil {
			return nil, err
		}
		in, err := bind[accountInput](r)
		if err != nil {
			return nil, err
		}
		if err = validateRole(in.Role); err != nil {
			return nil, err
		}
		if in.Role == "manager" && r.Role != "owner" {
			return nil, forbidden()
		}
		permissions, err := normalizePermissions(in.Permissions)
		if err != nil {
			return nil, err
		}
		if err = r.canGrant(permissions); err != nil {
			return nil, err
		}
		raw := make([]byte, 32)
		if _, err = rand.Read(raw); err != nil {
			return nil, err
		}
		token := hex.EncodeToString(raw)
		permissionJSON, _ := json.Marshal(permissions)
		scopeJSON, _ := json.Marshal(in.Scope)
		expires := time.Now().UTC().Add(72 * time.Hour)
		var id string
		err = r.Tx.QueryRow(r.Context(), `INSERT INTO user_invitations(organization_id,email,role,permissions,scope,token_hash,invited_by,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(organization_id,email) WHERE accepted_at IS NULL DO UPDATE SET role=excluded.role,permissions=excluded.permissions,scope=excluded.scope,token_hash=excluded.token_hash,invited_by=excluded.invited_by,expires_at=excluded.expires_at RETURNING id::text`, r.OrganizationID, strings.ToLower(strings.TrimSpace(in.Email)), in.Role, permissionJSON, scopeJSON, tokenHash(token), r.UserID, expires).Scan(&id)
		if err != nil {
			return nil, err
		}
		return gin.H{"id": id, "token": token, "expiresAt": expires}, nil
	}))
	api.GET("/organizations/:org/audit", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permAuditRead); err != nil {
			return nil, err
		}
		return listJSON(r, `SELECT jsonb_build_object('id',id,'actorId',actor_id,'action',action,'resource',resource,'createdAt',created_at) FROM audit_events WHERE organization_id=$1 ORDER BY id DESC LIMIT 200`, r.OrganizationID)
	}))
}

type profileInput struct {
	Name         string  `json:"name"`
	NationalCode *string `json:"nationalCode"`
	BirthDate    *string `json:"birthDate"`
	Gender       string  `json:"gender"`
	Phone        *string `json:"phone"`
	Email        *string `json:"email"`
	ClubID       *string `json:"clubId"`
	UserID       *string `json:"userId"`
}

const profileJSON = `jsonb_build_object('id',a.id,'name',a.name,'nationalCode',a.national_code,'birthDate',a.birth_date,'gender',a.gender,'phone',a.phone,'email',a.email,'clubId',a.club_id,'userId',a.user_id,'isActive',a.is_active)`
const profileAccess = `( $3 IN ('owner','manager','admin','organizer') OR a.user_id=$2 OR ($3='coach' AND EXISTS(SELECT 1 FROM clubs c WHERE c.id=a.club_id AND c.organization_id=a.organization_id AND c.coach_id=$2)))`

func (r *Request) canUseClub(id *string) error {
	if id == nil {
		if r.Role == "coach" {
			return forbidden()
		}
		return nil
	}
	var ok bool
	err := r.Tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM clubs WHERE id=$1 AND organization_id=$2 AND ($3 IN ('owner','manager','admin','organizer') OR coach_id=$4))`, id, r.OrganizationID, r.Role, r.UserID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return forbidden()
	}
	return nil
}
func (a *API) profileRoutes(org *gin.RouterGroup) {
	org.GET("/clubs", a.route(true, true, func(r *Request) (any, error) {
		return listJSON(r, `SELECT jsonb_build_object('id',id,'name',name,'coachId',coach_id) FROM clubs WHERE organization_id=$1 ORDER BY name`, r.OrganizationID)
	}))
	org.POST("/clubs", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permClubsManage); err != nil {
			return nil, err
		}
		in, err := bind[struct {
			Name    string  `json:"name"`
			CoachID *string `json:"coachId"`
		}](r)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(in.Name) == "" {
			return nil, bad("club name is required")
		}
		if in.CoachID != nil {
			var ok bool
			err = r.Tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM memberships WHERE organization_id=$1 AND user_id=$2 AND role='coach')`, r.OrganizationID, in.CoachID).Scan(&ok)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, bad("coach must have a coach membership")
			}
		}
		return oneJSON(r, `INSERT INTO clubs(organization_id,name,coach_id) VALUES($1,$2,$3) RETURNING jsonb_build_object('id',id,'name',name,'coachId',coach_id)`, r.OrganizationID, strings.TrimSpace(in.Name), in.CoachID)
	}))
	org.PUT("/clubs/:club", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permClubsManage); err != nil {
			return nil, err
		}
		in, err := bind[struct {
			Name string `json:"name"`
		}](r)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(in.Name) == "" {
			return nil, bad("club name is required")
		}
		return oneJSON(r, `UPDATE clubs SET name=$3 WHERE id=$1 AND organization_id=$2 RETURNING jsonb_build_object('id',id,'name',name,'coachId',coach_id)`, r.C.Param("club"), r.OrganizationID, strings.TrimSpace(in.Name))
	}))
	org.GET("/athletes", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permAthletesRead); err != nil {
			return nil, err
		}
		return listJSON(r, `SELECT `+profileJSON+` FROM athletes a WHERE organization_id=$1 AND `+profileAccess+` ORDER BY a.name,a.id LIMIT 200`, r.OrganizationID, r.UserID, r.Role)
	}))
	org.GET("/athletes/:athlete", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permAthletesRead); err != nil {
			return nil, err
		}
		return oneJSON(r, `SELECT `+profileJSON+` FROM athletes a WHERE organization_id=$1 AND `+profileAccess+` AND a.id=$4`, r.OrganizationID, r.UserID, r.Role, r.C.Param("athlete"))
	}))
	save := func(update bool) endpoint {
		return func(r *Request) (any, error) {
			if r.Role != "coach" && r.Role != "athlete" {
				if err := r.permit(permAthletesManage); err != nil {
					return nil, err
				}
			}
			in, err := bind[profileInput](r)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 || (in.Gender != "male" && in.Gender != "female") {
				return nil, bad("name and valid gender are required")
			}
			if r.Role == "athlete" {
				in.UserID = &r.UserID
				if in.ClubID != nil {
					return nil, bad("club assignment requires a coach or organizer")
				}
			}
			if err = r.canUseClub(in.ClubID); err != nil {
				return nil, err
			}
			if in.UserID != nil {
				var ok bool
				err = r.Tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM memberships WHERE organization_id=$1 AND user_id=$2)`, r.OrganizationID, in.UserID).Scan(&ok)
				if err != nil {
					return nil, err
				}
				if !ok {
					return nil, bad("account must be a member of this organization")
				}
			}
			if update {
				// Coaches may edit their club's athletes, but cannot claim an existing account.
				var existingUser *string
				err = r.Tx.QueryRow(r.Context(), `SELECT a.user_id::text FROM athletes a WHERE organization_id=$1 AND `+profileAccess+` AND a.id=$4 FOR UPDATE`, r.OrganizationID, r.UserID, r.Role, r.C.Param("athlete")).Scan(&existingUser)
				if err != nil {
					return nil, err
				}
				if r.Role == "coach" || r.Role == "athlete" {
					in.UserID = existingUser
				}
				return oneJSON(r, `UPDATE athletes a SET name=$3,national_code=$4,birth_date=$5,gender=$6,phone=$7,email=$8,club_id=$9,user_id=$10,updated_at=now() WHERE id=$1 AND organization_id=$2 RETURNING `+profileJSON, r.C.Param("athlete"), r.OrganizationID, strings.TrimSpace(in.Name), in.NationalCode, in.BirthDate, in.Gender, in.Phone, in.Email, in.ClubID, in.UserID)
			}
			if r.Role == "coach" && in.UserID != nil {
				return nil, bad("only an organizer can link another account")
			}
			return oneJSON(r, `INSERT INTO athletes AS a(id,organization_id,name,national_code,birth_date,gender,phone,email,club_id,user_id) VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+profileJSON, r.OrganizationID, strings.TrimSpace(in.Name), in.NationalCode, in.BirthDate, in.Gender, in.Phone, in.Email, in.ClubID, in.UserID)
		}
	}
	org.POST("/athletes", a.route(true, true, save(false)))
	org.PUT("/athletes/:athlete", a.route(true, true, save(true)))
	org.DELETE("/athletes/:athlete", a.route(true, true, func(r *Request) (any, error) {
		if err := r.permit(permAthletesManage); err != nil {
			return nil, err
		}
		tag, err := r.Tx.Exec(r.Context(), `UPDATE athletes SET is_active=false,updated_at=now() WHERE id=$1 AND organization_id=$2`, r.C.Param("athlete"), r.OrganizationID)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, notFound()
		}
		return nil, nil
	}))
}
