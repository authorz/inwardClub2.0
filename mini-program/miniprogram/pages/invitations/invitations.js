// 邀请好友 — 邀请码 / 分享 / 规则 / 邀请记录
// Reference: design/mini-program/final/member-subpages/10-invitations-v23.png
const api = require('../../services/api');
const ui = require('../../utils/ui');

const RULE_ICON = { gift: 'ic-ic-gift', doc: 'ic-ic-doc', shield: 'ic-ic-shield' };
const REC_STATUS = { effective: '已生效', pending: '待结算' };

Page({
  data: { loading: true, info: null },

  onLoad() {
    // Server GET /mini/invitations returns an ARRAY of InvitationView
    // {memberId,nickname,avatarUrl,joinedAt} — no invite code and no
    // effective/pending/rules aggregates. The invite code comes from the member
    // profile (MemberProfile.inviteCode). The mock returns a richer object;
    // tolerate both. Server-absent aggregates degrade to zero/hidden.
    Promise.all([
      api.getInvitations().catch(() => ({ data: [] })),
      api.getMe().catch(() => ({ data: {} })),
    ])
      .then(([invRes, meRes]) => {
        const raw = invRes.data;
        const isArray = Array.isArray(raw);
        const d = isArray ? {} : raw || {};
        const rows = isArray ? raw : d.records || [];
        const me = meRes.data || {};
        const records = rows.map((r) => ({
          id: r.id != null ? r.id : r.memberId,
          name: r.name != null ? r.name : r.nickname,
          date: r.date != null ? r.date : r.joinedAt,
          statusLabel: r.status ? REC_STATUS[r.status] || r.status : '',
          effective: r.status === 'effective',
        }));
        this.setData({
          loading: false,
          info: {
            inviteCode: d.inviteCode || me.inviteCode || '',
            invited: d.invited != null ? d.invited : records.length,
            effective: d.effective || 0,
            pending: d.pending || 0,
            rules: (d.rules || []).map((r) => ({ icon: RULE_ICON[r.icon] || 'ic-ic-gift', title: r.title, desc: r.desc })),
            records,
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  copyCode() {
    if (this.data.info && this.data.info.inviteCode) ui.copy(this.data.info.inviteCode, '邀请码已复制');
  },

  onShareAppMessage() {
    const code = this.data.info ? this.data.info.inviteCode : '';
    return { title: '邀请你加入 Inward Club', path: '/pages/index/index?invite=' + code };
  },
});
