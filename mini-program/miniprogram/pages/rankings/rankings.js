// 排行榜 — 周/月/总榜 + 我的排名
// Reference: design/mini-program/final/member-subpages/11-rankings-v23.png
const api = require('../../services/api');

Page({
  data: {
    loading: true,
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
    this.setData({ scope: e.detail.value });
    this.load();
  },

  load() {
    this.setData({ loading: true });
    // Server GET /mini/rankings takes ?period and returns an ARRAY of
    // RankingEntryView {rank,nickname,score,avatarUrl}. The mock returns an
    // object {myRank,myScore,top[],updatedText}; tolerate both. Aggregate fields
    // (myRank/myScore/updatedText) are mock-only and degrade to hidden/empty.
    api
      .getRankings({ period: this.data.scope })
      .then((res) => {
        const raw = res.data;
        const isArray = Array.isArray(raw);
        const d = isArray ? {} : raw || {};
        const rows = isArray ? raw : d.top || [];
        const top = rows.map((r) => {
          const name = r.name != null ? r.name : r.nickname;
          return {
            rank: r.rank,
            name,
            score: r.score,
            avatarUrl: r.avatarUrl || '',
            initial: (name || '?').slice(0, 1),
            top3: r.rank <= 3,
          };
        });
        this.setData({ loading: false, board: { myRank: d.myRank, myScore: d.myScore, top, updatedText: d.updatedText } });
      })
      .catch(() => this.setData({ loading: false }));
  },
});
