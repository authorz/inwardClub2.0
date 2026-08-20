package bootstrap

import (
	"github.com/gin-gonic/gin"

	"github.com/inwardclub/server/internal/platform/authn"
	"github.com/inwardclub/server/internal/platform/httpx"
	"github.com/inwardclub/server/internal/platform/idempotency"
	"github.com/inwardclub/server/internal/platform/storescope"
)

// Router builds the gin engine with all middleware and routes.
func (a *App) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Local-development CORS for the admin/store console Vite dev servers. Never
	// registered in production, where the consoles are served same-origin.
	if a.cfg.AppEnv != "production" {
		r.Use(httpx.DevCORS())
	}
	r.Use(httpx.RequestID(), httpx.AccessLog(a.log), a.diagnosticsSvc.Capture(), httpx.Recovery(a.log))

	miniAuth := authn.NewMiddleware(a.tokens, authn.AudienceMini, authn.WithTokenVersions(a.memberVersions))
	adminAuth := authn.NewMiddleware(a.tokens, authn.AudienceAdmin, authn.WithTokenVersions(a.accountVersions))
	storeAuth := authn.NewMiddleware(a.tokens, authn.AudienceStore, authn.WithTokenVersions(a.accountVersions))

	a.registerInternal(r)
	a.registerMini(r, miniAuth)
	a.registerAdmin(r, adminAuth)
	a.registerStore(r, storeAuth)
	return r
}

// registerInternal wires health checks and provider callbacks (no JWT).
func (a *App) registerInternal(r *gin.Engine) {
	g := r.Group("/internal")
	g.GET("/health", a.health.Live)
	g.GET("/ready", a.health.Ready)
	g.POST("/qiniu/upload-callback", a.assetHandler.Callback)
	g.POST("/payments/wechat/notify", a.paymentHandler.WeChatNotify)
	g.POST("/payments/offline-acquirer/notify", a.paymentHandler.OfflineNotify)
}

