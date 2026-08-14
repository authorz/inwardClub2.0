// 预约 — 固定桌台列表 · 预约用户头像 · 座位确认弹层
const api = require('../../services/api');
const validation = require('../../utils/validation');
const auth = require('../../utils/auth');
const http = require('../../utils/request');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const { saveProfile } = require('../../utils/member-profile');

const RESERVATION_DAY_START_HOUR = 4;

function isCurrentReservationDay(value) {
  const timestamp = Date.parse(value || '');
  if (!Number.isFinite(timestamp)) return false;
  const now = new Date();
  const start = new Date(
    now.getFullYear(),
    now.getMonth(),
    now.getDate(),
    RESERVATION_DAY_START_HOUR,
    0,
    0,
    0
  );
  if (now.getTime() < start.getTime()) start.setDate(start.getDate() - 1);
  const end = new Date(start.getTime());
  end.setDate(end.getDate() + 1);
  return timestamp >= start.getTime() && timestamp < end.getTime();
}

Page({
  data: {
    loading: true,
    store: null,
    tables: [],
    showConfirm: false,
    submitting: false,
    cancellingReservationId: '',
    hasDailyReservation: false,
    selected: null, // { tableId, tableName }
  },

  onLoad() {
    this.load();
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 1 });
    }
    const s = storeCtx.get();
    if (s && this.data.store && s.id !== this.data.store.id) {
      this.setData({ store: s });
      this.loadTables(s.id);
    } else if (s && this.data.store) {
      this.loadTables(s.id);
    }
  },

  load() {
    // Resolve the current store (persisted pick, else nearest) before loading its
    // tables — reservations are store-scoped.
    storeCtx.ensureStore().then((store) => {
      this.setData({ store });
      if (store) {
        this.loadTables(store.id);
      }
      else this.setData({ loading: false });
    });
  },

  loadTables(storeId) {
    this.setData({ loading: true });
    // Tables carry no seats; the seat endpoint includes current occupancy and
    // the booked member's display profile. Group the real seats by table.
    // When logged in, load the member's active reservation as well so their
    // occupied seat can be identified from their own point of view.
    const ownReservations = auth.hasReservationIdentity()
      ? api.getReservations({ pageSize: 50 }).catch(() => ({ data: [] }))
      : Promise.resolve({ data: [] });
    Promise.all([api.getTables(storeId), api.getSeats(storeId), ownReservations])
      .then(([tRes, sRes, rRes]) => {
        const activeReservations = (rRes.data || []).filter((reservation) =>
          (reservation.status === 'booked' || reservation.status === 'arrived') &&
          isCurrentReservationDay(reservation.createdAt || reservation.reservedAt)
        );
        const ownReservationsBySeat = new Map();
        const ownReservationsByTable = new Map();
        const currentStoreReservations = activeReservations
          .filter((reservation) =>
            String(reservation.storeId) === String(storeId)
          );
        currentStoreReservations.forEach((reservation) => {
          if (reservation.seatId != null) {
            ownReservationsBySeat.set(String(reservation.seatId), reservation);
          }
          if (reservation.tableId != null) {
            ownReservationsByTable.set(String(reservation.tableId), reservation);
          }
        });
        const byTable = {};
        (sRes.data || []).forEach((seat) => {
          const key = seat.tableId != null ? seat.tableId : '';
          (byTable[key] = byTable[key] || []).push(seat);
        });
        const tables = (tRes.data || []).map((t) =>
          this.decorate(
            t,
            byTable[t.id],
            ownReservationsBySeat,
            ownReservationsByTable,
            activeReservations.length > 0
          )
        );
        this.setData({
          tables,
          hasDailyReservation: activeReservations.length > 0,
          loading: false,
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  /** Attach display fields for a real SeatView, including the current occupant. */
  decorateSeat(s, ownReservationsBySeat) {
    const ownReservation = ownReservationsBySeat.get(String(s.id));
    const state = ownReservation || s.status !== 'available' ? 'reserved' : 'free';
    const isGuest = Boolean(s.isGuest);
    const nickname = isGuest ? 'inward会员' : s.nickname || '';
    const gender = s.gender === 'male' || s.gender === 'female' ? s.gender : '';
    const isMine = Boolean(ownReservation);
    return {
      no: s.name,
      seatId: s.id,
      state,
      avatarUrl: isGuest ? 'https://assets.inwardclub.com/public/images/inward-logo-optimized.gif?imageMogr2/format/png' : s.avatarUrl || '',
      nickname,
      accessibilityLabel: state === 'reserved' ? nickname || '已预约用户' : '空位',
      isMine,
      reservationId: ownReservation ? ownReservation.id : '',
      reservationStatus: ownReservation ? ownReservation.status : '',
      isGuest,
      initial: (nickname || '会').trim().charAt(0).toUpperCase(),
      gender,
      genderIcon: gender ? '/assets/icons/gender-' + gender + '.svg' : '',
    };
  },

  /** Attach fixed-row display fields to a table. */
  decorate(t, realSeats, ownReservationsBySeat, ownReservationsByTable, hasDailyReservation) {
    const seats = (realSeats || []).map((s) =>
      this.decorateSeat(s, ownReservationsBySeat)
    );
    const cap = t.capacity || seats.length;
    const free = seats.filter((s) => s.state === 'free').length;
    const mine = seats.find((s) => s.isMine && s.reservationId) || null;
    const ownReservation = mine
      ? { id: mine.reservationId, status: mine.reservationStatus, seatNo: mine.no }
      : ownReservationsByTable.get(String(t.id));
    return {
      id: t.id,
      name: t.name,
      reservedText: (t.reserved != null ? t.reserved : cap - free) + '/' + cap,
      canReserve: free > 0,
      isMineTable: Boolean(ownReservation),
      mineReservationId: ownReservation ? ownReservation.id : '',
      mineSeatNo: ownReservation ? ownReservation.seatNo || '' : '',
      actionText: ownReservation ? '取消预约' : '预约座位',
      actionDisabled: ownReservation
        ? ownReservation.status !== 'booked'
        : hasDailyReservation || free === 0,
      seats,
    };
  },

  onTableAction(e) {
    const id = e.currentTarget.dataset.id;
    const tables = this.data.tables;
    const table = tables.find((item) => String(item.id) === String(id));
    if (!table) return;
    if (table.isMineTable) {
      this.cancelOwnReservation(table);
      return;
    }
    if (this.data.hasDailyReservation) {
      ui.toast('一天只能预约一个座位');
      return;
    }
    if (!table.canReserve) {
      ui.toast('该桌暂时没有空位');
      return;
    }
    this.clearSelection();
    this.setData({
      selected: {
        tableId: table.id,
        tableName: table.name,
      },
      showConfirm: true,
    });
  },

  cancelOwnReservation(table) {
    const reservationId = table.mineReservationId;
    if (!reservationId || this.data.cancellingReservationId) return;
    ui.confirm({
      title: '取消预约',
      content: `确认取消 ${table.name} · ${table.mineSeatNo}号座位？`,
      confirmText: '取消预约',
    }).then((ok) => {
      if (!ok) return;
      this.setData({ cancellingReservationId: reservationId });
      ui.showLoading('取消中');
      api.cancelReservation(reservationId, http.uuid())
        .then(() => {
          ui.hideLoading();
          this.setData({ cancellingReservationId: '' });
          this.loadTables(this.data.store.id);
          ui.success('已取消预约');
        })
        .catch((err) => {
          ui.hideLoading();
          this.setData({ cancellingReservationId: '' });
          ui.error((err && err.message) || '取消失败');
        });
    });
  },

  clearSelection() {
    this.setData({ selected: null, showConfirm: false });
  },

  onCloseConfirm() {
    this.clearSelection();
  },

  onConfirm() {
    const sel = this.data.selected;
    const store = this.data.store;
    if (!sel || this.data.submitting) return;
	try {
	  validation.integer(store && store.id, { label: '门店', min: 1 });
	  validation.integer(sel.tableId, { label: '桌子', min: 1 });
	} catch (err) {
	  ui.toast(err.message);
	  return;
	}
    this.setData({ submitting: true });
    ui.showLoading('提交中');
    this.ensureReservationIdentity()
      .then(() => api.createReservation({
        storeId: store && store.id,
        tableId: sel.tableId,
        tableName: sel.tableName,
      }))
      .then(() => {
        ui.hideLoading();
        this.setData({ showConfirm: false, selected: null, submitting: false });
        this.loadTables(store.id);
        ui.success('预约成功');
      })
      .catch((err) => {
        ui.hideLoading();
        this.clearSelection();
        this.setData({ submitting: false });
        ui.error((err && err.message) || '预约失败');
      });
  },

  ensureReservationIdentity() {
    if (auth.hasReservationIdentity()) return Promise.resolve();
    return this.requestSilentReservationIdentity(0);
  },

  requestSilentReservationIdentity(attempt) {
    return new Promise((resolve, reject) => {
      wx.login({
        success: (loginRes) => {
          if (!loginRes || !loginRes.code) {
            reject(new Error('微信身份获取失败，请重试'));
            return;
          }
          resolve(loginRes.code);
        },
        fail: () => reject(new Error('微信身份获取失败，请重试')),
      });
    })
      .then((code) => api.preRegister({ code }))
      .then((res) => {
        const result = (res && res.data) || {};
        const token = result.token || {};
        if (!token.accessToken || !token.refreshToken) {
          throw new Error('微信身份获取失败，请重试');
        }
        const subjectType = result.subjectType || 'pre_member';
        auth.save({
          accessToken: token.accessToken,
          refreshToken: token.refreshToken,
          subjectType,
          storeId: result.storeId,
        });
        if (subjectType !== 'pre_member') {
          const profile = result.profile || {};
          saveProfile({
            avatarUrl: profile.avatarUrl || '',
            nickname: profile.nickname || '',
            gender: profile.gender || '',
          });
        }
      })
      .catch((err) => {
        if (attempt < 1 && err && (err.httpStatus === 401 || err.code === 'UNAUTHENTICATED')) {
          return this.requestSilentReservationIdentity(attempt + 1);
        }
        throw err;
      });
  },

  onWaitlist() {
    this.clearSelection();
    wx.navigateTo({ url: '/pages/waitlist/waitlist' });
  },

  goMyReservations() {
    wx.navigateTo({ url: '/pages/reservation-list/reservation-list' });
  },

  switchStore() {
    wx.navigateTo({ url: '/pages/store-select/store-select' });
  },
});
