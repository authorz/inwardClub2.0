// 会员权益 — 可滑动灰阶 VIP 身份 / 等级预览 / 成长值
const api = require('../../services/api');

const MEMBER_PRIVILEGES = [
  { name: '签到积分', icon: '/pages/benefits/assets/sign-in-points.svg' },
  { name: '月度小吃券', icon: '/pages/benefits/assets/monthly-snack-coupon.svg' },
  { name: '专属酒杯', icon: '/pages/benefits/assets/exclusive-wine-glass.svg' },
  { name: 'AI 挑战', icon: '/pages/benefits/assets/ai-challenge.svg' },
  { name: '3000 积分', icon: '/pages/benefits/assets/points-3000.svg' },
  { name: '更多奖励', icon: '/pages/benefits/assets/more-rewards.svg' },
];

// 客户指定的黑白灰等级色阶：VIP1 从浅灰开始，等级越高越接近纯黑。
const VIP_TONES = ['#DDDDDD', '#BBBBBB', '#999999', '#7F7F7F', '#666666', '#4D4D4D', '#333333', '#000000'];

function toneFor(index, total) {
  const toneIndex = total > 1 ? Math.round((index * (VIP_TONES.length - 1)) / (total - 1)) : 0;
  const tone = VIP_TONES[toneIndex];
  const dark = toneIndex >= 4;
  return {
    tone,
    ink: dark ? '#FFFFFF' : '#111111',
    logoClass: dark ? 'is-light' : 'is-dark',
  };
}

function vipLabel(level, index) {
  return 'VIP' + (Number(level) || index + 1);
}

function memberName(name, ver) {
  const value = String(name || '').replace(/^VIP\s*\d+\s*/i, '').trim();
  return value || ver + ' 会员';
}

function tierView(tiers, selectedIndex, currentIndex, growthValue) {
  const selected = tiers[selectedIndex] || tiers[0];
  if (!selected) return null;

  const isCurrent = selectedIndex === currentIndex;
  const isUnlocked = selectedIndex < currentIndex;
  const nextTier = tiers[selectedIndex + 1];
  const growthTarget = isCurrent
    ? (nextTier ? nextTier.threshold : 0)
    : selected.threshold;
  const progressValue = isUnlocked ? growthTarget : growthValue;
  const growthPct = isUnlocked
    ? 100
    : growthTarget
      ? Math.min(100, Math.round((growthValue / growthTarget) * 100))
      : 100;

  let growthHint;
  if (isUnlocked) {
    growthHint = '该等级已解锁';
  } else if (isCurrent) {
    growthHint = nextTier
      ? '距 ' + nextTier.currentVer + ' 还需 ' + Math.max(0, growthTarget - growthValue) + ' 成长值'
      : '已达当前最高等级';
  } else {
    growthHint = '距 ' + selected.currentVer + ' 还需 ' + Math.max(0, growthTarget - growthValue) + ' 成长值';
  }

  return Object.assign({}, selected, {
    isCurrent,
    growthLabel: isCurrent ? '成长值' : '解锁成长值',
    growthDisplay: isCurrent ? growthValue : growthTarget,
    growthTarget,
    progressValue,
    growthPct,
    growthHint,
  });
}

Page({
  data: {
    loading: true,
    tiers: [],
    activeTierIndex: 0,
    tier: null,
    privileges: MEMBER_PRIVILEGES,
  },

  onLoad() {
    api
      .getBenefitsOverview()
      .then((res) => {
        const d = /** @type {any} */ (res.data || {});
        const levels = (d.levels && d.levels.length)
          ? d.levels
          : [{ level: Number(d.currentLevel) || 1, name: '', threshold: 0 }];
        const matchedIndex = levels.findIndex((level) =>
          d.currentLevel ? Number(level.level) === Number(d.currentLevel) : level.code === d.current
        );
        const currentIdx = Math.max(0, matchedIndex);
        const growthValue = Number(d.growthValue) || 0;
        const tiers = levels.map((level, index) => {
          const currentVer = vipLabel(level.level, index);
          const currentTone = toneFor(index, levels.length);
          return {
            level: Number(level.level) || index + 1,
            currentVer,
            currentNo: String(Number(level.level) || index + 1),
            currentName: memberName(level.name, currentVer),
            threshold: Number(level.threshold) || 0,
            isCurrent: index === currentIdx,
            tone: currentTone.tone,
            ink: currentTone.ink,
            logoClass: currentTone.logoClass,
          };
        });

        this.currentTierIndex = currentIdx;
        this.growthValue = growthValue;

        this.setData({
          loading: false,
          tiers,
          activeTierIndex: currentIdx,
          tier: tierView(tiers, currentIdx, currentIdx, growthValue),
        });
      })
      .catch(() => this.setData({ loading: false }));
  },

  onTierChange(e) {
    const selectedIndex = Number(e.detail.current) || 0;
    this.setData({
      activeTierIndex: selectedIndex,
      tier: tierView(this.data.tiers, selectedIndex, this.currentTierIndex || 0, this.growthValue || 0),
    });
  },
});
