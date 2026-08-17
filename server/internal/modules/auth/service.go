package auth

import (
	"context"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/inwardclub/server/internal/platform/authn"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
	inputvalidation "github.com/inwardclub/server/internal/platform/validation"
)

// MemberTierResolver resolves a member's current VIP tier for the "me" surface.
// It is owned by the member module and wired at composition time; when nil, the
// profile is returned without tier info. A nil tier (with nil error) means the
// member is not yet ranked.
type MemberTierResolver interface {
	CurrentTier(ctx context.Context, memberID int64) (*MemberVIPTier, error)
}

// AvatarUploader uploads a registration avatar to object storage and returns its
// public URL. It is satisfied by the asset service; kept as an interface here so
// the auth module stays decoupled from asset.
type AvatarUploader interface {
	UploadAvatar(ctx context.Context, r io.Reader, size int64, contentType string) (string, error)
}

// Service implements the authentication flows for all three audiences.
type Service struct {
	tokens   *authn.Manager
	wechat   WeChatClient
	members  MemberRepository
	accounts AccountRepository
	staff    StaffRepository
	tiers    MemberTierResolver
	avatars  AvatarUploader
}

// NewService builds the auth service. tiers may be nil until the member VIP-tier
// resolver is wired; the "me" profile then omits tier info. avatars may be nil in
// tests that never exercise the registration avatar upload. staff is optional
// for older unit tests that do not exercise staff identity.
func NewService(tokens *authn.Manager, wechat WeChatClient, members MemberRepository, accounts AccountRepository, tiers MemberTierResolver, avatars AvatarUploader, staff ...StaffRepository) *Service {
	var staffRepo StaffRepository
	if len(staff) > 0 {
		staffRepo = staff[0]
	}
	return &Service{
		tokens: tokens, wechat: wechat, members: members, accounts: accounts,
		staff: staffRepo, tiers: tiers, avatars: avatars,
	}
}

// MiniLogin exchanges a WeChat code for a member session. A returning member is
// logged straight in. A first-time or pre-registered user receives IsNew=true
// and a short-lived register ticket instead of a full member session.
func (s *Service) MiniLogin(ctx context.Context, code string) (LoginResponse, error) {
	var err error
	code, err = inputvalidation.OpaqueToken("微信登录凭证", code, 2048)
	if err != nil {
		return LoginResponse{}, apperr.Invalid(err.Error())
	}
	session, err := s.wechat.Code2Session(ctx, code)
	if err != nil {
		return LoginResponse{}, apperr.Unauthenticated("微信身份获取失败").WithCause(err)
	}
	member, err := s.members.GetByOpenID(ctx, session.OpenID)
	if apperr.From(err) != nil && apperr.From(err).Code == apperr.CodeNotFound {
		// First-time user: defer creation until the form is submitted. Hand back a
		// register ticket (no phone yet) instead of a member session.
		ticket, terr := s.tokens.IssueRegisterTicket(session.OpenID, "")
		if terr != nil {
			return LoginResponse{}, apperr.Internal(terr)
		}
		return LoginResponse{IsNew: true, RegisterTicket: ticket}, nil
	}
	if err != nil {
		return LoginResponse{}, err
	}
	if !member.ProfileCompleted {
		// A reservation-only pre-registration is still an incomplete member. A
		// deliberate login must continue through avatar/nickname/phone/gender.
		ticket, terr := s.tokens.IssueRegisterTicket(session.OpenID, "")
		if terr != nil {
			return LoginResponse{}, apperr.Internal(terr)
		}
		return LoginResponse{IsNew: true, RegisterTicket: ticket}, nil
	}
	if s.staff != nil {
		staff, staffErr := s.staff.GetByMemberID(ctx, member.ID)
		if staffErr == nil && staff.Status == StatusActive {
			return s.issueStaffSession(member, staff, false)
		}
		if staffErr != nil && apperr.From(staffErr).Code != apperr.CodeNotFound {
			return LoginResponse{}, staffErr
		}
	}
	return s.issueMemberSession(member, false)
}

