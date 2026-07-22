// Status badge: text + tone-driven dot (never color-only — always has a label
// and a shape). Centralizes status→label/tone mapping from constants.
const C = require('../../constants/index');

const LABEL_MAPS = {
  order: C.ORDER_STATUS_LABEL,
  ticket: C.TICKET_STATUS_LABEL,
  coupon: C.COUPON_STATUS_LABEL,
  verify: C.VERIFY_RESULT_LABEL,
  reservation: C.RESERVATION_STATUS_LABEL,
};

const TONE_MAP = Object.assign(
  {},
  C.ORDER_STATUS_TONE,
  { success: 'done', failed: 'danger', void: 'danger' },
  { unused: 'active', pending_verify: 'active', used: 'done', expired: 'neutral', refunded: 'danger' },
  { active: 'active', arrived: 'done', completed: 'done' }
);

Component({
  properties: {
    status: { type: String, value: '' },
    map: { type: String, value: 'order' }, // order|ticket|coupon|verify|reservation
    label: { type: String, value: '' }, // optional override
    plain: { type: Boolean, value: false }, // text-only (no chip background)
  },
  data: { text: '', tone: 'neutral' },
  observers: {
    'status,map,label'(status, map, label) {
      const dict = LABEL_MAPS[map] || C.ORDER_STATUS_LABEL;
      this.setData({
        text: label || dict[status] || status,
        tone: TONE_MAP[status] || 'neutral',
      });
    },
  },
});
