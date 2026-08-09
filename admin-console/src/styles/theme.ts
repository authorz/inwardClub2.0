import type { GlobalThemeOverrides } from 'naive-ui'

/**
 * Naive UI 主题覆写（集中定义）。
 * 将黑白灰设计 token 映射到 Naive UI，保证全站按钮/表格/表单风格一致，
 * 避免营销风的默认蓝色主色。
 */
const PRIMARY = '#1a1a1a'
const PRIMARY_HOVER = '#333333'
const PRIMARY_PRESSED = '#000000'

export const adminThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: PRIMARY,
    primaryColorHover: PRIMARY_HOVER,
    primaryColorPressed: PRIMARY_PRESSED,
    primaryColorSuppl: PRIMARY_HOVER,
    borderRadius: '6px',
    borderRadiusSmall: '4px',
    fontSize: '14px',
    textColorBase: '#1a1a1a',
    bodyColor: '#f5f5f5',
    successColor: '#2f8f4e',
    warningColor: '#b8860b',
    errorColor: '#c0392b',
    infoColor: '#3a6ea5',
  },
  Button: {
    textColorPrimary: '#ffffff',
    borderRadiusMedium: '6px',
  },
  DataTable: {
    thColor: '#fafafa',
    thTextColor: '#5c5c5c',
    thFontWeight: '600',
    borderColor: '#e5e5e5',
  },
  Layout: {
    siderColor: '#ffffff',
    headerColor: '#ffffff',
  },
  Menu: {
    itemColorActive: 'transparent',
    itemColorActiveHover: '#f3f7ff',
    itemColorHover: '#f5f5f5',
    itemTextColorActive: '#1677ff',
    itemTextColorActiveHover: '#1677ff',
    itemIconColorActive: '#1677ff',
    itemIconColorActiveHover: '#1677ff',
  },
}
