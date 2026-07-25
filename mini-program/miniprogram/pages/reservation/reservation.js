// 预约 — 多桌 · 每张德州扑克桌 9 座 · 座位确认弹层
// Reference: design/mini-program/final/reservation/05-reservation-final-multi-table.png
//            design/mini-program/final/reservation/05-reservation-final-confirm-sheet.png
const api = require('../../services/api');
const storeCtx = require('../../utils/store-context');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const { loadProfile } = require('../../utils/member-profile');
const { SEAT_STATE_LABEL } = require('../../constants/index');

// Fixed 9-seat ring positions (percent of the felt container), seat 1..9.
const SEAT_POS = [
  'left:24%;top:6%;',
  'left:50%;top:1%;',
  'left:76%;top:6%;',
  'left:92%;top:40%;',
  'left:83%;top:74%;',
  'left:61%;top:88%;',
  'left:39%;top:88%;',
  'left:17%;top:74%;',
  'left:8%;top:40%;',
];

const TIME_OPTIONS = ['今天 20:00', '今天 21:00', '今天 22:00', '明天 20:00', '明天 21:00'];

Page({
  data: {
    loading: true,
    store: null,
    timeText: TIME_OPTIONS[0],
    timeOptions: TIME_OPTIONS,
    tables: [],
    expandedId: '',
    legend: [
      { state: 'free', label: '空闲' },
      { state: 'reserved', label: '已预约' },
      { state: 'selected', label: '已选择' },
    ],
    showConfirm: false,
    submitting: false,
    selected: null, // { tableId, tableName, seatNo }
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
    }
  },

  load() {
    // Resolve the current store (persisted pick, else nearest) before loading its
    // tables — reservations are store-scoped.
    storeCtx.ensureStore().then((store) => {
      this.setData({ store });
      if (store) this.loadTables(store.id);
      else this.setData({ loading: false });
    });
  },

  loadTables(storeId) {
    this.setData({ loading: true });
    // Tables (TableView {id,name,capacity,status}) carry no seats; seats come
    // from a separate endpoint (SeatView {id,tableId?,name,status}). Group the
    // real seats under their table.
    Promise.all([api.getTables(storeId), api.getSeats(storeId)])
      .then(([tRes, sRes]) => {
        const byTable = {};
        (sRes.data || []).forEach((seat) => {
          const key = seat.tableId != null ? seat.tableId : '';
          (byTable[key] = byTable[key] || []).push(seat);
        });
        const tables = (tRes.data || []).map((t) => this.decorate(t, byTable[t.id]));
        this.setData({
          tables,
          expandedId: tables.length ? tables[0].id : '',
          loading: false,
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  /** Attach display fields for a real SeatView {id,name,status}. */
  decorateSeat(s, i) {
    const state = s.status === 'available' ? 'free' : 'reserved';
    return {
      no: s.name,
      seatId: s.id,
      state,
      label: SEAT_STATE_LABEL[state] || '',
      style: SEAT_POS[i] || '',
      avatarUrl: '',
      nickname: '',
      initial: '',
      gender: '',
      genderIcon: '',
    };
  },

  /** attach display fields + ring positions to a table */
  decorate(t, realSeats) {
    const seats = (realSeats || []).map((s, i) => this.decorateSeat(s, i));
    const cap = t.capacity || seats.length;
    const free = seats.filter((s) => s.state === 'free').length;
    return {
      id: t.id,
      name: t.name,
      type: t.type,
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

  onPickTime(e) {
    this.setData({ timeText: this.data.timeOptions[e.detail.value] });
  },

  onSeatTap(e) {
    const { tid, no } = e.currentTarget.dataset;
    const t = this.data.tables.find((x) => x.id === tid);
    if (!t) return;
    const seat = t.seats.find((s) => s.no === no);
    if (!seat || seat.state === 'reserved') return;
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
    this.setData({ submitting: true });
    ui.showLoading('提交中');
    api
      .createReservation({
        storeId: store && store.id,
        tableId: sel.tableId,
        tableName: sel.tableName,
        seatNo: sel.seatNo,
        seatId: sel.seatId,
        timeText: this.data.timeText,
      })
      .then(() => {
        ui.hideLoading();
        const tables = this.data.tables;
        const t = tables.find((x) => x.id === sel.tableId);
        if (t) {
          const seat = t.seats.find((s) => s.no === sel.seatNo);
          if (seat) {
            const me = loadProfile() || {};
            const nickname = me.nickname || '我';
            const gender = me.gender === 'male' || me.gender === 'female' ? me.gender : '';
            seat.state = 'reserved';
            seat.label = SEAT_STATE_LABEL.reserved;
            seat.avatarUrl = me.avatarUrl || '';
            seat.nickname = nickname;
            seat.initial = nickname.trim().charAt(0).toUpperCase();
            seat.gender = gender;
            seat.genderIcon = gender ? '/assets/icons/gender-' + gender + '.svg' : '';
          }
        }
        this.setData({ tables, showConfirm: false, selected: null, submitting: false });
        ui.success('预约成功');
      })
      .catch((err) => {
        ui.hideLoading();
        this.setData({ submitting: false });
        ui.error((err && err.message) || '预约失败');
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
