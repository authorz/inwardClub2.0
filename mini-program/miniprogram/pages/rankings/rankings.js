// 微信支付成长值排行榜：¥1 微信实付业务金额 = 1 成长值，展示周榜、月榜和总榜前 50 名。
const api = require('../../services/api');
const auth = require('../../utils/auth');
const ui = require('../../utils/ui');

function formatNumber(value) {
  return String(value || 0).replace(/\B(?=(\d{3})+(?!\d))/g, ',');
}

function genderIcon(gender) {
  if (gender === 'female') {
    return '/assets/icons/gender-female.svg';
  }
  if (gender === 'male') {
    return '/assets/icons/gender-male.svg';
  }
  return '';
}

function buildBoard(scope, sourceRows, me) {
  const scopeLabel = scope === 'week' ? '本周' : scope === 'all' ? '总' : '本月';
  const memberID = me && Number(me.id || 0);
  const rows = (sourceRows || []).slice(0, 50).map((row, index) => {
    const rank = Number(row.rank || index + 1);
    const name = String(row.nickname || row.name || `会员${row.memberId || ''}`);
    return {
      rank,
      rankText: rank < 10 ? `0${rank}` : String(rank),
      memberId: Number(row.memberId || 0),
      name,
      genderIcon: genderIcon(row.gender),
      growth: Number(row.score || 0),
      growthText: formatNumber(row.score),
      avatarUrl: row.avatarUrl || '',
      initial: name.slice(0, 1),
      avatarTone: `tone-${(index % 5) + 1}`,
      isMe: memberID > 0 && memberID === Number(row.memberId),
    };
  });

  rows.forEach((item, index) => {
    item.gapText = index === 0
      ? '暂居榜首'
      : `距上一名 ${formatNumber(rows[index - 1].growth - item.growth + 1)}`;
  });

  let podium = [];
  if (rows.length >= 3) podium = [rows[1], rows[0], rows[2]];
  else if (rows.length === 2) podium = [rows[1], rows[0]];
  else if (rows.length === 1) podium = [rows[0]];
  podium = podium.map((item) => ({
    ...item,
    placeClass: item.rank === 1 ? 'is-first' : item.rank === 2 ? 'is-second' : 'is-third',
    medalText: item.rank === 1 ? '冠军' : item.rank === 2 ? '亚军' : '季军',
  }));

  const myIndex = rows.findIndex((item) => item.isMe);
  const myRow = myIndex >= 0 ? rows[myIndex] : null;
  const previousRow = myIndex > 0 ? rows[myIndex - 1] : null;
  const topTenRow = rows.length >= 10 ? rows[9] : null;
  const gapToPrevious = myRow && previousRow ? previousRow.growth - myRow.growth + 1 : 0;
  const gapToTopTen = myRow && myRow.rank > 10 && topTenRow ? topTenRow.growth - myRow.growth + 1 : 0;

  return {
    scopeLabel,
    hasEntries: rows.length > 0,
    podium,
    podiumClass: `count-${podium.length}`,
    list: rows.slice(3),
    total: rows.length,
    championGrowthText: rows.length ? rows[0].growthText : '0',
    myRank: myRow ? myRow.rank : 0,
    myGrowthText: myRow ? myRow.growthText : '0',
    myAvatarUrl: myRow ? myRow.avatarUrl : (me && me.avatarUrl) || '',
    myInitial: myRow ? myRow.initial : String((me && me.nickname) || '我').slice(0, 1),
    gapToPreviousText: formatNumber(gapToPrevious),
    gapToTopTenText: gapToTopTen > 0 ? formatNumber(gapToTopTen) : '已进入',
    chaseProgress: myRow && previousRow
      ? Math.max(8, Math.min(96, Math.round((myRow.growth / previousRow.growth) * 100)))
      : 100,
  };
}

Page({
  data: {
    loading: true,
    loggedIn: false,
    scopeOptions: [
      { label: '周榜', value: 'week' },
      { label: '月榜', value: 'month' },
      { label: '总榜', value: 'all' },
    ],
    scope: 'month',
    board: null,
  },

  onLoad() {
    this.load();
  },

  onScopeChange(e) {
    const scope = e.detail.value;
    this.setData({ scope });
    this.load(scope);
  },

  load(scope) {
    const rankingScope = scope || this.data.scope;
    const loggedIn = auth.isLoggedIn();
    const meRequest = loggedIn ? api.getMe().catch(() => ({ data: null })) : Promise.resolve({ data: null });
    this.setData({ loading: true, loggedIn });
    Promise.all([api.getRankings({ period: rankingScope }), meRequest])
      .then(([rankingsRes, meRes]) => {
        const rows = Array.isArray(rankingsRes.data) ? rankingsRes.data : [];
        this.setData({
          loading: false,
          board: buildBoard(rankingScope, rows, meRes.data),
        });
      })
      .catch((err) => {
        this.setData({ loading: false, board: buildBoard(rankingScope, [], null) });
        ui.toast((err && err.message) || '排行榜加载失败');
      });
  },
});