// MiniPreRegister creates or reuses an OpenID-backed member. Completed members
// recover their ordinary session; new users receive reservation-only access
// without completing registration.
func (s *Service) MiniPreRegister(ctx context.Context, code string) (LoginResponse, error) {
	var err error
	code, err = inputvalidation.OpaqueToken("微信登录凭证", code, 2048)
	if err != nil {
		return LoginResponse{}, apperr.Invalid(err.Error())
	}
	session, err := s.wechat.Code2Session(ctx, code)
	if err != nil {
		return LoginResponse{}, apperr.Unauthenticated("微信身份获取失败").WithCause(err)
	}
	member, err := s.members.GetByOpenID(ctx, session.OpenID)
	if apperr.From(err) != nil && apperr.From(err).Code == apperr.CodeNotFound {
		id, createErr := s.members.Create(ctx, Member{
			WeChatOpenID: session.OpenID, Nickname: "inward会员", ProfileCompleted: false,
		})
		if createErr != nil {
			// Concurrent wx.login calls may race on the unique OpenID. Re-read the
			// winner instead of creating a second pre-registration.
			member, err = s.members.GetByOpenID(ctx, session.OpenID)
			if err != nil {
				return LoginResponse{}, createErr
			}
		} else {
			member, err = s.members.GetByID(ctx, id)
		}
	}
	if err != nil {
		return LoginResponse{}, err
	}
	if member.Status != StatusActive {
		return LoginResponse{}, apperr.Forbidden("会员已被停用")
	}
	if member.ProfileCompleted {
		// A known, fully registered OpenID should recover the ordinary login
		// session transparently instead of being downgraded to pre_member.
		if s.staff != nil {
			staff, staffErr := s.staff.GetByMemberID(ctx, member.ID)
			if staffErr == nil && staff.Status == StatusActive {
				return s.issueStaffSession(member, staff, false)
			}
			if staffErr != nil && apperr.From(staffErr).Code != apperr.CodeNotFound {
				return LoginResponse{}, staffErr
			}
		}
		return s.issueMemberSession(member, false)
	}
	pair, err := s.tokens.Issue(authn.Identity{
		SubjectID: member.ID, SubjectType: authn.SubjectPreMember, Role: authn.RolePreMember,
		Audience: authn.AudienceMini, TokenVersion: member.TokenVersion,
	})
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}
	return LoginResponse{Token: pair, IsNew: true, SubjectType: authn.SubjectPreMember}, nil
}

// MiniRegister completes a member's registration: it verifies the register
// ticket and either inserts a new member or upgrades an OpenID-only
// pre-registration with avatar, nickname, gender and phone before minting a
// full mini session.
func (s *Service) MiniRegister(ctx context.Context, req WeChatRegisterRequest) (LoginResponse, error) {
	registerTicket, validationErr := inputvalidation.OpaqueToken("注册凭证", req.RegisterTicket, 4096)
	if validationErr != nil {
		return LoginResponse{}, apperr.Invalid(validationErr.Error())
	}
	openID, phone, err := s.tokens.ParseRegisterTicket(registerTicket)
	if err != nil || openID == "" {
		return LoginResponse{}, apperr.Unauthenticated("invalid or expired register ticket")
	}
	if phone == "" {
		// The phone must have been authorized (and embedded in the ticket) via the
		// phone-mask step before registration; the single-use WeChat code is never
		// decrypted here.
		return LoginResponse{}, apperr.Invalid("phone is required")
	}
	avatarURL, validationErr := inputvalidation.HTTPURL("头像", req.AvatarURL, false)
	if validationErr != nil {
		return LoginResponse{}, apperr.Invalid(validationErr.Error())
	}
	nickname, validationErr := inputvalidation.PlainText(req.Nickname, inputvalidation.TextOptions{
		Label: "昵称", MinRunes: 1, MaxRunes: 30,
	})
	if validationErr != nil {
		return LoginResponse{}, apperr.Invalid(validationErr.Error())
	}
	gender := strings.TrimSpace(req.Gender)
	if gender == "" {
		return LoginResponse{}, apperr.Invalid("gender is required")
	}
	if gender != "male" && gender != "female" && gender != "other" {
		return LoginResponse{}, apperr.Invalid("gender must be male, female, or other")
	}

	// Idempotent for completed members; pre-registered members are upgraded in
	// place so their existing reservations remain attached to the same ID.
	member, err := s.members.GetByOpenID(ctx, openID)
	if apperr.From(err) != nil && apperr.From(err).Code == apperr.CodeNotFound {
		invitedByMemberID, resolveErr := s.resolveInviter(ctx, req.InviterCode)
		if resolveErr != nil {
			return LoginResponse{}, resolveErr
		}
		id, cerr := s.createMember(ctx, openID, avatarURL, nickname, gender, phone, invitedByMemberID)
		if cerr != nil {
			return LoginResponse{}, cerr
		}
		member, err = s.members.GetByID(ctx, id)
	} else if err == nil && !member.ProfileCompleted {
		invitedByMemberID, resolveErr := s.resolveInviter(ctx, req.InviterCode)
		if resolveErr != nil {
			return LoginResponse{}, resolveErr
		}
		if completeErr := s.completePreRegisteredMember(ctx, member.ID, avatarURL, nickname, gender, phone, invitedByMemberID); completeErr != nil {
			return LoginResponse{}, completeErr
		}
		member, err = s.members.GetByID(ctx, member.ID)
	}
	if err != nil {
		return LoginResponse{}, err
	}
	return s.issueMemberSession(member, true)
}

