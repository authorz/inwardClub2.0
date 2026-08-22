package payment

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/modules/printer"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// spineRepo is an in-memory SettlementRepository that mirrors the SQL
// settlement contract branch-for-branch: it advances the payment order, its
// business order and (offline) collection order to paid exactly once, records a
// payment_transaction guarded by the (provider, out_trade_no) unique key, and
// treats an already-paid order as an idempotent success. Tests assert on this
// spine so the settlement semantics are covered without a live MySQL.
type spineRepo struct {
	paymentOrders map[string]*poRow // keyed by payment_order_no
	// collections maps acquirer_order_no -> payment_order_no for the offline
	// fallback lookup path.
	collections map[string]string
	// txns is the durable duplicate guard: provider|out_trade_no it has seen.
	txns map[string]bool
	// insertCount records how many transaction rows were actually written, so a
	// test can assert no double-insert on repeated notifies.
	insertCount int
	// postProcessCount records how many post-payment processing outbox events were
	// written; a member-bound offline settlement writes exactly one.
	postProcessCount int
	// deviceSNByStore mirrors an active printer_devices row per store: a settled,
	// store-bound order writes a print:receipt event only when its store has one.
	deviceSNByStore map[int64]string
	// receiptJobs records the printer.Job each print:receipt event would carry, so
	// a test asserts both how many receipts a flow produces and their rendered body.
	receiptJobs []printer.Job
}

type poRow struct {
	id           int64
	businessID   int64
	amount       int64
	status       string // pending | paid
	businessPaid bool
	// collectionStatus is non-empty for offline orders and tracks the collection
	// order state (pending | paid | cancelled | expired) alongside the payment order.
	collectionStatus string
	// memberID is the collection's bound member (0 = walk-in). A member-bound
	// settlement hands off to post-payment processing; a walk-in does not.
	memberID int64
	// storeID/orderType/businessNo are the receipt facts. A store-less order
	// (storeID == 0, e.g. recharge) prints nothing.
	storeID    int64
	orderType  string
	businessNo string
}

func newSpineRepo() *spineRepo {
	return &spineRepo{
		paymentOrders:   map[string]*poRow{},
		collections:     map[string]string{},
		txns:            map[string]bool{},
		deviceSNByStore: map[int64]string{},
	}
}

// maybeReceipt mirrors printer.WriteReceipt's producer rule: a store-bound order
// whose store has an active printer device appends one print:receipt job. It is
// called only on a fresh settlement, so replays never double-print.
func (r *spineRepo) maybeReceipt(po *poRow) {
	if po.storeID == 0 {
		return
	}
	sn, ok := r.deviceSNByStore[po.storeID]
	if !ok {
		return
	}
	r.receiptJobs = append(r.receiptJobs, printer.BuildReceiptJob(sn, true, printer.Receipt{
		StoreID:         po.storeID,
		PaymentOrderID:  po.id,
		BusinessOrderNo: po.businessNo,
		OrderType:       po.orderType,
		AmountCent:      po.amount,
		PaidAt:          time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC),
	}))
}

func (r *spineRepo) addWeChatOrder(no string, businessID, amount int64) {
	r.paymentOrders[no] = &poRow{businessID: businessID, amount: amount, status: paymentPending}
}

func (r *spineRepo) addOfflineOrder(no, acquirerNo string, businessID, amount int64) {
	r.paymentOrders[no] = &poRow{businessID: businessID, amount: amount, status: paymentPending, collectionStatus: CollectionPending}
	r.collections[acquirerNo] = no
}

