// 邀请好友 — 邀请码 / 分享 / 规则 / 邀请记录
// Reference: design/mini-program/final/member-subpages/10-invitations-v23.png
const api = require('../../services/api');
const auth = require('../../utils/auth');
const ui = require('../../utils/ui');
const fmt = require('../../utils/format');
const validation = require('../../utils/validation');

function formatRate(basisPoints) {
  const percent = basisPoints / 100;
  const text = Number.isInteger(percent) ? String(percent) : percent.toFixed(2).replace(/0+$/, '').replace(/\.$/, '');
  return `${text}%`;
}

function rewardView(config) {
  const enabled = !!(config && config.enabled);
  const firstCoins = Number((config && config.firstLowSpendRewardCoins) || 0);
  const firstPoints = Number((config && config.firstLowSpendRewardPoints) || 0);
  const rateBasisPoints = Number((config && config.commissionRateBasisPoints) || 0);
  const firstParts = [];
  if (firstCoins > 0) firstParts.push(`${firstCoins} 金币`);
  if (firstPoints > 0) firstParts.push(`${firstPoints} 积分`);
  const firstSummary = firstParts.join(' + ');
  const rateText = formatRate(rateBasisPoints);
  const rules = [];
  if (firstSummary) {
    rules.push(`好友绑定邀请码并首次完成门店低消后，邀请人获得 ${firstSummary}。`);
  }
  if (rateBasisPoints > 0) {
    rules.push(`好友绑定后每笔微信支付（含金币充值及绑定会员的门店微信收款），邀请人获得 ${rateText} 的金币奖励；金币支付、支付宝和现金不计入。`);
  }
  const hasFirstReward = !!firstSummary;
  const hasCommission = rateBasisPoints > 0;
  return {
    enabled,
    firstCoins,
    firstPoints,
    firstSummary,
    rateText,
    cardTitle: hasFirstReward && hasCommission ? '首次低消 + 持续返佣' : hasFirstReward ? '首次低消奖励' : '持续微信支付返佣',
    targetText: hasCommission ? `${rateText} 返金币` : '低消达标',
    note: hasCommission
      ? `好友绑定后，每笔微信支付（含金币充值及绑定会员的门店微信收款）都会按 ${rateText} 为邀请人累计金币；金币支付、支付宝和现金不计入。`
      : `好友绑定并首次完成门店低消后，邀请人可获得 ${firstSummary}。`,
    awardStep: hasCommission
      ? `邀请人首奖 ${firstSummary || '按配置发放'}，后续继续累计金币`
      : `邀请人获得 ${firstSummary}`,
    rules: rules.map((copy, index) => ({ no: index < 9 ? `0${index + 1}` : String(index + 1), copy })),
  };
}

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
      api.getInvitationRewardConfig().catch(() => ({ data: { enabled: false } })),
    ])
      .then(([invRes, meRes, rewardRes]) => {
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
            reward: rewardView(rewardRes.data || {}),
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
    let inviteCode;
	try {
	  inviteCode = validation.inviteCode(this.data.bindCode, false);
	} catch (err) {
	  return ui.toast(err.message);
	}

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
