// 会员权益 — 纯 LOGO + 灰阶 VIP 身份 / 当前会员等级 / 成长值
const api = require('../../services/api');

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

Page({
  data: { loading: true, tier: null },

  onLoad() {
    api
      .getBenefitsOverview()
      .then((res) => {
        const d = /** @type {any} */ (res.data || {});
        const levels = d.levels || [];
        const matchedIndex = levels.findIndex((level) =>
          d.currentLevel ? Number(level.level) === Number(d.currentLevel) : level.code === d.current
        );
        const currentIdx = Math.max(0, matchedIndex);
        const currentLevel = levels[currentIdx] || {};
        const nextLevel = levels[currentIdx + 1];
        const growthValue = Number(d.growthValue) || 0;
        const growthTarget = nextLevel ? Number(nextLevel.threshold) || Number(d.growthMax) || 0 : 0;
        const growthPct = growthTarget ? Math.min(100, Math.round((growthValue / growthTarget) * 100)) : 100;
        const currentVer = vipLabel(currentLevel.level || d.currentLevel, currentIdx);
        const currentTone = toneFor(currentIdx, Math.max(1, levels.length));
        const nextVer = nextLevel ? vipLabel(nextLevel.level, currentIdx + 1) : '';

        this.setData({
          loading: false,
          tier: {
            currentVer,
            currentName: memberName(currentLevel.name, currentVer),
            growthValue,
            growthTarget,
            growthPct,
            growthHint: nextLevel
              ? '距 ' + nextVer + ' 还需 ' + Math.max(0, growthTarget - growthValue) + ' 成长值'
              : '已达当前最高等级',
            tone: currentTone.tone,
            ink: currentTone.ink,
            logoClass: currentTone.logoClass,
          },
        });
      })
      .catch(() => this.setData({ loading: false }));
  },
});
