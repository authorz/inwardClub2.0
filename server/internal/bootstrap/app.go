// Package bootstrap wires configuration, infrastructure, adapters, modules and
// routing into a runnable application. It is the single composition root; no
// module reaches for global singletons.
package bootstrap

import (
	"context"
	"log/slog"
	"os"

	"github.com/inwardclub/server/internal/modules/activity"
	"github.com/inwardclub/server/internal/modules/admin"
	"github.com/inwardclub/server/internal/modules/asset"
	"github.com/inwardclub/server/internal/modules/auth"
	"github.com/inwardclub/server/internal/modules/catalog"
	"github.com/inwardclub/server/internal/modules/coupon"
	"github.com/inwardclub/server/internal/modules/diagnostics"
	"github.com/inwardclub/server/internal/modules/franchise"
	"github.com/inwardclub/server/internal/modules/member"
	"github.com/inwardclub/server/internal/modules/order"
	"github.com/inwardclub/server/internal/modules/payment"
	"github.com/inwardclub/server/internal/modules/printer"
	"github.com/inwardclub/server/internal/modules/referral"
	"github.com/inwardclub/server/internal/modules/reporting"
	"github.com/inwardclub/server/internal/modules/reservation"
	"github.com/inwardclub/server/internal/modules/store"
	"github.com/inwardclub/server/internal/modules/systemsetting"
	"github.com/inwardclub/server/internal/modules/tournament"
	"github.com/inwardclub/server/internal/modules/wallet"
	"github.com/inwardclub/server/internal/platform/audit"
	"github.com/inwardclub/server/internal/platform/authn"
	"github.com/inwardclub/server/internal/platform/config"
	"github.com/inwardclub/server/internal/platform/credentialcrypto"
	platdb "github.com/inwardclub/server/internal/platform/db"
)

// App holds the built dependencies and exposes the HTTP router.
type App struct {
	cfg *config.Config
	log *slog.Logger
	db  *platdb.DB

	tokens        *authn.Manager
	auditRecorder *audit.Recorder

	// Per-audience token-version checkers make the access-token middleware reject
	// tokens invalidated by a logout/disable (mini reads members or staff
	// bindings, admin/store read accounts).
	memberVersions  authn.TokenVersionChecker
	accountVersions authn.TokenVersionChecker

	authHandler       *auth.Handler
	assetHandler      *asset.Handler
	storeHandler      *store.Handler
	catalogHandler    *catalog.Handler
	activityHandler   *activity.Handler
	tournamentHandler *tournament.Handler
	walletHandler     *wallet.Handler
	paymentHandler    *payment.Handler
	credentialHandler *credentialcrypto.Handler

	paymentStoreHandler        *payment.StoreHandler
	paymentAdminHandler        *payment.AdminHandler
	activityStoreHandler       *activity.StoreHandler
	pointReviewSettingsHandler *activity.PointReviewSettingsHandler
	globalSettingsHandler      *systemsetting.Handler
	storeLowSpendRuleHandler   *systemsetting.StoreLowSpendRuleHandler
	franchiseHandler           *franchise.Handler
	referralHandler            *referral.Handler

	memberHandler      *member.Handler
	reservationHandler *reservation.Handler
	orderHandler       *order.Handler
	couponHandler      *coupon.Handler
	adminHandler       *admin.Handler
	reportingHandler   *reporting.Handler

	storeConsoleHandler       *store.ConsoleHandler
	catalogConsoleHandler     *catalog.ConsoleHandler
	activityConsoleHandler    *activity.ConsoleHandler
	couponConsoleHandler      *coupon.ConsoleHandler
	printerConsoleHandler     *printer.ConsoleHandler
	reservationConsoleHandler *reservation.ConsoleHandler
	orderStoreConsoleHandler  *order.StoreConsoleHandler

	diagnosticsSvc     *diagnostics.Service
	diagnosticsHandler *diagnostics.Handler

	health *healthHandler
}

