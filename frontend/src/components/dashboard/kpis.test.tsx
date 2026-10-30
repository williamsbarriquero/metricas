import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { Bloco, LinhaT1, LinhaT2, LinhaT3, Tabela } from '@/lib/api'

import { Kpis } from './kpis'

function tabela<L>(total: L): Tabela<L> {
  return { meses: [], total }
}

function bloco(t1: LinhaT1, t2: LinhaT2, t3: LinhaT3): Bloco {
  return {
    chave: 'HPAY',
    nome: 'HPAY',
    rotulo: 'HPAY',
    tem_dados: true,
    t1: tabela(t1),
    t2: { ...tabela(t2), overhead_detalhe: [] },
    t3: tabela(t3),
  }
}

describe('Kpis', () => {
  it('mostra razão, horas e vazão do bloco', () => {
    const dados = bloco(
      {
        historias: 4,
        test_bugs: 2,
        razao: 0.5,
        fluxo_principal: 2,
        fluxo_alternativo: 0,
        fluxo_excecao: 0,
      },
      {
        sustentacao: 10,
        evolucao: 1.5,
        overhead: 0,
        sem_conta: 0,
        recesso: 0,
        total: 11.5,
        pct_sustentacao: 10 / 11.5,
        pct_evolucao: 1.5 / 11.5,
        pct_overhead: 0,
      },
      { criados: 20, resolvidos: 10, estoque: 3, vazao: 50 },
    )

    render(<Kpis bloco={dados} />)

    expect(screen.getByText('50,0%')).toBeTruthy()
    expect(screen.getByText('2 test bugs em 4 histórias')).toBeTruthy()
    expect(screen.getByText('1:30:00 de 11:30:00')).toBeTruthy()
    expect(screen.getByText('50,00%')).toBeTruthy()
    expect(screen.getByText('10 resolvidos de 20 criados · estoque 3')).toBeTruthy()
  })
})
