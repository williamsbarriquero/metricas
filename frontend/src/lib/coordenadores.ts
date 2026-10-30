export const SEM_COORDENADOR = '__sem__'

export function opcoesCoordenador(canon: string[], coordenadores: string[]) {
  const vistos = new Set(canon)
  const extras = new Set<string>()
  for (const coordenador of coordenadores) {
    if (coordenador && !vistos.has(coordenador)) extras.add(coordenador)
  }
  const extrasOrd = [...extras].sort((a, b) => a.localeCompare(b, 'pt-BR'))
  return [
    { value: 'todos', label: 'Todos' },
    ...canon.map((c) => ({ value: c, label: c })),
    ...extrasOrd.map((c) => ({ value: c, label: c })),
    { value: SEM_COORDENADOR, label: 'Sem coordenador' },
  ]
}

export function secoesDe<T>(itens: T[], canon: string[], coordenadorDe: (item: T) => string) {
  const ordem = opcoesCoordenador(
    canon,
    itens.map((item) => coordenadorDe(item)),
  )
    .map((o) => o.value)
    .filter((v) => v !== 'todos')
  const por = new Map<string, T[]>()
  for (const item of itens) {
    const chave = coordenadorDe(item) || SEM_COORDENADOR
    const lista = por.get(chave) ?? []
    lista.push(item)
    por.set(chave, lista)
  }
  return ordem
    .filter((chave) => (por.get(chave)?.length ?? 0) > 0)
    .map((chave) => ({
      chave,
      rotulo: chave === SEM_COORDENADOR ? 'Sem coordenador' : chave,
      itens: por.get(chave) ?? [],
    }))
}