// registerMini wires the mini-program API: public reads plus member-authed writes.
func (a *App) registerMini(r *gin.Engine, mw *authn.Middleware) {
	g := r.Group("/api/v2/mini")

	// Public reads (no auth).
	g.GET("/stores", a.storeHandler.List)
	g.GET("/stores/:storeID", a.storeHandler.Detail)
	g.GET("/stores/:storeID/catalog/categories", a.catalogHandler.Categories)
	g.GET("/stores/:storeID/catalog/items", a.catalogHandler.Items)
	g.GET("/stores/:storeID/activities", a.activityHandler.ListForStore)
	g.GET("/stores/:storeID/tournament-events", a.tournamentHandler.PublicList)
	g.GET("/stores/:storeID/tables", a.reservationHandler.Tables)
	g.GET("/stores/:storeID/seats", a.reservationHandler.Seats)
	g.GET("/activities", a.activityHandler.List)
	g.GET("/activities/:activityID", a.activityHandler.Detail)
	g.GET("/tournament-events/:eventID", a.tournamentHandler.PublicDetail)
	g.GET("/membership-tiers", a.memberHandler.MembershipTiers)
	g.GET("/recharge-products", a.memberHandler.RechargeProducts)
	g.GET("/rankings", a.memberHandler.Rankings)
	g.GET("/franchise-inquiries/config", a.franchiseHandler.Config)
	g.POST("/franchise-inquiries", mw.OptionalAuth(authn.SubjectMember), a.franchiseHandler.Create)

	// Auth (login/register/refresh are public; logout/me require a token).
	// Register is authorized by the ticket MiniLogin returns for a new user.
	g.POST("/auth/wechat/login", a.authHandler.MiniLogin)
	g.POST("/auth/wechat/pre-register", a.authHandler.MiniPreRegister)
	g.POST("/auth/wechat/register", a.authHandler.MiniRegister)
	g.POST("/auth/wechat/register-avatar", a.authHandler.RegisterAvatar)
	g.POST("/auth/wechat/phone-mask", a.authHandler.GetPhoneMask)
	g.POST("/auth/refresh", a.authHandler.MiniRefresh)

	p := g.Group("", mw.RequireAuth(authn.SubjectMember, authn.SubjectStaff))
	p.POST("/auth/logout", a.authHandler.MiniLogout)
	p.GET("/me", a.authHandler.MiniMe)
	p.PATCH("/me", a.memberHandler.UpdateProfile)
	p.POST("/me/phone-bindings", idempotency.Optional(), a.memberHandler.BindPhone)
	p.POST("/assets/upload-credentials", a.assetHandler.UploadCredentials)
	p.GET("/wallet", a.walletHandler.Get)
	p.GET("/wallet/ledger", a.walletHandler.Ledger)
	p.GET("/sign-ins/status", a.walletHandler.SignInStatus)
	p.GET("/invitations", a.memberHandler.ListInvitations)
	p.GET("/invitation-reward-config", a.referralHandler.Config)
	p.GET("/coupons", a.couponHandler.List)
	p.GET("/coupon-redemptions/eligible-items", a.couponHandler.EligibleItems)
	p.GET("/coupon-redemptions", a.couponHandler.ListRedemptions)
	p.GET("/coupon-redemptions/:id", a.couponHandler.GetRedemption)

	// Reservations also accept a restricted OpenID-only pre-registration
	// identity.
	reservationRead := g.Group("", mw.RequireAuth(authn.SubjectMember, authn.SubjectStaff, authn.SubjectPreMember))
	reservationRead.GET("/reservations", a.reservationHandler.ListReservations)
	reservationRead.GET("/reservations/:reservationID", a.reservationHandler.GetReservation)
	reservationWrite := g.Group("", mw.RequireAuth(authn.SubjectMember, authn.SubjectStaff, authn.SubjectPreMember), idempotency.Require())
	reservationWrite.POST("/reservations", a.reservationHandler.CreateReservation)
	reservationWrite.POST("/reservations/:reservationID/cancel", a.reservationHandler.CancelReservation)

	// OpenID-only identities may buy an activity with WeChat and read the
	// resulting order/tickets. Coin payment and all other member assets remain
	// restricted to completed member/staff identities.
	activityBuyer := g.Group("", mw.RequireAuth(authn.SubjectMember, authn.SubjectStaff, authn.SubjectPreMember))
	activityBuyer.GET("/activity-orders", a.orderHandler.ListActivityOrders)
	activityBuyer.GET("/activity-orders/:orderID", a.orderHandler.GetActivityOrder)
	activityBuyer.GET("/tickets", a.orderHandler.ListTickets)
	activityBuyerIdem := g.Group("", mw.RequireAuth(authn.SubjectMember, authn.SubjectStaff, authn.SubjectPreMember), idempotency.Require())
	activityBuyerIdem.POST("/activity-orders", a.orderHandler.CreateActivityOrder)
	activityBuyerIdem.POST("/payment-orders/:paymentOrderID/wechat-jsapi", a.orderHandler.WeChatJSAPI)

	// Money/asset-mutating endpoints require an Idempotency-Key.
	idem := g.Group("", mw.RequireAuth(authn.SubjectMember, authn.SubjectStaff), idempotency.Require())
	idem.POST("/sign-ins", a.walletHandler.SignIn)
	idem.POST("/point-savings", a.walletHandler.SavePoints)
	idem.POST("/point-withdrawals", a.walletHandler.WithdrawPoints)
	idem.POST("/invitations/bind", a.memberHandler.BindInvitation)
	idem.POST("/food-orders", a.orderHandler.CreateFoodOrder)
	idem.POST("/recharge-orders", a.orderHandler.CreateRechargeOrder)
	idem.POST("/waitlist-entries", a.reservationHandler.CreateWaitlistEntry)
	idem.POST("/coupon-redemptions", a.couponHandler.Redeem)
	idem.POST("/payment-orders/:paymentOrderID/pay-by-coin", a.orderHandler.PayByCoin)

	// Order reads.
	p.GET("/food-orders", a.orderHandler.ListFoodOrders)
	p.GET("/food-orders/:orderID", a.orderHandler.GetFoodOrder)
	p.GET("/recharge-orders", a.orderHandler.ListRechargeOrders)
	p.GET("/recharge-orders/:orderID", a.orderHandler.GetRechargeOrder)

	// Mini-program staff operations use the mini audience with a staff-only
	// subject. Store scope comes from the bound staff account in the JWT.
	staff := g.Group(
		"/staff",
		mw.RequireAuth(authn.SubjectStaff),
		storescope.Inject(),
	)
	staff.GET("/home", a.activityStoreHandler.StaffHome)
	staff.GET("/point-savings", a.activityStoreHandler.ListPointSavings)
	staff.GET("/point-savings/:requestID", a.activityStoreHandler.GetPointSaving)
	staff.GET("/activities/today", a.activityStoreHandler.TodayActivities)
	staff.GET("/operations/today", a.activityStoreHandler.StaffTodayOperations)
	staff.GET("/verifications", a.activityStoreHandler.ListVerifications)

	staffIdem := g.Group(
		"/staff",
		mw.RequireAuth(authn.SubjectStaff),
		storescope.Inject(),
		idempotency.Require(),
	)
	staffIdem.POST("/tickets/verify", a.activityStoreHandler.VerifyTicket)
	staffIdem.POST("/point-savings/:requestID/review", a.activityStoreHandler.ReviewPointSaving)
}

