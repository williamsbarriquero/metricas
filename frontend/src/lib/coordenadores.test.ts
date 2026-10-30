import { expect, it } from 'vitest'

import { secoesDe } from './coordenadores'

it('secoes seguem o arquivo, extras em pt-BR e sem coordenador no fim', () => {
  const itens = [
    { nome: 'Sem A', coord: '' },
    { nome: 'Zulu', coord: 'Zulu Extra' },
    { nome: 'Depois', coord: 'Williams' },
    { nome: 'Alfa', coord: 'Álvaro Extra' },
    { nome: 'Antes', coord: 'Williams' },
    { nome: 'Primeira', coord: 'Thalita' },
    { nome: 'Sem B', coord: '' },
  ]
  const secoes = secoesDe(itens, ['Thalita', 'Williams', 'Fernando'], (item) => item.coord)
  expect(secoes.map((secao) => secao.rotulo)).toEqual([
    'Thalita',
    'Williams',
    'Álvaro Extra',
    'Zulu Extra',
    'Sem coordenador',
  ])
  expect(secoes[1].itens.map((item) => item.nome)).toEqual(['Depois', 'Antes'])
  expect(secoes[4].itens.map((item) => item.nome)).toEqual(['Sem A', 'Sem B'])
})