func (s *Service) resolveInviter(ctx context.Context, rawCode string) (*int64, error) {
	inviterCode, err := inputvalidation.InviteCode(rawCode, true)
	if err != nil {
		return nil, apperr.Invalid(err.Error())
	}
	if inviterCode == "" {
		return nil, nil
	}
	inviterID, found, err := s.members.FindIDByInviteCode(ctx, inviterCode)
	if err != nil || !found {
		return nil, err
	}
	return &inviterID, nil
}

func (s *Service) completePreRegisteredMember(ctx context.Context, memberID int64, avatarURL, nickname, gender, phone string, invitedByMemberID *int64) error {
	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := s.members.CompleteRegistration(ctx, memberID, Member{
			AvatarURL: avatarURL, Nickname: nickname, Gender: gender, Phone: phone,
			InviteCode: newInviteCode(), InvitedByMemberID: invitedByMemberID, ProfileCompleted: true,
		})
		if err == nil {
			return nil
		}
		if platdb.IsDuplicate(err) {
			continue
		}
		return err
	}
	return apperr.Internal(fmt.Errorf("could not allocate a unique invite code after %d attempts", maxAttempts))
}

// GetPhoneMask decrypts the single-use WeChat phone code ONCE, embeds the phone
// number into a fresh register ticket (so registration never re-decrypts the
// code), and returns the masked phone for display plus the new ticket. The
// client must submit the returned ticket to /register.
func (s *Service) GetPhoneMask(ctx context.Context, registerTicket, phoneCode string) (masked, ticket string, err error) {
	registerTicket, err = inputvalidation.OpaqueToken("注册凭证", registerTicket, 4096)
	if err != nil {
		return "", "", apperr.Invalid(err.Error())
	}
	phoneCode, err = inputvalidation.OpaqueToken("手机号授权凭证", phoneCode, 2048)
	if err != nil {
		return "", "", apperr.Invalid(err.Error())
	}
	openID, _, terr := s.tokens.ParseRegisterTicket(registerTicket)
	if terr != nil || openID == "" {
		return "", "", apperr.Unauthenticated("invalid or expired register ticket")
	}
	phone, perr := s.wechat.GetPhoneNumber(ctx, phoneCode)
	if perr != nil {
		return "", "", apperr.Invalid("phone authorization failed")
	}
	ticket, terr = s.tokens.IssueRegisterTicket(openID, phone)
	if terr != nil {
		return "", "", apperr.Internal(terr)
	}
	return maskPhone(phone), ticket, nil
}

// RegisterAvatar uploads a first-time user's chosen avatar during registration
// (authorized by the register ticket, since no member/session exists yet) and
// returns its public https URL, which the client then submits to /register.
func (s *Service) RegisterAvatar(ctx context.Context, registerTicket string, r io.Reader, size int64, contentType string) (string, error) {
	registerTicket, validationErr := inputvalidation.OpaqueToken("注册凭证", registerTicket, 4096)
	if validationErr != nil {
		return "", apperr.Invalid(validationErr.Error())
	}
	openID, _, err := s.tokens.ParseRegisterTicket(registerTicket)
	if err != nil || openID == "" {
		return "", apperr.Unauthenticated("invalid or expired register ticket")
	}
	if s.avatars == nil {
		return "", apperr.Internal(fmt.Errorf("avatar uploader not configured"))
	}
	return s.avatars.UploadAvatar(ctx, r, size, contentType)
}

