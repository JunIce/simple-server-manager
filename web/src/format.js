// 通用格式化工具

// fmtMB 将 MB 值格式化为易读单位，最多保留 2 位小数。
export function fmtMB(mb) {
  if (mb == null) return '--'
  if (mb >= 1024) return `${trimDecimals(mb / 1024, 2)} GB`
  return `${trimDecimals(mb, 2)} MB`
}

// trimDecimals 保留最多 maxDec 位小数，去掉多余的 0。
export function trimDecimals(value, maxDec = 2) {
  if (value == null || isNaN(value)) return '--'
  if (!Number.isFinite(value)) return '--'
  const v = Number(value)
  if (v === 0) return '0'
  if (v >= 100 || Number.isInteger(v)) return String(v)
  const fixed = v.toFixed(maxDec)
  return String(parseFloat(fixed))
}