// registerAdmin wires the headquarters console (aud=admin, super_admin only).
func (a *App) registerAdmin(r *gin.Engine, mw *authn.Middleware) {
	g := r.Group("/api/v2/admin")
	g.POST("/auth/login", a.authHandler.AdminLogin)
	g.POST("/auth/refresh", a.authHandler.AdminRefresh)

	p := g.Group("", mw.RequireAuth(authn.SubjectSuperAdmin))
	p.GET("/auth/me", a.authHandler.AdminMe)
	p.POST("/auth/logout", a.authHandler.AccountLogout)
	p.GET("/auth/password-encryption-key", a.credentialHandler.PublicKey)
	p.POST("/assets/upload-credentials", a.assetHandler.UploadCredentials)

	// Representative admin resources (business logic lands in later milestones).
	p.GET("/stores", a.adminHandler.Stores)
	p.GET("/catalog/items", a.catalogConsoleHandler.Items)
	p.GET("/coupon-templates", a.adminHandler.CouponTemplates)
	p.GET("/activities", a.adminHandler.Activities)
	p.GET("/orders", a.adminHandler.Orders)
	p.GET("/payment-transactions", a.adminHandler.PaymentTransactions)
	p.GET("/payment-orders", a.paymentAdminHandler.ListPaymentOrders)
	p.GET("/payment-orders/:paymentOrderID", a.paymentAdminHandler.GetPaymentOrder)
	p.GET("/payment-channel-settings", a.paymentAdminHandler.PaymentChannelSettings)
	p.PUT("/payment-channel-settings", a.paymentAdminHandler.UpdatePaymentChannelSettings)
	p.GET("/point-review-settings", a.pointReviewSettingsHandler.Get)
	p.PUT("/point-review-settings", a.pointReviewSettingsHandler.Update)
	p.GET("/global-settings", a.globalSettingsHandler.Get)
	p.PUT("/global-settings", a.globalSettingsHandler.Update)
	p.GET("/store-low-spend-rules", a.storeLowSpendRuleHandler.AdminList)
	p.PUT("/store-low-spend-rules/:storeID", a.storeLowSpendRuleHandler.AdminUpdate)
	p.DELETE("/store-low-spend-rules/:storeID", a.storeLowSpendRuleHandler.AdminDelete)
	p.GET("/franchise-inquiries", a.franchiseHandler.AdminList)
	p.PATCH("/franchise-inquiries/:inquiryID/status", a.franchiseHandler.AdminUpdateStatus)
	p.GET("/error-events", a.diagnosticsHandler.ListErrorEvents)
	p.GET("/refunds", a.adminHandler.Refunds)
	p.GET("/refund-orders", a.adminHandler.Refunds)
	p.POST("/refunds", idempotency.Require(), a.paymentAdminHandler.CreateRefund)
	p.GET("/members", a.adminHandler.Members)
	p.GET("/member-lookup", a.adminHandler.AdminLookupMember)
	p.GET("/members/:memberID", a.adminHandler.MemberDetail)
	p.GET("/wallet-ledger", a.adminHandler.WalletLedger)
	p.GET("/admin-accounts", a.adminHandler.AdminAccounts)
	p.POST("/admin-accounts", a.adminHandler.CreateAdminAccount)
	p.PATCH("/admin-accounts/:accountID", a.adminHandler.UpdateAdminAccount)
	p.DELETE("/admin-accounts/:accountID", a.adminHandler.DeleteAdminAccount)
	p.POST("/admin-accounts/:accountID/disable", a.adminHandler.DisableAdminAccount)
	p.GET("/store-admin-accounts", a.adminHandler.StoreAdminAccounts)
	p.POST("/store-admin-accounts", a.adminHandler.CreateStoreAdminAccount)
	p.PATCH("/store-admin-accounts/:accountID", a.adminHandler.UpdateStoreAdminAccount)
	p.POST("/store-admin-accounts/:accountID/disable", a.adminHandler.DisableStoreAdminAccount)
	p.GET("/staff-accounts", a.adminHandler.StaffAccounts)
	p.POST("/staff-accounts", a.adminHandler.AdminCreateStaffAccount)
	p.PATCH("/staff-accounts/:staffID", a.adminHandler.AdminUpdateStaffAccount)
	p.POST("/staff-accounts/:staffID/disable", a.adminHandler.AdminDisableStaffAccount)
	p.DELETE("/staff-accounts/:staffID/binding", a.adminHandler.AdminDeleteStaffAccount)
	p.GET("/audit-logs", a.adminHandler.AuditLogs)
	p.GET("/membership-tiers", a.memberHandler.AdminMembershipTiers)
	p.GET("/membership-tiers/:tierID", a.memberHandler.AdminGetMembershipTier)
	p.POST("/membership-tiers", a.memberHandler.CreateMembershipTier)
	p.PATCH("/membership-tiers/:tierID", a.memberHandler.UpdateMembershipTier)
	p.POST("/membership-tiers/:tierID/disable", a.memberHandler.DisableMembershipTier)
	p.GET("/recharge-products", a.memberHandler.AdminRechargeProducts)
	p.GET("/recharge-products/:productID", a.memberHandler.AdminGetRechargeProduct)
	p.POST("/recharge-products", a.memberHandler.CreateRechargeProduct)
	p.PATCH("/recharge-products/:productID", a.memberHandler.UpdateRechargeProduct)
	p.POST("/recharge-products/:productID/disable", a.memberHandler.DisableRechargeProduct)
	p.GET("/rule-definitions", a.adminHandler.RuleDefinitions)
	p.POST("/rule-definitions", a.adminHandler.CreateRuleDefinition)
	p.PATCH("/rule-definitions/:ruleID", a.adminHandler.UpdateRuleDefinition)
	p.POST("/rule-definitions/:ruleID/disable", a.adminHandler.DisableRuleDefinition)
	p.POST("/rule-definitions/:ruleID/publish", a.adminHandler.PublishRuleDefinition)
	p.GET("/reports/overview", a.reportingHandler.Overview)
	p.GET("/reports/revenue", a.reportingHandler.Revenue)
	p.GET("/reports/catalog-items", a.reportingHandler.CatalogItems)
	p.GET("/reports/activities", a.reportingHandler.Activities)
	p.GET("/reports/coupons", a.reportingHandler.Coupons)
	p.GET("/reports/records", a.reportingHandler.Records)
	p.GET("/reports/members", a.reportingHandler.Members)
	p.GET("/reports/reservations", a.reportingHandler.Reservations)
	p.GET("/reports/stores", a.reportingHandler.Stores)

	// Printer devices (cross-store read; optional ?storeId filter).
	p.GET("/printer-devices", a.printerConsoleHandler.AdminList)

	a.registerAdminConsole(p)
}

