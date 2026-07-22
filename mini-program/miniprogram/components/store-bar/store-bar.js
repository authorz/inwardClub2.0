// Store context bar. Reused on home (最近门店: 切换门店 / 导航) and ordering
// (store + table + scan). Members can switch store here; staff pages never use
// the switch affordance.
const { distance } = require('../../utils/format');

Component({
  properties: {
    name: { type: String, value: '' },
    distanceMeters: { type: Number, value: 0 },
    subtitle: { type: String, value: '' }, // e.g. 桌台/座位
    showSwitch: { type: Boolean, value: false },
    showNav: { type: Boolean, value: false },
    showScan: { type: Boolean, value: false },
    nameArrow: { type: Boolean, value: false },
    flat: { type: Boolean, value: false }, // no card background
  },
  data: { distanceText: '' },
  observers: {
    distanceMeters(v) {
      this.setData({ distanceText: v ? '距您 ' + distance(v) : '' });
    },
  },
  methods: {
    onSwitch() { this.triggerEvent('switch'); },
    onNav() { this.triggerEvent('nav'); },
    onScan() { this.triggerEvent('scan'); },
    onName() { this.triggerEvent('tapname'); },
  },
});
