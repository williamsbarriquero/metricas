import { useQuery } from '@tanstack/react-query'
import { ChevronRight } from 'lucide-react'
import { useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { api, type MembroEquipe, type PessoaHoras } from '@/lib/api'
import { secoesDe } from '@/lib/coordenadores'
import { data, horas } from '@/lib/format'

function posicaoNaEquipe(pessoa: PessoaHoras, membros: MembroEquipe[]) {
  const porConta = membros.findIndex(
    (m) => m.account_id !== '' && m.account_id === pessoa.account_id,
  )
  if (porConta >= 0) return porConta
  const porNome = membros.findIndex((m) => m.nome === pessoa.nome)
  return porNome >= 0 ? porNome : membros.length
}

function coordenadorDaPessoa(pessoa: PessoaHoras, membros: MembroEquipe[]) {
  if (pessoa.coordenador) return pessoa.coordenador
  const i = posicaoNaEquipe(pessoa, membros)
  return i < membros.length ? membros[i].coordenador : ''
}

function pessoasNaOrdemDaEquipe(pessoas: PessoaHoras[], membros: MembroEquipe[]) {
  return [...pessoas].sort(
    (a, b) =>
      posicaoNaEquipe(a, membros) - posicaoNaEquipe(b, membros) ||
      a.nome.localeCompare(b.nome, 'pt-BR'),
  )
}

function CartaoPessoa({ pessoa }: { pessoa: PessoaHoras }) {
  return (
    <details className="group rounded-xl bg-card text-sm text-card-foreground ring-1 ring-foreground/10">
      <summary className="flex cursor-pointer list-none items-center gap-3 px-4 py-4 [&::-webkit-details-marker]:hidden">
        <ChevronRight className="mt-0.5 size-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-90" />
        <div className="min-w-0">
          <div className="font-heading text-base leading-snug font-medium">{pessoa.nome}</div>
          <p className="text-muted-foreground">
            {pessoa.account_id ? `${pessoa.account_id} · ` : ''}
            Total {horas(pessoa.total_horas)}
          </p>
        </div>
      </summary>
      <div className="space-y-4 px-4 pb-4">
        <div>
          <h3 className="mb-2 text-sm font-medium">Por mês</h3>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Mês</TableHead>
                <TableHead className="text-right">Horas</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pessoa.meses.map((m) => (
                <TableRow key={m.rotulo}>
                  <TableCell>{m.rotulo}</TableCell>
                  <TableCell className="text-right tabular-nums">{horas(m.horas)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
        <div>
          <h3 className="mb-2 text-sm font-medium">Por conta de esforço</h3>
          {pessoa.contas.length === 0 ? (
            <p className="text-muted-foreground">Sem lançamentos no período.</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Conta</TableHead>
                  <TableHead className="text-right">Horas</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pessoa.contas.map((c) => (
                  <TableRow key={c.rotulo}>
                    <TableCell>{c.rotulo}</TableCell>
                    <TableCell className="text-right tabular-nums">{horas(c.horas)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
        <div>
          <h3 className="mb-2 text-sm font-medium">Lançamentos</h3>
          {pessoa.lancamentos.length === 0 ? (
            <p className="text-muted-foreground">Nenhum lançamento no período.</p>
          ) : (
            <div className="max-h-96 overflow-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Data</TableHead>
                    <TableHead>Espaço</TableHead>
                    <TableHead>Issue</TableHead>
                    <TableHead>Resumo</TableHead>
                    <TableHead>Conta</TableHead>
                    <TableHead className="text-right">Horas</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pessoa.lancamentos.map((l, i) => (
                    <TableRow key={`${l.chave}-${l.data}-${i}`}>
                      <TableCell>{l.data}</TableCell>
                      <TableCell>{l.espaco}</TableCell>
                      <TableCell>
                        {l.link ? (
                          <a
                            href={l.link}
                            target="_blank"
                            rel="noopener"
                            className="font-medium underline"
                          >
                            {l.chave}
                          </a>
                        ) : (
                          l.chave
                        )}
                      </TableCell>
                      <TableCell className="max-w-md whitespace-normal">{l.resumo}</TableCell>
                      <TableCell>{l.conta}</TableCell>
                      <TableCell className="text-right tabular-nums">{horas(l.horas)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      </div>
    </details>
  )
}

export default function HorasEquipePage() {
  const [horasFora, setHorasFora] = useState(false)
  const {
    data: analitico,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['horas-equipe', horasFora],
    queryFn: () => api.horasEquipe(horasFora),
  })
  const { data: equipe, isLoading: carregandoEquipe } = useQuery({
    queryKey: ['equipe'],
    queryFn: api.equipe,
  })
  const { data: canon, isLoading: carregandoCanon } = useQuery({
    queryKey: ['equipe-coordenadores'],
    queryFn: api.coordenadoresEquipe,
  })

  if (isLoading || carregandoEquipe || carregandoCanon) return <Skeleton className="h-96" />
  if (error || !analitico)
    return (
      <Alert variant="destructive">
        <AlertDescription>
          {error instanceof Error ? error.message : 'Erro ao carregar as horas.'}
        </AlertDescription>
      </Alert>
    )

  const membros = equipe ?? []
  const grupos = secoesDe(
    pessoasNaOrdemDaEquipe(analitico.pessoas, membros),
    canon ?? [],
    (pessoa) => coordenadorDaPessoa(pessoa, membros),
  )

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>
            Período {data(analitico.inicio)} a {data(analitico.fim)}
          </CardTitle>
          <CardDescription>
            Cada pessoa da Equipe com Incluir ligado, inclusive sem lançamento. Clique no nome para
            abrir os lançamentos. Os grupos seguem a ordem de coordenadores da Equipe.
          </CardDescription>
          <div className="flex items-center gap-2 pt-2 text-sm">
            <Switch
              id="horas-fora-equipe"
              aria-labelledby="horas-fora-equipe-rotulo"
              checked={horasFora}
              onCheckedChange={setHorasFora}
            />
            <span id="horas-fora-equipe-rotulo">Ver horas fora dos espaços mapeados</span>
          </div>
        </CardHeader>
      </Card>
      {analitico.pessoas.length === 0 ? (
        <Alert>
          <AlertDescription>
            Nenhuma pessoa com Incluir ligado. Cadastre a Equipe e sincronize as horas.
          </AlertDescription>
        </Alert>
      ) : (
        grupos.map((grupo) => (
          <section key={grupo.chave} className="space-y-3">
            <h2 className="rounded-md bg-muted/50 px-3 py-2 text-sm font-semibold">
              {grupo.rotulo}
            </h2>
            {grupo.itens.map((pessoa) => (
              <CartaoPessoa key={`${pessoa.account_id}-${pessoa.nome}`} pessoa={pessoa} />
            ))}
          </section>
        ))
      )}
    </div>
  )
}