// maskPhone returns a masked phone number for display (e.g. "13812345678" -> "138****5678").
func maskPhone(phone string) string {
	if len(phone) != 11 {
		return phone // fallback for non-standard formats
	}
	return phone[:3] + "****" + phone[7:]
}

// issueMemberSession mints a mini access/refresh pair for an active member.
func (s *Service) issueMemberSession(member Member, isNew bool) (LoginResponse, error) {
	if member.Status != StatusActive {
		return LoginResponse{}, apperr.Forbidden("member is disabled")
	}
	pair, err := s.tokens.Issue(authn.Identity{
		SubjectID:    member.ID,
		SubjectType:  authn.SubjectMember,
		Role:         authn.RoleMember,
		Audience:     authn.AudienceMini,
		TokenVersion: member.TokenVersion,
	})
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}
	return LoginResponse{
		Token: pair, Profile: memberProfile(member), IsNew: isNew,
		SubjectType: authn.SubjectMember,
	}, nil
}

func (s *Service) issueStaffSession(member Member, staff Staff, isNew bool) (LoginResponse, error) {
	if member.Status != StatusActive || staff.Status != StatusActive {
		return LoginResponse{}, apperr.Forbidden("staff account is disabled")
	}
	pair, err := s.tokens.Issue(authn.Identity{
		// Staff retains the member id as the subject because the mini application
		// also exposes the member's wallet/orders/profile to the same identity.
		SubjectID: member.ID, SubjectType: authn.SubjectStaff, Role: authn.RoleStaff,
		Audience: authn.AudienceMini, StoreID: staff.StoreID, TokenVersion: staff.TokenVersion,
	})
	if err != nil {
		return LoginResponse{}, apperr.Internal(err)
	}
	return LoginResponse{
		Token: pair, Profile: memberProfile(member), IsNew: isNew,
		SubjectType: authn.SubjectStaff, StoreID: staff.StoreID,
	}, nil
}

// createMember inserts a new member at registration time with the submitted
// avatar, nickname, gender, and WeChat-resolved phone, assigning a unique 6-digit
// numeric invite code. Uniqueness is enforced by the members.invite_code UNIQUE
// constraint; a duplicate-key collision retries with a new code (a handful of
// attempts is ample given the 1e6 code space and small membership).
func (s *Service) createMember(ctx context.Context, openID, avatarURL, nickname, gender, phone string, invitedByMemberID *int64) (int64, error) {
	if nickname == "" {
		nickname = "会员"
	}
	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		id, err := s.members.Create(ctx, Member{
			WeChatOpenID:      openID,
			AvatarURL:         avatarURL,
			Nickname:          nickname,
			Gender:            gender,
			Phone:             phone,
			InviteCode:        newInviteCode(),
			InvitedByMemberID: invitedByMemberID,
			ProfileCompleted:  true,
		})
		if err == nil {
			return id, nil
		}
		if platdb.IsDuplicate(err) {
			continue // invite_code collided — retry with a fresh code
		}
		return 0, err
	}
	return 0, apperr.Internal(fmt.Errorf("could not allocate a unique invite code after %d attempts", maxAttempts))
}

// AdminLogin authenticates a super_admin against the admin console.
func (s *Service) AdminLogin(ctx context.Context, username, password string) (LoginResponse, error) {
	account, err := s.authenticate(ctx, username, password)
	if err != nil {
		return LoginResponse{}, err
	}
	if account.Role != string(authn.RoleSuperAdmin) {
		return LoginResponse{}, apperr.Forbidden("admin console requires super_admin")
	}
	pair, err := s.issueAccountToken(account, authn.AudienceAdmin)
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{Token: pair, Profile: accountProfile(account)}, nil
}

// StoreLogin authenticates a store_admin/cashier against the store console. The
// store scope comes from the account binding, never from the request.
func (s *Service) StoreLogin(ctx context.Context, username, password string) (LoginResponse, error) {
	account, err := s.authenticate(ctx, username, password)
	if err != nil {
		return LoginResponse{}, err
	}
	if account.Role != string(authn.RoleStoreAdmin) && account.Role != string(authn.RoleCashier) {
		return LoginResponse{}, apperr.Forbidden("store console requires store_admin or cashier")
	}
	if account.StoreID <= 0 {
		return LoginResponse{}, apperr.New(apperr.CodeStoreScopeRequired, "account has no store binding")
	}
	pair, err := s.issueAccountToken(account, authn.AudienceStore)
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{Token: pair, Profile: accountProfile(account)}, nil
}