// settle mirrors the SQL settlement branch-for-branch and reports whether a
// fresh settlement occurred (a transaction was inserted), so a caller can attach
// post-settlement side effects only on the settling call and not on idempotent
// replays.
func (r *spineRepo) settle(po *poRow, provider, outTradeNo string) (bool, error) {
	offline := po.collectionStatus != ""
	if po.status == paymentPaid && (!offline || po.collectionStatus == CollectionPaid) {
		return false, nil // idempotent: duplicate notify for an already-settled order
	}
	if po.status != paymentPending {
		return false, apperr.Conflict("payment order is not payable")
	}
	if offline && po.collectionStatus != CollectionPending {
		// A cancelled/expired collection must never be advanced to paid.
		return false, apperr.Conflict("collection order is not payable")
	}
	key := provider + "|" + outTradeNo
	if r.txns[key] {
		// Concurrent/duplicate notify already recorded this transaction.
		return false, apperr.Conflict("payment already settled")
	}
	r.txns[key] = true
	r.insertCount++
	po.status = paymentPaid
	po.businessPaid = true
	if offline {
		po.collectionStatus = CollectionPaid
	}
	return true, nil
}

func (r *spineRepo) SettleWeChat(_ context.Context, n WeChatNotification, _ time.Time) error {
	po, ok := r.paymentOrders[n.OutTradeNo]
	if !ok {
		return apperr.NotFound("payment order not found")
	}
	settled, err := r.settle(po, wechatProvider, n.OutTradeNo)
	if err != nil {
		return err
	}
	if settled {
		if po.collectionStatus != "" && po.memberID != 0 {
			r.postProcessCount++
		}
		r.maybeReceipt(po)
	}
	return nil
}

func TestSettleWeChatNativeCollection(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addOfflineOrder("PO-NATIVE", "", 20, 2000)
	po := repo.paymentOrders["PO-NATIVE"]
	po.id, po.storeID, po.orderType, po.businessNo, po.memberID = 91, 7, collectionType, "BO-NATIVE", 42

	n := WeChatNotification{OutTradeNo: "PO-NATIVE", TransactionID: "wx-native", AmountCent: 2000, Success: true}
	for i := 0; i < 2; i++ {
		if err := svc.SettleWeChat(context.Background(), n); err != nil {
			t.Fatalf("settle #%d: %v", i, err)
		}
	}
	if po.status != paymentPaid || !po.businessPaid || po.collectionStatus != CollectionPaid {
		t.Fatalf("expected native collection spine paid: %+v", po)
	}
	if repo.postProcessCount != 1 {
		t.Fatalf("expected one member post-process event, got %d", repo.postProcessCount)
	}
}

func (r *spineRepo) SettleOffline(_ context.Context, n OfflineNotification, _ time.Time) error {
	no := n.OutTradeNo
	if no == "" {
		no = r.collections[n.AcquirerOrderNo]
	}
	po, ok := r.paymentOrders[no]
	if !ok {
		return apperr.NotFound("collection order not found")
	}
	settled, err := r.settle(po, offlineProvider, poNoOrAcquirer(n))
	if err != nil {
		return err
	}
	// Only a fresh, member-bound settlement writes the post-payment outbox event.
	if settled && po.memberID != 0 {
		r.postProcessCount++
	}
	// The counter collection is store-bound, so a fresh settlement prints a receipt.
	if settled {
		r.maybeReceipt(po)
	}
	return nil
}

func newTestSettlementService() (*SettlementService, *spineRepo) {
	repo := newSpineRepo()
	svc := NewSettlementService(repo)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	return svc, repo
}

func TestSettleWeChatIdempotent(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addWeChatOrder("PO-1", 10, 1500)
	n := WeChatNotification{OutTradeNo: "PO-1", TransactionID: "wx-1", AmountCent: 1500, Success: true}
	for i := 0; i < 3; i++ {
		if err := svc.SettleWeChat(context.Background(), n); err != nil {
			t.Fatalf("settle #%d: %v", i, err)
		}
	}
	po := repo.paymentOrders["PO-1"]
	if repo.insertCount != 1 {
		t.Fatalf("expected exactly one transaction, got %d", repo.insertCount)
	}
	if po.status != paymentPaid || !po.businessPaid {
		t.Fatalf("expected payment order and business order paid: %+v", po)
	}
}