// registerAdminConsole wires the HQ console CRUD (stores, catalog, activities,
// coupons) onto the authenticated admin group. High-risk coupon entitlement
// actions (grant/void/verify) additionally require an Idempotency-Key.
//
// The full activity console (activity, session and ticket-type CRUD) is mounted;
// session and ticket-type writes additionally require an Idempotency-Key (they
// hang off the idem group below). Catalog variants are mounted under their own
// /catalog/variants/:itemID prefix because the variant handler reads the parent
// id as :itemID, which would collide with the :id wildcard used by the item
// routes if nested under /catalog/items.
func (a *App) registerAdminConsole(p *gin.RouterGroup) {
	// Stores.
	p.POST("/stores", a.storeConsoleHandler.AdminCreateStore)
	p.GET("/stores/:storeID", a.storeConsoleHandler.AdminGetStore)
	p.PATCH("/stores/:storeID", a.storeConsoleHandler.AdminUpdateStore)
	p.DELETE("/stores/:storeID", idempotency.Require(), a.storeConsoleHandler.AdminDeleteStore)
	p.GET("/stores/:storeID/settings", a.storeConsoleHandler.AdminGetSettings)
	p.PUT("/stores/:storeID/settings", a.storeConsoleHandler.AdminUpdateSettings)

	// Tables and seats (cross-store resources; seats always inherit their table's store).
	p.GET("/tables", a.reservationConsoleHandler.ListTables)
	p.GET("/tables/:tableID", a.reservationConsoleHandler.GetTable)
	p.POST("/tables", a.reservationConsoleHandler.CreateTable)
	p.PATCH("/tables/:tableID", a.reservationConsoleHandler.UpdateTable)
	p.DELETE("/tables/:tableID", a.reservationConsoleHandler.DeleteTable)
	p.GET("/seats", a.reservationConsoleHandler.ListSeats)
	p.GET("/seats/:seatID", a.reservationConsoleHandler.GetSeat)
	p.POST("/seats", a.reservationConsoleHandler.CreateSeat)
	p.PATCH("/seats/:seatID", a.reservationConsoleHandler.UpdateSeat)
	p.DELETE("/seats/:seatID", a.reservationConsoleHandler.DeleteSeat)

	// Catalog categories.
	p.GET("/catalog/categories", a.catalogConsoleHandler.Categories)
	p.GET("/catalog/categories/:id", a.catalogConsoleHandler.GetCategory)
	p.POST("/catalog/categories", a.catalogConsoleHandler.CreateCategory)
	p.PUT("/catalog/categories/:id", a.catalogConsoleHandler.UpdateCategory)
	p.DELETE("/catalog/categories/:id", a.catalogConsoleHandler.DeleteCategory)

	// Catalog items (list read already registered as catalogConsoleHandler.Items).
	p.GET("/catalog/items/:id", a.catalogConsoleHandler.GetItem)
	p.POST("/catalog/items", a.catalogConsoleHandler.CreateItem)
	p.PUT("/catalog/items/:id", a.catalogConsoleHandler.UpdateItem)
	p.DELETE("/catalog/items/:id", a.catalogConsoleHandler.DeleteItem)

	// Catalog variants (scoped by :itemID; see prefix note above).
	p.GET("/catalog/variants/:itemID", a.catalogConsoleHandler.Variants)
	p.GET("/catalog/variants/:itemID/:id", a.catalogConsoleHandler.GetVariant)
	p.POST("/catalog/variants/:itemID", a.catalogConsoleHandler.CreateVariant)
	p.PUT("/catalog/variants/:itemID/:id", a.catalogConsoleHandler.UpdateVariant)
	p.DELETE("/catalog/variants/:itemID/:id", a.catalogConsoleHandler.DeleteVariant)

	// Activities (list read already registered as adminHandler.Activities).
	p.GET("/activities/:activityID", a.activityConsoleHandler.ActivityDetail)
	p.POST("/activities", a.activityConsoleHandler.CreateActivity)
	p.PUT("/activities/:activityID", a.activityConsoleHandler.UpdateActivity)
	p.DELETE("/activities/:activityID", a.activityConsoleHandler.DeleteActivity)

	// Activity sessions and ticket types (reads).
	p.GET("/activities/:activityID/sessions", a.activityConsoleHandler.Sessions)
	p.GET("/activities/:activityID/sessions/:sessionID", a.activityConsoleHandler.SessionDetail)
	p.GET("/activities/:activityID/ticket-types", a.activityConsoleHandler.TicketTypes)
	p.GET("/activities/:activityID/ticket-types/:ticketTypeID", a.activityConsoleHandler.TicketTypeDetail)

	// Coupon templates (list read already registered as adminHandler.CouponTemplates).
	p.GET("/coupon-templates/:id", a.couponConsoleHandler.Get)
	p.POST("/coupon-templates", a.couponConsoleHandler.Create)
	p.PUT("/coupon-templates/:id", a.couponConsoleHandler.Update)
	p.DELETE("/coupon-templates/:id", a.couponConsoleHandler.Delete)
	p.POST("/coupon-templates/:id/publish", a.couponConsoleHandler.Publish)
	p.POST("/coupon-templates/:id/disable", a.couponConsoleHandler.Disable)
	p.GET("/coupon-templates/:id/applicable-items", a.couponConsoleHandler.ApplicableItems)

	// High-risk coupon entitlement actions require an Idempotency-Key.
	idem := p.Group("", idempotency.Require())
	idem.POST("/members/:memberID/wallet-adjustments", a.adminHandler.CreateWalletAdjustment)
	idem.POST("/coupon-grants", a.couponConsoleHandler.Grant)
	idem.POST("/coupon-voids", a.couponConsoleHandler.Void)
	idem.POST("/coupon-verifications", a.couponConsoleHandler.Verify)

	// Activity sessions and ticket types (writes).
	idem.POST("/activities/:activityID/sessions", a.activityConsoleHandler.CreateSession)
	idem.PUT("/activities/:activityID/sessions/:sessionID", a.activityConsoleHandler.UpdateSession)
	idem.DELETE("/activities/:activityID/sessions/:sessionID", a.activityConsoleHandler.DeleteSession)
	idem.POST("/activities/:activityID/ticket-types", a.activityConsoleHandler.CreateTicketType)
	idem.PUT("/activities/:activityID/ticket-types/:ticketTypeID", a.activityConsoleHandler.UpdateTicketType)
	idem.DELETE("/activities/:activityID/ticket-types/:ticketTypeID", a.activityConsoleHandler.DeleteTicketType)
}