// Refresh validates a refresh token for the audience and reissues a pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string, audience authn.Audience) (authn.TokenPair, error) {
	refreshToken, validationErr := inputvalidation.OpaqueToken("刷新凭证", refreshToken, 4096)
	if validationErr != nil {
		return authn.TokenPair{}, apperr.Unauthenticated("登录状态无效")
	}
	claims, err := s.tokens.Parse(refreshToken, audience)
	if err != nil || claims.Kind != authn.TokenRefresh {
		return authn.TokenPair{}, apperr.Unauthenticated("invalid refresh token")
	}
	if audience == authn.AudienceMini {
		member, err := s.members.GetByID(ctx, claims.SubjectID())
		if err != nil {
			return authn.TokenPair{}, err
		}
		if member.Status != StatusActive {
			return authn.TokenPair{}, apperr.Unauthenticated("session expired")
		}
		if claims.SubjectType == authn.SubjectPreMember {
			if member.TokenVersion != claims.TokenVersion {
				return authn.TokenPair{}, apperr.Unauthenticated("预注册状态已失效，请重新登录")
			}
			return s.tokens.Issue(authn.Identity{
				SubjectID: member.ID, SubjectType: authn.SubjectPreMember, Role: authn.RolePreMember,
				Audience: authn.AudienceMini, TokenVersion: member.TokenVersion,
			})
		}
		if claims.SubjectType == authn.SubjectStaff {
			if s.staff == nil {
				return authn.TokenPair{}, apperr.Unauthenticated("session expired")
			}
			staff, err := s.staff.GetByMemberID(ctx, member.ID)
			if err != nil {
				if apperr.From(err).Code == apperr.CodeNotFound {
					return authn.TokenPair{}, apperr.Unauthenticated("session expired")
				}
				return authn.TokenPair{}, err
			}
			if staff.Status != StatusActive || staff.TokenVersion != claims.TokenVersion {
				return authn.TokenPair{}, apperr.Unauthenticated("session expired")
			}
			resp, err := s.issueStaffSession(member, staff, false)
			return resp.Token, err
		}
		if member.TokenVersion != claims.TokenVersion {
			return authn.TokenPair{}, apperr.Unauthenticated("session expired")
		}
		return s.tokens.Issue(authn.Identity{
			SubjectID: member.ID, SubjectType: authn.SubjectMember, Role: authn.RoleMember,
			Audience: authn.AudienceMini, TokenVersion: member.TokenVersion,
		})
	}
	account, err := s.accounts.GetByID(ctx, claims.SubjectID())
	if err != nil {
		return authn.TokenPair{}, err
	}
	if account.TokenVersion != claims.TokenVersion || account.Status != StatusActive {
		return authn.TokenPair{}, apperr.Unauthenticated("session expired")
	}
	return s.issueAccountToken(account, audience)
}

// MemberProfile returns the member profile payload, including the member's
// current VIP tier (and its banner URL) when a tier resolver is wired.
func (s *Service) MemberProfile(ctx context.Context, id int64) (MemberProfile, error) {
	member, err := s.members.GetByID(ctx, id)
	if err != nil {
		return MemberProfile{}, err
	}
	profile := memberProfile(member)
	if s.tiers != nil {
		tier, err := s.tiers.CurrentTier(ctx, id)
		if err != nil {
			return MemberProfile{}, err
		}
		profile.VipTier = tier
	}
	return profile, nil
}

// AccountProfile returns the account profile payload.
func (s *Service) AccountProfile(ctx context.Context, id int64) (AccountProfile, error) {
	account, err := s.accounts.GetByID(ctx, id)
	if err != nil {
		return AccountProfile{}, err
	}
	return accountProfile(account), nil
}

// LogoutMini invalidates the current mini identity without invalidating the
// member's other role: staff tokens bump staff_accounts, member tokens bump
// members.
func (s *Service) LogoutMini(ctx context.Context, subject authn.SubjectType, id int64) error {
	if subject == authn.SubjectStaff {
		if s.staff == nil {
			return apperr.NotFound("staff account not found")
		}
		return s.staff.BumpTokenVersionByMemberID(ctx, id)
	}
	return s.members.BumpTokenVersion(ctx, id)
}