func TestSettleOfflineIdempotent(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addOfflineOrder("PO-1", "acq-1", 20, 2000)
	n := OfflineNotification{OutTradeNo: "PO-1", ExternalTradeNo: "ext-1", Channel: "wechat", AmountCent: 2000, Success: true}
	for i := 0; i < 3; i++ {
		if err := svc.SettleOffline(context.Background(), n); err != nil {
			t.Fatalf("settle #%d: %v", i, err)
		}
	}
	po := repo.paymentOrders["PO-1"]
	if repo.insertCount != 1 {
		t.Fatalf("expected exactly one transaction, got %d", repo.insertCount)
	}
	// The whole spine must be pulled through: payment, business and collection.
	if po.status != paymentPaid || !po.businessPaid || po.collectionStatus != CollectionPaid {
		t.Fatalf("expected payment/business/collection all paid: %+v", po)
	}
}

func TestSettleOfflineMemberBoundPostProcess(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addOfflineOrder("PO-M", "acq-m", 30, 3000)
	repo.paymentOrders["PO-M"].memberID = 42 // bound member
	n := OfflineNotification{OutTradeNo: "PO-M", ExternalTradeNo: "ext-m", Channel: "wechat", AmountCent: 3000, Success: true}
	// Repeated notifies must settle once and hand off to post-processing once.
	for i := 0; i < 3; i++ {
		if err := svc.SettleOffline(context.Background(), n); err != nil {
			t.Fatalf("settle #%d: %v", i, err)
		}
	}
	if repo.insertCount != 1 {
		t.Fatalf("expected exactly one transaction, got %d", repo.insertCount)
	}
	if repo.postProcessCount != 1 {
		t.Fatalf("member-bound settlement must write exactly one post-process event, got %d", repo.postProcessCount)
	}
}

func TestSettleOfflineWalkInNoPostProcess(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addOfflineOrder("PO-W", "acq-w", 31, 3100) // walk-in: memberID stays 0
	n := OfflineNotification{OutTradeNo: "PO-W", ExternalTradeNo: "ext-w", Channel: "alipay", AmountCent: 3100, Success: true}
	if err := svc.SettleOffline(context.Background(), n); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if repo.postProcessCount != 0 {
		t.Fatalf("walk-in settlement must not write a post-process event, got %d", repo.postProcessCount)
	}
}

func TestSettleOfflineAcquirerFallback(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addOfflineOrder("PO-2", "acq-1", 20, 2000)
	n := OfflineNotification{AcquirerOrderNo: "acq-1", Channel: "alipay", AmountCent: 2000, Success: true}
	if err := svc.SettleOffline(context.Background(), n); err != nil {
		t.Fatalf("settle by acquirer no: %v", err)
	}
	po := repo.paymentOrders["PO-2"]
	if po.status != paymentPaid || po.collectionStatus != CollectionPaid {
		t.Fatalf("expected settlement via acquirer fallback: %+v", po)
	}
}

// TestSettleOfflineCancelledCollectionConflict is the core Off-1 regression: a
// forged/late offline notify for a cancelled collection order must be rejected
// with CONFLICT and must never advance the order to paid.
func TestSettleOfflineCancelledCollectionConflict(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addOfflineOrder("PO-6", "acq-6", 20, 2000)
	repo.paymentOrders["PO-6"].collectionStatus = CollectionCancelled
	n := OfflineNotification{OutTradeNo: "PO-6", ExternalTradeNo: "ext-6", Channel: "wechat", AmountCent: 2000, Success: true}
	if err := svc.SettleOffline(context.Background(), n); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected CONFLICT for cancelled collection, got %v", err)
	}
	po := repo.paymentOrders["PO-6"]
	if po.status == paymentPaid || po.collectionStatus != CollectionCancelled {
		t.Fatalf("cancelled collection must never be paid: %+v", po)
	}
	if repo.insertCount != 0 {
		t.Fatalf("cancelled collection must not record a transaction, got %d", repo.insertCount)
	}
}

