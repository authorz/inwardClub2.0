// 邀请好友 — 邀请码 / 分享 / 规则 / 邀请记录
// Reference: design/mini-program/final/member-subpages/10-invitations-v23.png
const api = require('../../services/api');
const auth = require('../../utils/auth');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');

Page({
  data: {
    loading: true,
    info: null,
    bindCode: '',
    binding: false,
  },

  onLoad(options) {
    if (!auth.isLoggedIn()) {
      const inviteCode = (options && options.invite) || '';
      const url = inviteCode
        ? '/pages/index/index?invite=' + encodeURIComponent(inviteCode)
        : '/pages/index/index';
      wx.reLaunch({ url });
      return;
    }
    // Server GET /mini/invitations returns an ARRAY of InvitationView
    // {memberId,nickname,avatarUrl,joinedAt} — no invite code and no
    // effective/pending/rules aggregates. The invite code comes from the member
    // profile (MemberProfile.inviteCode).
    Promise.all([
      api.getInvitations().catch(() => ({ data: [] })),
      api.getMe().catch(() => ({ data: {} })),
    ])
      .then(([invRes, meRes]) => {
        const rows = Array.isArray(invRes.data) ? invRes.data : [];
        const me = meRes.data || {};
        const total = invRes.meta && invRes.meta.total;
        const records = rows.map((r) => {
          const name = r.name != null ? r.name : r.nickname || '会员';
          return {
            id: r.id != null ? r.id : r.memberId,
            name,
            avatarUrl: r.avatarUrl || '',
            avatarText: String(name).slice(0, 1),
            registeredAt: fmt.dateTime(r.joinedAt || r.date, { relative: false }),
          };
        });
        this.setData({
          loading: false,
          info: {
            nickname: me.nickname || me.nickName || '会员',
            inviteCode: me.inviteCode || '',
            invited: total != null ? total : records.length,
            inviterBound: !!me.inviterBound,
            rules: [],
            records,
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  copyCode() {
    if (this.data.info && this.data.info.inviteCode) ui.copy(this.data.info.inviteCode, '邀请码已复制');
  },

  onBindCodeInput(e) {
    this.setData({ bindCode: (e.detail.value || '').trim() });
  },

  bindInvitation() {
    if (this.data.binding) return;
    const inviteCode = (this.data.bindCode || '').trim();
    if (!inviteCode) return ui.toast('请输入邀请码');

    ui.confirm({
      title: '确认绑定',
      content: '邀请码绑定后不可更改，请确认填写无误。',
      confirmText: '确认绑定',
    }).then((confirmed) => {
      if (!confirmed) return;
      this.setData({ binding: true });
      api
        .bindInvitation({ inviteCode })
        .then(() => {
          this.setData({
            binding: false,
            bindCode: '',
            'info.inviterBound': true,
          });
          ui.success('邀请码绑定成功');
        })
        .catch((err) => {
          if (err && err.code === 'CONFLICT') {
            this.setData({
              binding: false,
              bindCode: '',
              'info.inviterBound': true,
            });
            ui.toast('该账号已绑定邀请码');
            return;
          }
          this.setData({ binding: false });
          if (err && err.code === 'NOT_FOUND') return ui.error('邀请码不存在');
          if (err && err.code === 'INVALID_ARGUMENT') return ui.error('不能绑定自己的邀请码');
          ui.error('邀请码绑定失败，请稍后重试');
        });
    });
  },

  inviteShareData() {
    const info = this.data.info || {};
    return {
      inviteCode: info.inviteCode || '',
      title: (info.nickname || '会员') + '邀请你加入inwardClub会员',
    };
  },

  onShareAppMessage() {
    const share = this.inviteShareData();
    return {
      title: share.title,
      path: '/pages/index/index?invite=' + encodeURIComponent(share.inviteCode),
      imageUrl: 'https://assets.inwardclub.com/logo/logo-2.jpg',
    };
  },

  onShareTimeline() {
    const share = this.inviteShareData();
    return {
      title: share.title,
      query: 'invite=' + encodeURIComponent(share.inviteCode),
      imageUrl: 'https://assets.inwardclub.com/logo/logo-2.jpg',
    };
  },
});