// registerStore wires the store console (aud=store, store scope from token).
func (a *App) registerStore(r *gin.Engine, mw *authn.Middleware) {
	g := r.Group("/api/v2/store")
	g.POST("/auth/login", a.authHandler.StoreLogin)
	g.POST("/auth/refresh", a.authHandler.StoreRefresh)

	// Every authed store route pins the store scope from the token.
	p := g.Group("", mw.RequireAuth(authn.SubjectStoreAdmin, authn.SubjectCashier), storescope.Inject())
	p.GET("/auth/me", a.authHandler.StoreMe)
	p.POST("/auth/logout", a.authHandler.AccountLogout)
	p.GET("/auth/password-encryption-key", a.credentialHandler.PublicKey)
	p.POST("/assets/upload-credentials", a.assetHandler.UploadCredentials)

	p.GET("/profile", a.storeConsoleHandler.GetOwnProfile)
	p.GET("/catalog/items", a.catalogConsoleHandler.StoreItems)
	p.GET("/activities", a.adminHandler.StoreActivities)
	p.GET("/coupon-templates", a.adminHandler.StoreCouponTemplates)
	p.GET("/orders", a.adminHandler.StoreOrders)
	p.GET("/payment-transactions", a.adminHandler.StorePaymentTransactions)
	p.GET("/payment-orders", a.paymentStoreHandler.ListPaymentOrders)
	p.GET("/payment-orders/:paymentOrderID", a.paymentStoreHandler.GetPaymentOrder)
	p.GET("/refunds", a.adminHandler.StoreRefunds)
	p.GET("/refund-orders", a.adminHandler.StoreRefunds)
	p.GET("/members", a.adminHandler.StoreMembers)
	p.GET("/member-lookup", a.adminHandler.StoreLookupMember)
	p.GET("/members/:memberID", a.adminHandler.StoreMemberDetail)
	p.GET("/wallet-ledger", a.adminHandler.StoreWalletLedger)
	p.GET("/cashiers", a.adminHandler.StoreCashiers)
	p.GET("/staff-accounts", a.adminHandler.StoreStaffAccounts)

	// Store operational reads (offline collection orders, point savings review
	// and verification history).
	p.GET("/offline-collection-orders/:collectionOrderID", a.paymentStoreHandler.GetCollectionOrder)
	p.GET("/staff/home", a.activityStoreHandler.StaffHome)
	p.GET("/point-savings", a.activityStoreHandler.ListPointSavings)
	p.GET("/point-savings/:requestID", a.activityStoreHandler.GetPointSaving)
	p.GET("/verifications", a.activityStoreHandler.ListVerifications)
	p.GET("/reservations", a.reservationHandler.StoreList)
	p.GET("/reservations/:reservationID", a.reservationHandler.StoreGet)

	// Own-store console reports (read models). The reporting handler reads the
	// pinned store scope from the token via storescope, so these reuse the same
	// handler methods as the admin dashboard but can only ever see this store.
	p.GET("/reports/overview", a.reportingHandler.Overview)
	p.GET("/reports/revenue", a.reportingHandler.Revenue)
	p.GET("/reports/catalog-items", a.reportingHandler.CatalogItems)
	p.GET("/reports/activities", a.reportingHandler.Activities)
	p.GET("/reports/coupons", a.reportingHandler.Coupons)

	// Store money/verification endpoints require an Idempotency-Key.
	idem := g.Group("", mw.RequireAuth(authn.SubjectStoreAdmin, authn.SubjectCashier), storescope.Inject(), idempotency.Require())
	idem.POST("/offline-collection-orders", a.paymentStoreHandler.CreateCollectionOrder)
	idem.POST("/offline-collection-orders/:collectionOrderID/cancel", a.paymentStoreHandler.CancelCollectionOrder)
	idem.POST("/tickets/verify", a.activityStoreHandler.VerifyTicket)
	idem.POST("/point-savings/:requestID/review", a.activityStoreHandler.ReviewPointSaving)
	idem.POST("/refunds", a.paymentStoreHandler.CreateRefund)
	idem.POST("/members/:memberID/wallet-adjustments", a.adminHandler.StoreCreateWalletAdjustment)

	a.registerStoreConsole(p, idem)
}

