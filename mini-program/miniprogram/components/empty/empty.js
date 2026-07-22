// Empty state — one component for every "no data" view across the app.
Component({
  properties: {
    text: { type: String, value: '暂无数据' },
    hint: { type: String, value: '' },
    compact: { type: Boolean, value: false },
    icon: { type: String, value: '/assets/empty/activity-empty.png' },
  },
});
