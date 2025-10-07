/**
 * 格式化工具函数
 */

/**
 * 格式化长ID为缩短版本
 * @param id 完整ID
 * @param length 显示长度，默认8个字符
 * @returns 缩短后的ID
 */
export function formatId(id: string | undefined | null, length: number = 8): string {
  if (!id) return ''
  return id.length > length ? `${id.substring(0, length)}...` : id
}

/**
 * 格式化时间戳为时间字符串
 * @param timestamp Unix时间戳（秒）
 * @returns 格式化的时间字符串
 */
export function formatTime(timestamp: number): string {
  if (!timestamp) return ''
  return new Date(timestamp * 1000).toLocaleTimeString()
}

/**
 * 格式化时间戳为日期字符串
 * @param timestamp Unix时间戳（秒）
 * @returns 格式化的日期字符串
 */
export function formatDate(timestamp: number): string {
  if (!timestamp) return ''
  return new Date(timestamp * 1000).toLocaleDateString()
}

/**
 * 格式化时间戳为完整日期时间字符串
 * @param timestamp Unix时间戳（秒）
 * @returns 格式化的日期时间字符串
 */
export function formatDateTime(timestamp: number): string {
  if (!timestamp) return ''
  return new Date(timestamp * 1000).toLocaleString()
}

/**
 * 格式化文件大小
 * @param bytes 字节数
 * @returns 格式化的文件大小字符串
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

/**
 * 复制文本到剪贴板
 * @param text 要复制的文本
 * @returns Promise<boolean> 是否复制成功
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch (err) {
    console.error('Failed to copy:', err)
    return false
  }
}