// TestSettleOfflineExpiredCollectionConflict mirrors the cancelled path for an
// expired collection order.
func TestSettleOfflineExpiredCollectionConflict(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addOfflineOrder("PO-7", "acq-7", 20, 2000)
	repo.paymentOrders["PO-7"].collectionStatus = CollectionExpired
	n := OfflineNotification{OutTradeNo: "PO-7", ExternalTradeNo: "ext-7", Channel: "wechat", AmountCent: 2000, Success: true}
	if err := svc.SettleOffline(context.Background(), n); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected CONFLICT for expired collection, got %v", err)
	}
	if po := repo.paymentOrders["PO-7"]; po.status == paymentPaid || po.collectionStatus != CollectionExpired {
		t.Fatalf("expired collection must never be paid: %+v", po)
	}
}

// TestSettleWeChatDuplicateTxnConflict asserts that when the (provider,
// out_trade_no) unique key is already taken (a concurrent notify won the race),
// the settlement surfaces CONFLICT and does not write a second transaction.
func TestSettleWeChatDuplicateTxnConflict(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addWeChatOrder("PO-3", 10, 1500)
	repo.txns[wechatProvider+"|PO-3"] = true // pre-existing transaction row
	n := WeChatNotification{OutTradeNo: "PO-3", TransactionID: "wx-3", AmountCent: 1500, Success: true}
	if err := svc.SettleWeChat(context.Background(), n); apperr.From(err).Code != apperr.CodeConflict {
		t.Fatalf("expected CONFLICT on duplicate transaction, got %v", err)
	}
	if repo.insertCount != 0 {
		t.Fatalf("duplicate key must not insert a second transaction, got %d", repo.insertCount)
	}
}

// TestSettleWeChatAlreadyPaid asserts that a duplicate notify for a payment
// order already marked paid returns success without recording anything.
func TestSettleWeChatAlreadyPaid(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addWeChatOrder("PO-4", 10, 1500)
	repo.paymentOrders["PO-4"].status = paymentPaid // already settled
	n := WeChatNotification{OutTradeNo: "PO-4", TransactionID: "wx-4", AmountCent: 1500, Success: true}
	if err := svc.SettleWeChat(context.Background(), n); err != nil {
		t.Fatalf("already-paid notify must succeed idempotently: %v", err)
	}
	if repo.insertCount != 0 {
		t.Fatalf("already-paid order must not record a new transaction, got %d", repo.insertCount)
	}
}

// TestSettleOfflineAlreadyPaid mirrors the already-paid path for offline notifies.
func TestSettleOfflineAlreadyPaid(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addOfflineOrder("PO-5", "acq-5", 20, 2000)
	repo.paymentOrders["PO-5"].status = paymentPaid
	repo.paymentOrders["PO-5"].collectionStatus = CollectionPaid
	n := OfflineNotification{OutTradeNo: "PO-5", ExternalTradeNo: "ext-5", Channel: "wechat", AmountCent: 2000, Success: true}
	if err := svc.SettleOffline(context.Background(), n); err != nil {
		t.Fatalf("already-paid offline notify must succeed idempotently: %v", err)
	}
	if repo.insertCount != 0 {
		t.Fatalf("already-paid order must not record a new transaction, got %d", repo.insertCount)
	}
}

func TestSettleWeChatNonSuccessNoop(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addWeChatOrder("PO-1", 10, 1500)
	n := WeChatNotification{OutTradeNo: "PO-1", Success: false}
	if err := svc.SettleWeChat(context.Background(), n); err != nil {
		t.Fatalf("non-success settle: %v", err)
	}
	if repo.insertCount != 0 {
		t.Fatalf("failed notify must not settle")
	}
}

