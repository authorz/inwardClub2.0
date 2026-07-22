// Bottom sheet — generic slide-up panel with mask, title and slotted body.
// Reused by: reservation confirm, activity ticket purchase, coin recharge,
// point saving. Pages provide the body via <slot>.
Component({
  options: { multipleSlots: true },
  properties: {
    show: { type: Boolean, value: false },
    title: { type: String, value: '' },
    showClose: { type: Boolean, value: true },
    // rounded top only; content scrolls if too tall
    maxHeight: { type: String, value: '82vh' },
    height: { type: String, value: 'auto' },
    // when false, body renders as a static (non-scrolling) view — for short content
    scrollBody: { type: Boolean, value: true },
  },
  methods: {
    onMask() {
      this.triggerEvent('close');
    },
    onClose() {
      this.triggerEvent('close');
    },
    noop() {},
  },
});
