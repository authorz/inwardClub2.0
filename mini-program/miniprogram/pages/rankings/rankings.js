// 积分排行榜：月榜/总榜按已审核存入积分排名，水上榜按本月盈利积分排名。
// 每一行同时展示会员当前成长值，但成长值不参与排序。
const api = require('../../services/api');
const auth = require('../../utils/auth');
const ui = require('../../utils/ui');

const BOARD_CONFIG = {
  month: {
    title: '月榜',
    periodLabel: '本自然月',
    metricLabel: '存入积分',
    summaryText: '按本自然月审核通过的存入积分排名',
    footText: '月榜按本自然月审核通过的存入积分实时统计',
    emptyTitle: '本月暂无存入积分记录',
    emptySub: '完成积分存入并审核通过后即可进入月榜',
  },
  all: {
    title: '总榜',
    periodLabel: '历史累计',
    metricLabel: '存入积分',
    summaryText: '按历史累计审核通过的存入积分排名',
    footText: '总榜按历史所有审核通过的存入积分实时统计',
    emptyTitle: '暂无存入积分记录',
    emptySub: '完成积分存入并审核通过后即可进入总榜',
  },
  water: {
    title: '水上榜',
    periodLabel: '本自然月',
    metricLabel: '盈利积分',
    summaryText: '按本自然月实际获得的盈利积分排名',
    footText: '水上榜按本自然月审核后实际获得的盈利积分实时统计',
    emptyTitle: '本月暂无盈利积分记录',
    emptySub: '本月获得盈利积分后即可进入水上榜',
  },
};

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
  const config = BOARD_CONFIG[scope] || BOARD_CONFIG.month;
  const memberID = me && Number(me.id || 0);
  const rows = (sourceRows || []).slice(0, 50).map((row, index) => {
    const rank = Number(row.rank || index + 1);
    const name = String(row.nickname || row.name || `会员${row.memberId || ''}`);
    const score = Number(row.score || 0);
    return {
      rank,
      rankText: rank < 10 ? `0${rank}` : String(rank),
      memberId: Number(row.memberId || 0),
      name,
      genderIcon: genderIcon(row.gender),
      score,
      scoreText: formatNumber(score),
      growthText: formatNumber(row.growthValue),
      avatarUrl: row.avatarUrl || '',
      initial: name.slice(0, 1),
      avatarTone: `tone-${(index % 5) + 1}`,
      isMe: memberID > 0 && memberID === Number(row.memberId),
    };
  });

  rows.forEach((item, index) => {
    item.gapText = index === 0
      ? '暂居榜首'
      : `距上一名 ${formatNumber(rows[index - 1].score - item.score + 1)}`;
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
  const gapToPrevious = myRow && previousRow ? previousRow.score - myRow.score + 1 : 0;
  const gapToTopTen = myRow && myRow.rank > 10 && topTenRow ? topTenRow.score - myRow.score + 1 : 0;

  return {
    ...config,
    hasEntries: rows.length > 0,
    podium,
    podiumClass: `count-${podium.length}`,
    list: rows.slice(3),
    total: rows.length,
    championScoreText: rows.length ? rows[0].scoreText : '0',
    myRank: myRow ? myRow.rank : 0,
    myScoreText: myRow ? myRow.scoreText : '0',
    myGrowthText: myRow ? myRow.growthText : '0',
    myAvatarUrl: myRow ? myRow.avatarUrl : (me && me.avatarUrl) || '',
    myInitial: myRow ? myRow.initial : String((me && me.nickname) || '我').slice(0, 1),
    gapToPreviousText: formatNumber(gapToPrevious),
    gapToTopTenText: gapToTopTen > 0 ? formatNumber(gapToTopTen) : '已进入',
    chaseProgress: myRow && previousRow
      ? Math.max(8, Math.min(96, Math.round((myRow.score / previousRow.score) * 100)))
      : 100,
  };
}

Page({
  data: {
    loading: true,
    loggedIn: false,
    scopeOptions: [
      { label: '月榜', value: 'month' },
      { label: '总榜', value: 'all' },
      { label: '水上榜', value: 'water' },
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
