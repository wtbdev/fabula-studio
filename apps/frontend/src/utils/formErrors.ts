export const getFormValidationMessage = (error: unknown) => {
  if (!Array.isArray(error)) return ''

  for (const group of error) {
    if (!Array.isArray(group)) continue

    for (const item of group) {
      if (item && typeof item === 'object' && 'message' in item) {
        const message = (item as { message?: unknown }).message
        if (typeof message === 'string' && message.trim()) return message
      }
    }
  }

  return '请检查表单输入'
}
