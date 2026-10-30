import { useQuery } from '@tanstack/react-query'
import { Check, Minus } from 'lucide-react'
import { Link } from 'react-router-dom'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { api } from '@/lib/api'

function Sim({ valor }: { valor: boolean }) {
  return valor ? (
    <Check className="mx-auto size-4 text-emerald-600" aria-label="Sim" />
  ) : (
    <Minus className="mx-auto size-4 text-muted-foreground/50" aria-label="Não" />
  )
}

export default function GlossarioPage() {
  const { data, isLoading, error } = useQuery({ queryKey: ['glossario'], queryFn: api.glossario })

  if (isLoading) return <Skeleton className="h-96" />
  if (error || !data)
    return (
      <Alert variant="destructive">
        <AlertDescription>
          {(error as Error | null)?.message ?? 'Erro ao carregar o glossário.'}
        </AlertDescription>
      </Alert>
    )

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Espaços</CardTitle>
          <CardDescription>
            Chave do projeto no Jira, nome do squad e em quais coletas cada Espaço está habilitado
            (veja Configurações). As consultas usadas para gerar os dados estão no{' '}
            <Link to="/jql" className="font-medium underline">
              Glossário de JQL
            </Link>
            .
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-24">Chave</TableHead>
                <TableHead className="w-36">BU</TableHead>
                <TableHead>Squad</TableHead>
                <TableHead className="text-center">Histórias e Test Bugs</TableHead>
                <TableHead className="text-center">Incidentes</TableHead>
                <TableHead className="text-center">Horas</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.espacos.map((e) => (
                <TableRow key={e.chave}>
                  <TableCell className="font-mono font-medium">{e.chave}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-1.5">
                      {e.cor && (
                        <span
                          className="inline-block size-2.5 rounded-full ring-1 ring-foreground/15"
                          style={{ backgroundColor: e.cor }}
                          aria-hidden
                        />
                      )}
                      {e.bu_nome || '—'}
                    </span>
                  </TableCell>
                  <TableCell>{e.nome}</TableCell>
                  <TableCell>
                    <Sim valor={e.historias} />
                  </TableCell>
                  <TableCell>
                    <Sim valor={e.incidentes} />
                  </TableCell>
                  <TableCell>
                    <Sim valor={e.horas} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {data.tiers.map((t) => (
        <Card key={t.id}>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <span className="rounded-md bg-primary/10 px-1.5 py-0.5 text-xs font-semibold text-primary">
                {t.id}
              </span>
              {t.titulo}
            </CardTitle>
            <CardDescription>{t.descricao}</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-20">Código</TableHead>
                  <TableHead>Indicador</TableHead>
                  <TableHead>Formato</TableHead>
                  <TableHead>Regra</TableHead>
                  <TableHead>Exemplo</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {t.campos.map((c) => (
                  <TableRow key={c.cod}>
                    <TableCell className="font-mono font-medium">{c.cod}</TableCell>
                    <TableCell>{c.nome}</TableCell>
                    <TableCell className="text-muted-foreground">{c.formato}</TableCell>
                    <TableCell className="text-muted-foreground">{c.regra}</TableCell>
                    <TableCell className="font-mono text-xs">{c.exemplo}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      ))}

      {data.gerais && data.gerais.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Marcadores gerais do Slide</CardTitle>
            <CardDescription>
              Cole no modelo. Os valores de cada Espaço estão na tela{' '}
              <Link to="/mapa-slide" className="font-medium underline">
                MapaSlide
              </Link>
              .
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Marcador</TableHead>
                  <TableHead>O que traz</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.gerais.map((g) => (
                  <TableRow key={g.chave}>
                    <TableCell className="font-mono text-xs">{g.chave}</TableCell>
                    <TableCell className="text-muted-foreground">{g.descricao}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
