import { describe, expect, it } from 'vitest'
import {
  formatKickoffDay,
  formatMatchLabel,
  formatMoney,
  formatOdds,
  formatStakeInput,
  parseStakeToMinor,
  pluralRu,
} from './format'

const plain = (value: string): string => value.replace(/[\u00A0\u202F\u2009]/g, ' ')

describe('money and odds formatting', () => {
  it('formats integer kopecks as ru-RU roubles', () => {
    expect(plain(formatMoney(2_500_000))).toBe('25 000,00 ₽')
    expect(plain(formatMoney(95_500))).toBe('955,00 ₽')
    expect(plain(formatMoney(0))).toBe('0,00 ₽')
  })

  it('formats price thousandths as decimal odds with a comma', () => {
    expect(formatOdds(1950)).toBe('1,95')
    expect(formatOdds(1050)).toBe('1,05')
    expect(formatOdds(20000)).toBe('20,00')
  })

  it('builds the match label with the same separator as the server', () => {
    expect(formatMatchLabel({ homeTeam: 'Арсенал', awayTeam: 'Челси' })).toBe('Арсенал — Челси')
  })

  it('parses rouble input into exact kopecks without float money', () => {
    expect(parseStakeToMinor('1 000,50')).toBe(100_050)
    expect(parseStakeToMinor('1000.5')).toBe(100_050)
    expect(parseStakeToMinor('500')).toBe(50_000)
    expect(parseStakeToMinor('10,005')).toBeNull()
    expect(parseStakeToMinor('1,2,3')).toBeNull()
    expect(plain(formatStakeInput(100_050))).toBe('1 000,50')
    expect(formatStakeInput(50_000)).toBe('500')
  })

  it('formats Russian plurals for the masthead', () => {
    expect(pluralRu(8, ['матч', 'матча', 'матчей'])).toBe('матчей')
    expect(pluralRu(2, ['матч', 'матча', 'матчей'])).toBe('матча')
    expect(pluralRu(21, ['матч', 'матча', 'матчей'])).toBe('матч')
  })

  it('labels kickoff days relative to today', () => {
    const now = new Date(2026, 10, 20, 12, 0)
    expect(formatKickoffDay(new Date(2026, 10, 20, 19, 30).toISOString(), now)).toBe('сегодня')
    expect(formatKickoffDay(new Date(2026, 10, 21, 19, 30).toISOString(), now)).toBe('завтра')
    expect(formatKickoffDay(new Date(2026, 10, 19, 19, 30).toISOString(), now)).toBe('вчера')
    expect(formatKickoffDay(new Date(2026, 10, 23, 19, 30).toISOString(), now)).toBe('23 нояб.')
  })
})
