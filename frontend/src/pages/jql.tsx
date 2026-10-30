import { useQuery } from '@tanstack/react-query'
import { Copy } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { api } from '@/lib/api'
import { data } from '@/lib/format'

async function copiar(texto: string) {
  try {
    await navigator.clipboard.writeText(texto)
    toast.success('JQL copiada')
  } catch {
    toast.error('Não foi possível copiar a JQL')
  }
}

export default function JqlPage() {
  const { data: painel, isLoading, error } = useQuery({ queryKey: ['jql'], queryFn: api.jql })

  if (isLoading) return <Skeleton className="h-96" />
  if (error || !painel)
    return (
      <Alert variant="destructive">
        <AlertDescription>
          {error instanceof Error ? error.message : 'Erro ao carregar as consultas.'}
        </AlertDescription>
      </Alert>
    )

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Consultas que geram os dados</CardTitle>
          <CardDescription>
            A sincronização usa estas JQL. Período de {data(painel.inicio)} a {data(painel.fim)}.
            Histórias e Test Bugs usam o limite exclusivo {data(painel.limite_exclusivo)}, o dia 1
            do mês seguinte. Cada cartão é um Espaço: copie a consulta e confira no Jira.
          </CardDescription>
        </CardHeader>
      </Card>
      {painel.espacos.length === 0 ? (
        <Alert>
          <AlertDescription>Nenhum Espaço habilitado nas Configurações.</AlertDescription>
        </Alert>
      ) : (
        painel.espacos.map((espaco) => (
          <Card key={espaco.chave}>
            <CardHeader>
              <CardTitle>
                {espaco.chave}{' '}
                <span className="font-normal text-muted-foreground">— {espaco.nome}</span>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {espaco.consultas.map((c) => (
                <div key={c.id} className="space-y-2">
                  <div className="flex items-center justify-between gap-2">
                    <h3 className="text-sm font-medium">{c.titulo}</h3>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => void copiar(c.jql)}
                    >
                      <Copy /> Copiar
                    </Button>
                  </div>
                  <pre className="overflow-x-auto rounded-md border bg-muted p-3 text-xs whitespace-pre-wrap">
                    {c.jql}
                  </pre>
                </div>
              ))}
            </CardContent>
          </Card>
        ))
      )}
    </div>
  )
}
