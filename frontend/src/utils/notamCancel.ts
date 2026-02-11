const CANCELLED_REF_REGEX = /CANCELLED NOTAM:\s*([A-Z0-9]+\/\d{2})/i
const REPLACES_REF_REGEX = /REPLACES NOTAM:\s*([A-Z0-9]+\/\d{2})/i
const NOTAMR_REF_REGEX = /NOTAMR\s+([A-Z0-9]+\/\d{2})/

/**
 * Extracts the cancelled NOTAM reference from formatted text or plainText
 * (e.g. "E) NOTAM CANCELLED - CANCELLED NOTAM: H2459/26" or "NOTAM CANCELLED - CANCELLED NOTAM: H2459/26")
 */
export function getCancelledNotamRef(
  formattedText: string | undefined,
  plainText?: string
): string | null {
  const fromFormatted = formattedText && typeof formattedText === 'string'
    ? formattedText.match(CANCELLED_REF_REGEX)
    : null
  if (fromFormatted) return fromFormatted[1].trim()
  const fromPlain = plainText && typeof plainText === 'string'
    ? plainText.match(CANCELLED_REF_REGEX)
    : null
  return fromPlain ? fromPlain[1].trim() : null
}

/**
 * For cancelled NOTAMs, removes Q) line from formatted text so it is not displayed in the list.
 */
export function stripQLineFromFormattedText(text: string | undefined): string {
  if (!text || typeof text !== 'string') return ''
  return text
    .split('\n')
    .filter((line) => !/^\s*Q\)\s/.test(line.trim()))
    .join('\n')
}

/**
 * For cancelled NOTAMs, returns display text that includes which NOTAM was cancelled.
 * Prefers ref from formattedText or plainText; otherwise returns plainText as-is.
 */
export function getCancelledNotamDisplayText(
  plainText: string,
  formattedText: string | undefined
): string {
  const ref = getCancelledNotamRef(formattedText, plainText)
  if (ref) return `NOTAM CANCELLED - CANCELLED NOTAM: ${ref}`
  return plainText
}

/**
 * Extracts the replaced NOTAM reference from formatted text or plainText
 * (e.g. "REPLACES NOTAM: A1617/26" or "NOTAMR A1617/26")
 */
export function getReplacedNotamRef(
  formattedText: string | undefined,
  plainText?: string
): string | null {
  const fromPlain = plainText && typeof plainText === 'string'
    ? plainText.match(REPLACES_REF_REGEX)
    : null
  if (fromPlain) return fromPlain[1].trim()
  const fromFormatted = formattedText && typeof formattedText === 'string'
    ? formattedText.match(REPLACES_REF_REGEX) ?? formattedText.match(NOTAMR_REF_REGEX)
    : null
  return fromFormatted ? fromFormatted[1].trim() : null
}

/**
 * For replace NOTAMs, returns display text without the "REPLACES NOTAM: xxx" prefix for body.
 */
export function getReplacedNotamDisplayText(
  plainText: string,
  formattedText: string | undefined
): string {
  const ref = getReplacedNotamRef(formattedText, plainText)
  if (ref) {
    const prefix = `REPLACES NOTAM: ${ref}\n\n`
    if (plainText.startsWith(prefix)) return plainText.slice(prefix.length)
    return plainText.replace(new RegExp(`^REPLACES NOTAM:\\s*${ref}\\s*\\n?\\n?`, 'i'), '')
  }
  return plainText
}
