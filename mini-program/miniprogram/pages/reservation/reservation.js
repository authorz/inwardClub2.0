// 预约 — 多桌 · 每张德州扑克桌 9 座 · 座位确认弹层
// Reference: design/mini-program/final/reservation/05-reservation-final-multi-table.png
//            design/mini-program/final/reservation/05-reservation-final-confirm-sheet.png
const api = require('../../services/api');
const validation = require('../../utils/validation');
const auth = require('../../utils/auth');
const http = require('../../utils/request');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const { saveProfile } = require('../../utils/member-profile');
const { SEAT_STATE_LABEL } = require('../../constants/index');

// Fixed 9-seat ring positions around the full table background, clockwise from top-left.
const SEAT_POS = [
  'left:28.4%;top:11.5%;',
  'left:50%;top:11.5%;',
  'left:71.6%;top:11.5%;',
  'left:91.5%;top:49%;',
  'left:73.5%;top:86%;',
  'left:58.6%;top:86%;',
  'left:41.4%;top:86%;',
  'left:26%;top:86%;',
  'left:8.5%;top:49%;',
];

Page({
  data: {
    loading: true,
    store: null,
    tournamentEvent: null,
    tables: [],
    expandedId: '',
    legend: [
      { state: 'free', label: '空闲' },
      { state: 'reserved', label: '已预约' },
      { state: 'selected', label: '已选择' },
    ],
    showConfirm: false,
    submitting: false,
    cancellingReservationId: '',
    selected: null, // { tableId, tableName, seatNo }
    navStatusBar: 20,
    navContentHeight: 44,
    navRightGap: 96,
  },

  onLoad() {
    this.measureNav();
    this.load();
  },

  measureNav() {
    try {
      const win = wx.getWindowInfo ? wx.getWindowInfo() : wx.getSystemInfoSync();
      const cap = wx.getMenuButtonBoundingClientRect();
      const statusBar = win.statusBarHeight || 20;
      const gap = Math.max(cap.top - statusBar, 4);
      this.setData({
        navStatusBar: statusBar,
        navContentHeight: cap.height + gap * 2,
        navRightGap: Math.max(win.windowWidth - cap.left + 8, 96),
      });
    } catch {
      /* keep defaults */
    }
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) {
      this.getTabBar().setData({ selected: 1 });
    }
    const s = storeCtx.get();
    if (s && this.data.store && s.id !== this.data.store.id) {
      this.setData({ store: s });
      this.loadTables(s.id);
      this.loadTournamentEvent(s.id);
    } else if (s && this.data.store) {
      this.loadTables(s.id);
      this.loadTournamentEvent(s.id);
    }
  },

  load() {
    // Resolve the current store (persisted pick, else nearest) before loading its
    // tables — reservations are store-scoped.
    storeCtx.ensureStore().then((store) => {
      this.setData({ store });
      if (store) {
        this.loadTables(store.id);
        this.loadTournamentEvent(store.id);
      }
      else this.setData({ loading: false });
    });
  },

  loadTournamentEvent(storeId) {
    api.getTournamentEvents(storeId)
      .then((res) => {
        const events = res.data || [];
        this.setData({ tournamentEvent: events[0] || null });
      })
      .catch(() => this.setData({ tournamentEvent: null }));
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
        const ownReservationsBySeat = new Map();
        (rRes.data || [])
          .filter((reservation) =>
            reservation.seatId != null && String(reservation.storeId) === String(storeId)
          )
          .forEach((reservation) => {
            ownReservationsBySeat.set(String(reservation.seatId), reservation);
          });
        const byTable = {};
        (sRes.data || []).forEach((seat) => {
          const key = seat.tableId != null ? seat.tableId : '';
          (byTable[key] = byTable[key] || []).push(seat);
        });
        const tables = (tRes.data || []).map((t) =>
          this.decorate(t, byTable[t.id], ownReservationsBySeat)
        );
        this.setData({
          tables,
          expandedId: tables.length ? tables[0].id : '',
          loading: false,
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  /** Attach display fields for a real SeatView, including the current occupant. */
  decorateSeat(s, i, ownReservationsBySeat) {
    const state = s.status === 'available' ? 'free' : 'reserved';
    const isGuest = Boolean(s.isGuest);
    const nickname = isGuest ? 'inward会员' : s.nickname || '';
    const gender = s.gender === 'male' || s.gender === 'female' ? s.gender : '';
    const ownReservation = ownReservationsBySeat.get(String(s.id));
    const isMine = state === 'reserved' && Boolean(ownReservation);
    return {
      no: s.name,
      seatId: s.id,
      state,
      label: SEAT_STATE_LABEL[state] || '',
      style: SEAT_POS[i] || '',
      labelTop: i < 3,
      avatarUrl: isGuest ? '/assets/brand/logo-wordmark.png' : s.avatarUrl || '',
      nickname,
      displayName: isMine ? '我' : nickname || '已预约',
      isMine,
      reservationId: ownReservation ? ownReservation.id : '',
      isGuest,
      initial: nickname.trim().charAt(0).toUpperCase(),
      gender,
      genderIcon: gender ? '/assets/icons/gender-' + gender + '.svg' : '',
    };
  },

  /** attach display fields + ring positions to a table */
  decorate(t, realSeats, ownReservationsBySeat) {
    const seats = (realSeats || []).map((s, i) =>
      this.decorateSeat(s, i, ownReservationsBySeat)
    );
    const cap = t.capacity || seats.length;
    const free = seats.filter((s) => s.state === 'free').length;
    return {
      id: t.id,
      name: t.name,
      type: t.type,
      layoutUrl: t.layoutUrl || '',
      reservedText: (t.reserved != null ? t.reserved : cap - free) + '/' + cap,
      updatedText: fmt.dateTime(t.updatedAt, { timeOnly: true }),
      seats,
    };
  },

  onToggleTable(e) {
    const id = e.currentTarget.dataset.id;
    this.clearSelection();
    this.setData({ expandedId: this.data.expandedId === id ? '' : id });
  },

  onSeatTap(e) {
    const { tid, no } = e.currentTarget.dataset;
    const t = this.data.tables.find((x) => x.id === tid);
    if (!t) return;
    const seat = t.seats.find((s) => s.no === no);
    if (!seat) return;
    if (seat.state === 'reserved') {
      if (seat.isMine && seat.reservationId) this.cancelOwnReservation(seat, t);
      return;
    }
    if (seat.state === 'selected') {
      this.clearSelection();
      return;
    }
    this.clearSelection(false); // revert any prior selection, keep sheet handling below
    const tables = this.data.tables;
    const tt = tables.find((x) => x.id === tid);
    const target = tt.seats.find((s) => s.no === no);
    target.state = 'selected';
    target.label = SEAT_STATE_LABEL.selected;
    this.setData({
      tables,
      selected: { tableId: tid, tableName: tt.name, seatNo: no, seatId: target.seatId },
      showConfirm: true,
    });
  },

  cancelOwnReservation(seat, table) {
    const reservationId = seat && seat.reservationId;
    if (!reservationId || this.data.cancellingReservationId) return;
    ui.confirm({
      title: '取消预约',
      content: `确认取消 ${table.name} · ${seat.no}号座位的预约？`,
      confirmText: '取消预约',
    }).then((confirmed) => {
      if (!confirmed) return;
      this.setData({ cancellingReservationId: reservationId });
      ui.showLoading('取消中');
      api.cancelReservation(reservationId, http.uuid())
        .then(() => {
          ui.hideLoading();
          this.setData({ cancellingReservationId: '' });
          this.loadTables(this.data.store.id);
          ui.success('预约已取消');
        })
        .catch((err) => {
          ui.hideLoading();
          this.setData({ cancellingReservationId: '' });
          ui.error((err && err.message) || '取消预约失败');
        });
    });
  },

  clearSelection(commit) {
    const sel = this.data.selected;
    if (sel) {
      const tables = this.data.tables;
      const t = tables.find((x) => x.id === sel.tableId);
      if (t) {
        const seat = t.seats.find((s) => s.no === sel.seatNo);
        if (seat && seat.state === 'selected') {
          seat.state = 'free';
          seat.label = SEAT_STATE_LABEL.free;
        }
      }
      this.setData({ tables });
    }
    if (commit !== false) this.setData({ selected: null, showConfirm: false });
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
	  validation.integer(sel.seatId, { label: '座位', min: 1 });
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
        seatNo: sel.seatNo,
        seatId: sel.seatId,
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

  openTournamentEvent() {
    const event = this.data.tournamentEvent;
    if (!event || !event.id) {
      ui.toast('今日暂无赛事');
      return;
    }
    wx.navigateTo({ url: `/pages/tournament-detail/tournament-detail?id=${event.id}` });
  },
});