func TestSettleWeChatMissingOutTradeNo(t *testing.T) {
	svc, _ := newTestSettlementService()
	n := WeChatNotification{Success: true}
	if err := svc.SettleWeChat(context.Background(), n); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestSettleOfflineMissingKeys(t *testing.T) {
	svc, _ := newTestSettlementService()
	n := OfflineNotification{Channel: "wechat", Success: true}
	if err := svc.SettleOffline(context.Background(), n); apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

// TestSettleWeChatStoreBoundReceipt: a store-bound WeChat order whose store has
// an active printer prints exactly one receipt, even across duplicate notifies,
// and the receipt carries the rendered order facts for the store's device.
func TestSettleWeChatStoreBoundReceipt(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addWeChatOrder("PO-R", 10, 1500)
	po := repo.paymentOrders["PO-R"]
	po.id, po.storeID, po.orderType, po.businessNo = 77, 5, "food", "BO-R"
	repo.deviceSNByStore[5] = "SN-5" // store 5 has an active printer
	n := WeChatNotification{OutTradeNo: "PO-R", TransactionID: "wx-r", AmountCent: 1500, Success: true}
	for i := 0; i < 3; i++ {
		if err := svc.SettleWeChat(context.Background(), n); err != nil {
			t.Fatalf("settle #%d: %v", i, err)
		}
	}
	if len(repo.receiptJobs) != 1 {
		t.Fatalf("expected exactly one receipt, got %d", len(repo.receiptJobs))
	}
	job := repo.receiptJobs[0]
	if job.DeviceSN != "SN-5" || job.Template != printer.ReceiptTemplate {
		t.Fatalf("unexpected receipt job: %+v", job)
	}
	for _, want := range []string{"餐饮订单", "BO-R", "¥15.00"} {
		if !strings.Contains(job.Content, want) {
			t.Fatalf("receipt content missing %q:\n%s", want, job.Content)
		}
	}
}

// TestSettleWeChatRechargeNoReceipt: a store-less order (recharge) prints nothing.
func TestSettleWeChatRechargeNoReceipt(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addWeChatOrder("PO-N", 11, 5000)
	po := repo.paymentOrders["PO-N"]
	po.id, po.orderType, po.businessNo = 78, "recharge", "BO-N" // storeID stays 0
	n := WeChatNotification{OutTradeNo: "PO-N", TransactionID: "wx-n", AmountCent: 5000, Success: true}
	if err := svc.SettleWeChat(context.Background(), n); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if len(repo.receiptJobs) != 0 {
		t.Fatalf("store-less order must not print, got %d", len(repo.receiptJobs))
	}
}

// TestSettleWeChatStoreWithoutPrinterNoReceipt: a store-bound order whose store
// has no active printer device prints nothing (settlement still succeeds).
func TestSettleWeChatStoreWithoutPrinterNoReceipt(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addWeChatOrder("PO-D", 12, 1500)
	po := repo.paymentOrders["PO-D"]
	po.id, po.storeID, po.orderType, po.businessNo = 79, 6, "food", "BO-D" // store 6 has no device
	n := WeChatNotification{OutTradeNo: "PO-D", TransactionID: "wx-d", AmountCent: 1500, Success: true}
	if err := svc.SettleWeChat(context.Background(), n); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if len(repo.receiptJobs) != 0 {
		t.Fatalf("store without a printer must not print, got %d", len(repo.receiptJobs))
	}
}

// TestSettleOfflineReceipt: an offline counter collection prints one receipt on
// its store's printer, and only once across duplicate notifies.
func TestSettleOfflineReceipt(t *testing.T) {
	svc, repo := newTestSettlementService()
	repo.addOfflineOrder("PO-OF", "acq-of", 20, 2000)
	po := repo.paymentOrders["PO-OF"]
	po.id, po.storeID, po.orderType, po.businessNo = 80, 9, "offline_collection", "BO-OF"
	repo.deviceSNByStore[9] = "SN-9"
	n := OfflineNotification{OutTradeNo: "PO-OF", ExternalTradeNo: "ext-of", Channel: "wechat", AmountCent: 2000, Success: true}
	for i := 0; i < 2; i++ {
		if err := svc.SettleOffline(context.Background(), n); err != nil {
			t.Fatalf("settle #%d: %v", i, err)
		}
	}
	if len(repo.receiptJobs) != 1 {
		t.Fatalf("expected exactly one offline receipt, got %d", len(repo.receiptJobs))
	}
	if job := repo.receiptJobs[0]; job.DeviceSN != "SN-9" || !strings.Contains(job.Content, "门店收款") {
		t.Fatalf("unexpected offline receipt: %+v", job)
	}
}
