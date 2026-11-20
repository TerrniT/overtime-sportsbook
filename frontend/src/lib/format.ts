const moneyFormat = new Intl.NumberFormat('ru-RU', {
  style: 'currency',
  currency: 'RUB',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

const moneyWholeFormat = new Intl.NumberFormat('ru-RU', {
  style: 'currency',
  currency: 'RUB',
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
})

const oddsFormat = new Intl.NumberFormat('ru-RU', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

const integerFormat = new Intl.NumberFormat('ru-RU', {
  maximumFractionDigits: 0,
})

const timeFormat = new Intl.DateTimeFormat('ru-RU', {
  hour: '2-digit',
  minute: '2-digit',
})

const dateLongFormat = new Intl.DateTimeFormat('ru-RU', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})

const dateShortFormat = new Intl.DateTimeFormat('ru-RU', {
  day: 'numeric',
  month: 'short',
})

const dateTimeFormat = new Intl.DateTimeFormat('ru-RU', {
  day: '2-digit',
  month: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
})

const kickoffFullFormat = new Intl.DateTimeFormat('ru-RU', {
  weekday: 'short',
  day: 'numeric',
  month: 'long',
  hour: '2-digit',
  minute: '2-digit',
})

function validDate(value: string | Date): Date | null {
  const date = value instanceof Date ? value : new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

/** Integer kopecks to a full rouble amount: 2500000 → «25 000,00 ₽». */
export function formatMoney(minor: number, options: { whole?: boolean } = {}): string {
  const safe = Number.isFinite(minor) ? Math.trunc(minor) : 0
  const formatter = options.whole ? moneyWholeFormat : moneyFormat
  return formatter.format(safe / 100)
}

/** Integer kopecks to whole roubles, for the header balance and stake chips. */
export function formatMoneyWhole(minor: number): string {
  return formatMoney(minor, { whole: true })
}

/** 1950 → «1,95». Prices are always shown with two decimals. */
export function formatOdds(priceMilli: number): string {
  return oddsFormat.format(priceMilli / 1000)
}

/** «Арсенал — Челси», matching the label the server stores on a bet. */
export function formatMatchLabel(match: { homeTeam: string; awayTeam: string }): string {
  return `${match.homeTeam} — ${match.awayTeam}`
}

/** Formats an already-decimal odds value, used by the animated ticker. */
export function formatOddsValue(value: number): string {
  return oddsFormat.format(value)
}

/** Formats a rouble value, used by the balance ticker. */
export function formatMoneyValue(value: number): string {
  return moneyWholeFormat.format(value)
}

/** Formats a rouble value with kopecks, used by the payout ticker. */
export function formatPayoutValue(value: number): string {
  return moneyFormat.format(value)
}

export function formatKickoffFull(iso: string): string {
  const date = validDate(iso)
  return date ? kickoffFullFormat.format(date) : '—'
}

export function formatDateLong(date: Date = new Date()): string {
  return dateLongFormat.format(date).replace(/\s*г\.$/, '')
}

export function formatDateShort(iso: string): string {
  const date = validDate(iso)
  return date ? dateShortFormat.format(date) : '—'
}

export function isToday(iso: string, now: Date = new Date()): boolean {
  const date = validDate(iso)
  if (!date) return false
  return (
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate()
  )
}

function startOfLocalDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
}

/** Kickoff day relative to now: «сегодня», «завтра», «вчера» or «21 нояб.». */
export function formatKickoffDay(iso: string, now: Date = new Date()): string {
  const date = validDate(iso)
  if (!date) return '—'
  const days = Math.round((startOfLocalDay(date) - startOfLocalDay(now)) / 86_400_000)
  if (days === 0) return 'сегодня'
  if (days === 1) return 'завтра'
  if (days === -1) return 'вчера'
  return dateShortFormat.format(date)
}

/** Today → «18:00», another day → «20.11, 18:00». */
export function formatPlacedAt(iso: string, now: Date = new Date()): string {
  const date = validDate(iso)
  if (!date) return '—'
  return isToday(iso, now) ? timeFormat.format(date) : dateTimeFormat.format(date)
}

export function formatTime(iso: string): string {
  const date = validDate(iso)
  return date ? timeFormat.format(date) : '—'
}

export function formatInteger(value: number): string {
  return integerFormat.format(Number.isFinite(value) ? Math.trunc(value) : 0)
}

/** Russian plural: pluralRu(8, ['матч', 'матча', 'матчей']) → «матчей». */
export function pluralRu(count: number, forms: [one: string, few: string, many: string]): string {
  const mod10 = Math.abs(count) % 10
  const mod100 = Math.abs(count) % 100
  if (mod10 === 1 && mod100 !== 11) return forms[0]
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return forms[1]
  return forms[2]
}

/**
 * Parses a rouble input into integer kopecks without floating point money.
 * Accepts «1 000,50», «1000.5», «500». Rejects more than two decimals.
 */
export function parseStakeToMinor(input: string): number | null {
  const cleaned = input.replace(/[\s\u00A0\u202F]/g, '').replace(',', '.')
  if (!/^\d+(?:\.\d{1,2})?$/.test(cleaned)) return null

  const [whole, fraction = ''] = cleaned.split('.')
  const minor = Number(whole) * 100 + Number(fraction.padEnd(2, '0'))
  return Number.isSafeInteger(minor) ? minor : null
}

/** Integer kopecks to a plain editable rouble string: 100050 → «1 000,50». */
export function formatStakeInput(minor: number): string {
  const safe = Math.max(0, Math.trunc(minor))
  const roubles = Math.floor(safe / 100)
  const kopecks = safe % 100
  const whole = integerFormat.format(roubles)
  return kopecks === 0 ? whole : `${whole},${String(kopecks).padStart(2, '0')}`
}
