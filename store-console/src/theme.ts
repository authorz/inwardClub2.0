/**
 * Naive UI 主题覆盖，映射 design token，落实黑白灰运营工作台风格。
 * 主色为黑，无彩色促销风、无渐变。
 */

import type { GlobalThemeOverrides } from 'naive-ui'

export const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#1a1a1a',
    primaryColorHover: '#333333',
    primaryColorPressed: '#000000',
    primaryColorSuppl: '#333333',
    borderRadius: '6px',
    borderColor: '#e5e5e5',
    textColorBase: '#1a1a1a',
    bodyColor: '#f5f5f5',
    fontSize: '14px',
    fontFamily:
      "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif",
  },
  Button: {
    textColorPrimary: '#ffffff',
    borderRadiusmedium: '6px',
  },
  Card: {
    borderRadius: '6px',
  },
  DataTable: {
    thColor: '#fafafa',
    thTextColor: '#595959',
    borderRadius: '6px',
  },
  Menu: {
    itemTextColorActive: '#1a1a1a',
    itemTextColorActiveHover: '#1a1a1a',
    itemColorActive: '#f0f0f0',
    itemColorActiveHover: '#f0f0f0',
    itemColorActiveCollapsed: '#f0f0f0',
    itemIconColorActive: '#1a1a1a',
    itemIconColorActiveHover: '#1a1a1a',
  },
}
