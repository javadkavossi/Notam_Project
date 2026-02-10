let audioContext: AudioContext | null = null

function getAudioContext(): AudioContext | null {
  if (typeof window === 'undefined') return null
  if (!audioContext) {
    audioContext = new (window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext)()
  }
  return audioContext
}

/**
 * پخش یک بوق کوتاه برای اعلان (بدون نیاز به فایل صوتی)
 */
export function playAlertSound(): void {
  const ctx = getAudioContext()
  if (!ctx) return
  try {
    if (ctx.state === 'suspended') {
      ctx.resume().catch(() => {})
    }
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.frequency.value = 880
    osc.type = 'sine'
    gain.gain.setValueAtTime(0.15, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.3)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.3)
  } catch {
    // ignore
  }
}

/**
 * پخش یک صدای بسیار کوتاه برای باز کردن قفل پخش صدا در مرورگر (بعد از کلیک کاربر)
 */
export function unlockAudio(): void {
  const ctx = getAudioContext()
  if (!ctx) return
  try {
    if (ctx.state === 'suspended') {
      ctx.resume().then(() => playAlertSound())
    } else {
      playAlertSound()
    }
  } catch {
    // ignore
  }
}