// registerStoreConsole wires the store console CRUD onto the store audience.
// Reads mount on the scoped read group p; every write mounts on idem, so it is
// pinned to the token's store scope and guarded by an Idempotency-Key. Catalog
// list reads for items live on adminHandler (already registered above), so only
// the detail read is added here. Catalog variants use the /catalog/variants/
// :itemID prefix for the same param-name reason as the admin console.
func (a *App) registerStoreConsole(p, idem *gin.RouterGroup) {
	// Food-order fulfillment (own store only).
	p.GET("/food-orders", a.orderStoreConsoleHandler.List)
	p.GET("/food-orders/:orderID", a.orderStoreConsoleHandler.Get)
	idem.POST("/food-orders/:orderID/:action", a.orderStoreConsoleHandler.Action)
	idem.POST("/reservations/:reservationID/cancel", a.reservationHandler.StoreCancel)

	// Own-store profile update.
	idem.PATCH("/profile", a.storeConsoleHandler.UpdateOwnProfile)
	idem.PATCH("/profile/status", a.storeConsoleHandler.UpdateOwnStatus)

	// Own-store settings.
	p.GET("/settings", a.storeConsoleHandler.GetOwnSettings)
	idem.PUT("/settings", a.storeConsoleHandler.UpdateOwnSettings)
	p.GET("/low-spend-reward-rule", a.storeLowSpendRuleHandler.StoreGet)
	idem.PUT("/low-spend-reward-rule", a.storeLowSpendRuleHandler.StoreUpdate)

	// Tables and seats (own store only).
	p.GET("/tables", a.reservationConsoleHandler.StoreListTables)
	p.GET("/tables/:tableID", a.reservationConsoleHandler.StoreGetTable)
	idem.POST("/tables", a.reservationConsoleHandler.StoreCreateTable)
	idem.PATCH("/tables/:tableID", a.reservationConsoleHandler.StoreUpdateTable)
	idem.DELETE("/tables/:tableID", a.reservationConsoleHandler.StoreDeleteTable)
	p.GET("/seats", a.reservationConsoleHandler.StoreListSeats)
	p.GET("/seats/:seatID", a.reservationConsoleHandler.StoreGetSeat)
	idem.POST("/seats", a.reservationConsoleHandler.StoreCreateSeat)
	idem.PATCH("/seats/:seatID", a.reservationConsoleHandler.StoreUpdateSeat)
	idem.DELETE("/seats/:seatID", a.reservationConsoleHandler.StoreDeleteSeat)

	// Catalog categories.
	p.GET("/catalog/categories", a.catalogConsoleHandler.StoreCategories)
	p.GET("/catalog/categories/:id", a.catalogConsoleHandler.StoreGetCategory)
	idem.POST("/catalog/categories", a.catalogConsoleHandler.StoreCreateCategory)
	idem.PUT("/catalog/categories/:id", a.catalogConsoleHandler.StoreUpdateCategory)
	idem.DELETE("/catalog/categories/:id", a.catalogConsoleHandler.StoreDeleteCategory)

	// Catalog items (list read already registered above).
	p.GET("/catalog/items/:id", a.catalogConsoleHandler.StoreGetItem)
	idem.POST("/catalog/items", a.catalogConsoleHandler.StoreCreateItem)
	idem.PUT("/catalog/items/:id", a.catalogConsoleHandler.StoreUpdateItem)
	idem.DELETE("/catalog/items/:id", a.catalogConsoleHandler.StoreDeleteItem)

	// Catalog variants (scoped by :itemID; see admin console prefix note).
	p.GET("/catalog/variants/:itemID", a.catalogConsoleHandler.StoreVariants)
	p.GET("/catalog/variants/:itemID/:id", a.catalogConsoleHandler.StoreGetVariant)
	idem.POST("/catalog/variants/:itemID", a.catalogConsoleHandler.StoreCreateVariant)
	idem.PUT("/catalog/variants/:itemID/:id", a.catalogConsoleHandler.StoreUpdateVariant)
	idem.DELETE("/catalog/variants/:itemID/:id", a.catalogConsoleHandler.StoreDeleteVariant)

	// Activities (list read already registered as adminHandler.StoreActivities).
	p.GET("/activities/:activityID", a.activityConsoleHandler.StoreActivityDetail)
	idem.POST("/activities", a.activityConsoleHandler.StoreCreateActivity)
	idem.PUT("/activities/:activityID", a.activityConsoleHandler.StoreUpdateActivity)
	idem.DELETE("/activities/:activityID", a.activityConsoleHandler.StoreDeleteActivity)

	// Activity sessions and ticket types.
	p.GET("/activities/:activityID/sessions", a.activityConsoleHandler.StoreSessions)
	p.GET("/activities/:activityID/sessions/:sessionID", a.activityConsoleHandler.StoreSessionDetail)
	idem.POST("/activities/:activityID/sessions", a.activityConsoleHandler.StoreCreateSession)
	idem.PUT("/activities/:activityID/sessions/:sessionID", a.activityConsoleHandler.StoreUpdateSession)
	idem.DELETE("/activities/:activityID/sessions/:sessionID", a.activityConsoleHandler.StoreDeleteSession)
	p.GET("/activities/:activityID/ticket-types", a.activityConsoleHandler.StoreTicketTypes)
	p.GET("/activities/:activityID/ticket-types/:ticketTypeID", a.activityConsoleHandler.StoreTicketTypeDetail)
	idem.POST("/activities/:activityID/ticket-types", a.activityConsoleHandler.StoreCreateTicketType)
	idem.PUT("/activities/:activityID/ticket-types/:ticketTypeID", a.activityConsoleHandler.StoreUpdateTicketType)
	idem.DELETE("/activities/:activityID/ticket-types/:ticketTypeID", a.activityConsoleHandler.StoreDeleteTicketType)

	// Coupon templates (list read already registered as adminHandler.StoreCouponTemplates).
	p.GET("/coupon-templates/:id", a.couponConsoleHandler.StoreGet)
	p.GET("/coupon-templates/:id/applicable-items", a.couponConsoleHandler.StoreApplicableItems)
	idem.POST("/coupon-templates", a.couponConsoleHandler.StoreCreate)
	idem.PUT("/coupon-templates/:id", a.couponConsoleHandler.StoreUpdate)
	idem.DELETE("/coupon-templates/:id", a.couponConsoleHandler.StoreDelete)
	idem.POST("/coupon-templates/:id/publish", a.couponConsoleHandler.StorePublish)
	idem.POST("/coupon-templates/:id/disable", a.couponConsoleHandler.StoreDisable)

	// High-risk coupon entitlement actions.
	idem.POST("/coupon-grants", a.couponConsoleHandler.StoreGrant)
	idem.POST("/coupon-voids", a.couponConsoleHandler.StoreVoid)
	idem.POST("/coupon-verifications", a.couponConsoleHandler.StoreVerify)

	// Printer devices (own store only). Writes are pinned to the token scope and
	// guarded by an Idempotency-Key like every other store mutation.
	p.GET("/printer-devices", a.printerConsoleHandler.StoreList)
	idem.POST("/printer-devices", a.printerConsoleHandler.StoreCreate)
	idem.PATCH("/printer-devices/:id", a.printerConsoleHandler.StoreUpdate)
	idem.DELETE("/printer-devices/:id", a.printerConsoleHandler.StoreDelete)

	// Cashier accounts (admin_accounts, role=cashier; own store only). List read
	// already registered as adminHandler.StoreCashiers.
	idem.POST("/cashiers", a.adminHandler.StoreCreateCashier)
	idem.PATCH("/cashiers/:cashierID", a.adminHandler.StoreUpdateCashier)
	idem.POST("/cashiers/:cashierID/disable", a.adminHandler.StoreDisableCashier)
	idem.POST("/cashiers/:cashierID/password-reset", a.adminHandler.StoreResetCashierPassword)

	// Staff accounts (staff_accounts; own store only). List read already
	// registered as adminHandler.StoreStaffAccounts.
	idem.POST("/staff-accounts", a.adminHandler.StoreCreateStaffAccount)
	idem.PATCH("/staff-accounts/:staffID", a.adminHandler.StoreUpdateStaffAccount)
	idem.POST("/staff-accounts/:staffID/disable", a.adminHandler.StoreDisableStaffAccount)
	idem.DELETE("/staff-accounts/:staffID/binding", a.adminHandler.StoreDeleteStaffAccount)
}
