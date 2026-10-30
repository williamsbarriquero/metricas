import { describe, expect, it } from 'vitest'

import {
  data,
  duracao,
  fimDoMes,
  horas,
  inicioDoMes,
  numero,
  paraMes,
  percentual,
  percentualDireto,
  VAZIO,
} from './format'

describe('formatadores do painel', () => {
  it('mostra travessão quando o número não existe', () => {
    expect(numero(null)).toBe(VAZIO)
    expect(numero(undefined)).toBe(VAZIO)
    expect(percentual(null)).toBe(VAZIO)
    expect(percentualDireto(undefined)).toBe(VAZIO)
    expect(horas(null)).toBe(VAZIO)
  })

  it('formata inteiro, razão e percentual já em pontos', () => {
    expect(numero(1000)).toBe('1.000')
    expect(percentual(0.476)).toBe('47,6%')
    expect(percentualDireto(102.82)).toBe('102,82%')
  })

  it('converte horas decimais no relógio dos marcadores de slide', () => {
    expect(horas(1.5)).toBe('1:30:00')
    expect(horas(0)).toBe('0:00:00')
  })

  it('recorta o mês e devolve o primeiro e o último dia', () => {
    expect(paraMes('2026-07-01')).toBe('2026-07')
    expect(inicioDoMes('2026-07')).toBe('2026-07-01')
    expect(fimDoMes('2026-09')).toBe('2026-09-30')
    expect(fimDoMes('2024-02')).toBe('2024-02-29')
    expect(data('2026-07-01')).toBe('01/07/2026')
  })

  it('mede a duração entre dois instantes', () => {
    expect(duracao('2026-01-01T00:00:00Z', '2026-01-01T00:01:30Z')).toBe('1min 30s')
    expect(duracao('2026-01-01T00:00:00Z', '2026-01-01T00:00:12Z')).toBe('12s')
  })
})
