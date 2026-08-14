Component({
  properties: {
    title: {
      type: String,
      value: '',
    },
    back: {
      type: Boolean,
      value: true,
    },
    leftText: {
      type: String,
      value: '',
    },
    storeName: {
      type: String,
      value: '',
    },
    storeAddress: {
      type: String,
      value: '',
    },
    storeHours: {
      type: String,
      value: '',
    },
  },

  data: {
    statusBarHeight: 20,
    contentHeight: 44,
    sideGap: 96,
  },

  lifetimes: {
    attached() {
      try {
        const windowInfo = wx.getWindowInfo();
        const capsule = wx.getMenuButtonBoundingClientRect();
        const statusBarHeight = windowInfo.statusBarHeight || 20;
        const verticalGap = Math.max(capsule.top - statusBarHeight, 4);
        this.setData({
          statusBarHeight,
          contentHeight: capsule.height + verticalGap * 2,
          sideGap: Math.max(windowInfo.windowWidth - capsule.left + 8, 96),
        });
      } catch {
        // Keep conservative defaults on older base libraries.
      }
    },
  },

  methods: {
    onLeftTap() {
      this.triggerEvent('lefttap');
    },

    onStoreTap() {
      this.triggerEvent('storetap');
    },

    onBack() {
      const pages = getCurrentPages();
      if (pages.length > 1) {
        wx.navigateBack();
        return;
      }
      wx.switchTab({ url: '/pages/home/home' });
    },
  },
});