// Build opens infrastructure and constructs every module. The caller owns the
// context (used for connection setup) and must call Close when done.
func Build(ctx context.Context, cfg *config.Config, log *slog.Logger) (*App, error) {
	database, err := platdb.Open(ctx, cfg.MySQLDSN)
	if err != nil {
		return nil, err
	}

	tokens := authn.NewManager(cfg.JWT.SigningKey, cfg.JWT.Issuer, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	credentialCipher, err := credentialcrypto.New(cfg.CredentialEncryption.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	// Business clock: the configurable "business day" reference shared by the
	// daily sign-in calendar and the leaderboard windows, so neither drifts with
	// the host clock.
	businessClock, err := cfg.BusinessClock()
	if err != nil {
		return nil, err
	}

	// Adapters: fakes keep dev/tests offline. The asset store and WeChat login /
	// phone-resolver client have real implementations wired behind
	// USE_FAKE_ADAPTERS; other real providers are pending (see delivery notes).
	objectStore := buildObjectStore(cfg, log)
	wechatLogin := buildWeChatClient(cfg, log)
	wechatPay, err := buildWeChatPayGateway(cfg, log)
	if err != nil {
		return nil, err
	}
	offlineAcquirer := buildOfflineAcquirer(cfg, log)

	// Asset module.
	assetRepo := asset.NewRepository(database)
	assetSvc := asset.NewService(assetRepo, objectStore, cfg.AppEnv)

	// Auth module. The service is constructed after the member service below so
	// the mini "me" profile can resolve the member's current VIP tier + banner.
	authMembers := auth.NewMemberRepository(database)
	authAccounts := auth.NewAccountRepository(database)
	authStaff := auth.NewStaffRepository(database)

	// Storefront read modules.
	storeSvc := store.NewService(store.NewRepository(database), assetSvc)
	catalogSvc := catalog.NewService(catalog.NewRepository(database), assetSvc)
	activitySvc := activity.NewService(activity.NewRepository(database), assetSvc)
	tournamentSvc := tournament.NewService(tournament.NewRepository(database), assetSvc)
	walletSvc := wallet.NewService(wallet.NewRepository(database))
	walletPointsSvc := wallet.NewPointsService(wallet.NewPointsRepository(database, businessClock))
	paymentRepo := payment.NewStoreRepository(database)
	activityStoreSvc := activity.NewStoreService(
		activity.NewStoreRepository(database), assetSvc, businessClock.Location(),
	)
	pointReviewSettingsSvc := activity.NewPointReviewSettingsService(
		activity.NewPointReviewSettingsRepository(database),
	)
	globalSettingsSvc := systemsetting.NewService(systemsetting.NewRepository(database))
	cloudPrinter := printer.SelectWithSettings(globalSettingsSvc, cfg.Xpyun, !cfg.PrinterReal())
	storeLowSpendRuleSvc := systemsetting.NewStoreLowSpendRuleService(
		systemsetting.NewStoreLowSpendRuleRepository(database),
	)
	franchiseSvc := franchise.NewService(franchise.NewRepository(database), globalSettingsSvc)
	referralSvc := referral.NewService(referral.NewRepository(database))

	// Member/order/reservation/coupon/console modules. Phone binding exchanges a
	// WeChat phone code via the WeChat client (fake offline, real once wired).
	memberSvc := member.NewService(
		member.NewRepository(database, businessClock),
		assetSvc,
		phoneResolverAdapter{wechatLogin},
		globalSettingsSvc,
	)
	authSvc := auth.NewService(
		tokens, wechatLogin, authMembers, authAccounts,
		memberTierAdapter{memberSvc}, assetSvc, authStaff,
	)
	paymentStoreSvc := payment.NewStoreService(
		paymentRepo,
		wechatPay,
		authSvc,
		cfg.WeChatPayAmountOverrideCent(),
	)
	paymentAdminSvc := payment.NewAdminService(
		paymentRepo,
		wechatPay,
		offlineAcquirer,
		authSvc,
		cfg.WeChatPayAmountOverrideCent(),
	)
	reservationSvc := reservation.NewService(
		reservation.NewRepository(database), assetSvc, businessClock.Location(), storeLowSpendRuleSvc,
	)
	couponSvc := coupon.NewService(coupon.NewRepository(database), catalogSvc)
	wechatPayAmountOverrideCent := cfg.WeChatPayAmountOverrideCent()
	if wechatPayAmountOverrideCent > 0 {
		log.Warn("payment debug mode enabled", "wechatAmountCent", wechatPayAmountOverrideCent)
	}
	orderSvc := order.NewService(
		order.NewRepository(database),
		wechatPay,
		memberOpenIDAdapter{authMembers},
		assetSvc,
		wechatPayAmountOverrideCent,
	)
	adminSvc := admin.NewService(
		admin.NewRepository(database), storeProfileAdapter{storeSvc}, walletSvc, assetSvc,
	)
	reportingSvc := reporting.NewService(reporting.NewRepository(database))

	// Console CRUD services (admin + store) sharing the module repositories.
	storeConsoleSvc := store.NewConsoleService(store.NewRepository(database), assetSvc, authSvc)
	catalogConsoleSvc := catalog.NewConsoleService(catalog.NewConsoleRepository(database), assetSvc)
	activityConsoleSvc := activity.NewConsoleService(activity.NewConsoleRepository(database), assetSvc)
	couponConsoleSvc := coupon.NewConsoleService(coupon.NewConsoleRepository(database))
	printerConsoleSvc := printer.NewConsoleService(printer.NewRepository(database), cloudPrinter)
	printJobRepo := printer.NewJobRepository(database)
	reservationConsoleSvc := reservation.NewConsoleService(reservation.NewConsoleRepository(database))
	orderStoreConsoleSvc := order.NewStoreConsoleService(order.NewStoreConsoleRepository(database), paymentAdminSvc, authSvc, assetSvc)

	// Diagnostics: durable admin error-events feed backed by the error_events
	// table (db/migrations 00018); the Capture middleware persists 5xx failures.
	diagnosticsSvc := diagnostics.NewService(diagnostics.NewRepository(database), log)

	return &App{
		cfg:               cfg,
		log:               log,
		db:                database,
		tokens:            tokens,
		auditRecorder:     audit.NewRecorder(database),
		memberVersions:    auth.NewMemberTokenVersions(authMembers, authStaff),
		accountVersions:   auth.NewAccountTokenVersions(authAccounts),
		authHandler:       auth.NewHandler(authSvc),
		assetHandler:      asset.NewHandler(assetSvc),
		storeHandler:      store.NewHandler(storeSvc),
		catalogHandler:    catalog.NewHandler(catalogSvc),
		activityHandler:   activity.NewHandler(activitySvc),
		tournamentHandler: tournament.NewHandler(tournamentSvc),
		walletHandler:     wallet.NewHandler(walletSvc, walletPointsSvc),
		paymentHandler:    payment.NewHandler(wechatPay, offlineAcquirer, payment.NewSettlementService(payment.NewSettlementRepository(database))),

		paymentStoreHandler:        payment.NewStoreHandler(paymentStoreSvc, credentialCipher),
		paymentAdminHandler:        payment.NewAdminHandler(paymentAdminSvc, credentialCipher),
		activityStoreHandler:       activity.NewStoreHandler(activityStoreSvc),
		pointReviewSettingsHandler: activity.NewPointReviewSettingsHandler(pointReviewSettingsSvc),
		globalSettingsHandler:      systemsetting.NewHandler(globalSettingsSvc),
		storeLowSpendRuleHandler:   systemsetting.NewStoreLowSpendRuleHandler(storeLowSpendRuleSvc),
		franchiseHandler:           franchise.NewHandler(franchiseSvc),
		referralHandler:            referral.NewHandler(referralSvc),

		memberHandler:      member.NewHandler(memberSvc),
		reservationHandler: reservation.NewHandler(reservationSvc),
		orderHandler:       order.NewHandler(orderSvc),
		couponHandler:      coupon.NewHandler(couponSvc),
		adminHandler:       admin.NewHandler(adminSvc),
		reportingHandler:   reporting.NewHandler(reportingSvc),

		storeConsoleHandler:       store.NewConsoleHandler(storeConsoleSvc, credentialCipher),
		catalogConsoleHandler:     catalog.NewConsoleHandler(catalogConsoleSvc),
		activityConsoleHandler:    activity.NewConsoleHandler(activityConsoleSvc),
		couponConsoleHandler:      coupon.NewConsoleHandler(couponConsoleSvc),
		printerConsoleHandler:     printer.NewConsoleHandler(printerConsoleSvc, printJobRepo),
		reservationConsoleHandler: reservation.NewConsoleHandler(reservationConsoleSvc),
		orderStoreConsoleHandler:  order.NewStoreConsoleHandler(orderStoreConsoleSvc),

		diagnosticsSvc:     diagnosticsSvc,
		diagnosticsHandler: diagnostics.NewHandler(diagnosticsSvc),
		credentialHandler:  credentialcrypto.NewHandler(credentialCipher),

		health: &healthHandler{db: database},
	}, nil
}

// memberOpenIDAdapter satisfies order.MemberDirectory by reading the member's
// WeChat openid from the auth member repository.
type memberOpenIDAdapter struct{ repo auth.MemberRepository }

func (a memberOpenIDAdapter) OpenIDByMemberID(ctx context.Context, memberID int64) (string, error) {
	m, err := a.repo.GetByID(ctx, memberID)
	if err != nil {
		return "", err
	}
	return m.WeChatOpenID, nil
}

// memberTierAdapter satisfies auth.MemberTierResolver by mapping the member
// module's current-tier view onto the auth "me" tier shape. A nil view (member
// unranked) maps to a nil tier so the profile omits it.
type memberTierAdapter struct{ svc *member.Service }

func (a memberTierAdapter) CurrentTier(ctx context.Context, memberID int64) (*auth.MemberVIPTier, error) {
	v, err := a.svc.CurrentTierView(ctx, memberID)
	if err != nil || v == nil {
		return nil, err
	}
	return &auth.MemberVIPTier{
		ID:        v.ID,
		Label:     member.VIPShortLabel(v.Level),
		Level:     v.Level,
		Threshold: v.Threshold,
		IconURL:   v.IconURL,
	}, nil
}

// phoneResolverAdapter satisfies member.PhoneResolver by delegating to the
// WeChat client's phone-code exchange. Swapping in the real WeChat client (in
// place of the fake) is the only change needed for production phone binding.
type phoneResolverAdapter struct{ client auth.WeChatClient }

func (a phoneResolverAdapter) ResolvePhone(ctx context.Context, code string) (string, error) {
	return a.client.GetPhoneNumber(ctx, code)
}

// storeProfileAdapter satisfies admin.StoreProfileProvider by mapping the store
// module's read view onto the console profile view.
type storeProfileAdapter struct{ svc *store.Service }

func (a storeProfileAdapter) StoreProfile(ctx context.Context, storeID int64) (admin.StoreProfileView, error) {
	v, err := a.svc.GetStore(ctx, storeID, store.Geo{})
	if err != nil {
		return admin.StoreProfileView{}, err
	}
	return admin.StoreProfileView{
		ID:                       v.ID,
		Name:                     v.Name,
		LogoURL:                  v.LogoURL,
		Phone:                    v.Phone,
		CustomerServiceQRAssetID: v.CustomerServiceQRAssetID,
		CustomerServiceQRURL:     v.CustomerServiceQRURL,
		Address:                  v.Address,
		BusinessHours:            v.BusinessHours,
		Status:                   v.Status,
	}, nil
}

// buildWeChatClient returns the real WeChat mini-program client (login + phone
// resolver) when fakes are disabled. The fake keeps dev/tests offline; the real
// client needs the mini app id/secret, which config.Validate requires when
// USE_FAKE_ADAPTERS=false.
func buildWeChatClient(cfg *config.Config, log *slog.Logger) auth.WeChatClient {
	if !cfg.WeChatLoginReal() {
		// DEV_FAKE_OPENID lets a developer simulate a specific member in DevTools;
		// empty pins to a stable default so repeated logins hit the same member.
		return auth.NewFakeWeChatClient(os.Getenv("DEV_FAKE_OPENID"))
	}
	log.Info("using real wechat mini-program client", slog.String("appId", cfg.WeChat.MiniAppID))
	return auth.NewWeChatClient(cfg.WeChat.MiniAppID, cfg.WeChat.MiniAppSecret)
}

// buildWeChatPayGateway returns the real WeChat Pay v3 gateway (JSAPI prepay,
// refund, notify verify/decrypt) when fakes are disabled. The fake keeps
// dev/tests offline; the real client loads merchant/platform keys from disk, so
// it can fail at startup on misconfiguration (config.Validate requires the pay
// credentials when USE_FAKE_ADAPTERS=false).
func buildWeChatPayGateway(cfg *config.Config, log *slog.Logger) (payment.WeChatPayGateway, error) {
	if !cfg.WeChatPayReal() {
		return payment.NewFakeWeChatPayGateway(), nil
	}
	log.Info("using real wechat pay gateway", slog.String("mchId", cfg.WeChat.PayMchID))
	return payment.NewWeChatPayClient(cfg.WeChat)
}

// buildObjectStore uses the real Qiniu store whenever Qiniu credentials are
// configured, independent of USE_FAKE_ADAPTERS — so assets can be real while the
// offline acquirer/printer stay faked. It falls back to the in-process fake when
// no credentials are present (CI / offline dev).
func buildObjectStore(cfg *config.Config, log *slog.Logger) asset.ObjectStore {
	if cfg.Qiniu.AccessKey == "" || cfg.Qiniu.SecretKey == "" || cfg.Qiniu.Bucket == "" {
		return asset.NewFakeObjectStore(cfg.Qiniu.Bucket, cfg.Qiniu.PublicDomain)
	}
	log.Info("using real qiniu object store", slog.String("bucket", cfg.Qiniu.Bucket))
	return asset.NewQiniuObjectStore(cfg.Qiniu)
}

// buildOfflineAcquirer returns the real HTTP acquirer client when fakes are
// disabled. The fake keeps dev/tests offline; the real client needs the acquirer
// merchant credentials + base URL, which config.Validate requires when
// USE_FAKE_ADAPTERS=false.
func buildOfflineAcquirer(cfg *config.Config, log *slog.Logger) payment.OfflineAcquirer {
	if cfg.UseFakeAdapters {
		return payment.NewFakeOfflineAcquirer()
	}
	log.Info("using real offline acquirer", slog.String("provider", cfg.Offline.Provider))
	return payment.NewHTTPAcquirer(cfg.Offline)
}

// Close releases infrastructure connections.
func (a *App) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