// LogoutAccount invalidates all of an account's tokens.
func (s *Service) LogoutAccount(ctx context.Context, id int64) error {
	return s.accounts.BumpTokenVersion(ctx, id)
}

func (s *Service) authenticate(ctx context.Context, username, password string) (Account, error) {
	account, err := s.accounts.GetByUsername(ctx, strings.TrimSpace(username))
	if apperr.From(err) != nil && apperr.From(err).Code == apperr.CodeNotFound {
		return Account{}, apperr.Unauthenticated("invalid credentials")
	}
	if err != nil {
		return Account{}, err
	}
	if account.Status != StatusActive {
		return Account{}, apperr.Forbidden("account is disabled")
	}
	if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)) != nil {
		return Account{}, apperr.Unauthenticated("invalid credentials")
	}
	return account, nil
}

// VerifyAccountPassword re-authenticates a back-office account before a
// high-risk action such as issuing a refund.
func (s *Service) VerifyAccountPassword(ctx context.Context, accountID int64, password string) error {
	if strings.TrimSpace(password) == "" {
		return apperr.Invalid("请输入管理员登录密码")
	}
	account, err := s.accounts.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Status != StatusActive {
		return apperr.Forbidden("管理员账号已停用")
	}
	if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)) != nil {
		return apperr.Forbidden("管理员登录密码错误")
	}
	return nil
}

// VerifyStoreAdminPassword confirms a high-risk store action using any active
// store administrator account bound to the JWT-scoped store. Cashier passwords
// never satisfy this check.
func (s *Service) VerifyStoreAdminPassword(ctx context.Context, storeID int64, password string) error {
	if storeID <= 0 {
		return apperr.Invalid("门店范围无效")
	}
	if strings.TrimSpace(password) == "" {
		return apperr.Invalid("请输入门店管理员登录密码")
	}
	accounts, err := s.accounts.ListActiveStoreAdminsByStoreID(ctx, storeID)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)) == nil {
			return nil
		}
	}
	return apperr.Forbidden("门店管理员登录密码错误")
}

func (s *Service) issueAccountToken(account Account, audience authn.Audience) (authn.TokenPair, error) {
	pair, err := s.tokens.Issue(authn.Identity{
		SubjectID:    account.ID,
		SubjectType:  subjectForRole(account.Role),
		Role:         authn.Role(account.Role),
		Audience:     audience,
		StoreID:      account.StoreID,
		TokenVersion: account.TokenVersion,
	})
	if err != nil {
		return authn.TokenPair{}, apperr.Internal(err)
	}
	return pair, nil
}

func subjectForRole(role string) authn.SubjectType {
	switch authn.Role(role) {
	case authn.RoleSuperAdmin:
		return authn.SubjectSuperAdmin
	case authn.RoleStoreAdmin:
		return authn.SubjectStoreAdmin
	case authn.RoleCashier:
		return authn.SubjectCashier
	case authn.RoleStaff:
		return authn.SubjectStaff
	default:
		return authn.SubjectMember
	}
}

func memberProfile(m Member) MemberProfile {
	return MemberProfile{
		ID:           m.ID,
		Nickname:     m.Nickname,
		Gender:       m.Gender,
		AvatarURL:    m.AvatarURL,
		MemberNo:     formatMemberNo(m.ID),
		Phone:        m.Phone,
		InviteCode:   m.InviteCode,
		InviterBound: m.InvitedByMemberID != nil,
		Status:       m.Status,
	}
}

// formatMemberNo derives a stable, display-only membership card number from the
// member id (e.g. id 2 -> "6272 0000 002"). It is deterministic and backend
// owned; the mini home card renders it as "NO.<memberNo>".
func formatMemberNo(id int64) string {
	n := fmt.Sprintf("6272%07d", id) // "6272" prefix + 7-digit zero-padded id
	return n[0:4] + " " + n[4:8] + " " + n[8:11]
}

func accountProfile(a Account) AccountProfile {
	return AccountProfile{ID: a.ID, Username: a.Username, DisplayName: a.DisplayName, Role: a.Role, StoreID: a.StoreID, Status: a.Status}
}
