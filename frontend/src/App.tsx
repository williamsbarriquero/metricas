import { Navigate, Route, Routes } from 'react-router-dom'

import { Layout } from '@/components/layout'
import ConfiguracoesPage from '@/pages/configuracoes'
import DadosPage from '@/pages/dados'
import DashboardPage from '@/pages/dashboard'
import EquipePage from '@/pages/equipe'
import GlossarioPage from '@/pages/glossario'
import HorasEquipePage from '@/pages/horas-equipe'
import JqlPage from '@/pages/jql'
import MapaSlidePage from '@/pages/mapa-slide'
import MetricasPage from '@/pages/metricas'
import PrintCartaoPage from '@/pages/print-cartao'
import SincronizacaoPage from '@/pages/sincronizacao'

export default function App() {
  return (
    <Routes>
      <Route path="print" element={<PrintCartaoPage />} />
      <Route element={<Layout />}>
        <Route index element={<DashboardPage />} />
        <Route path="metricas" element={<MetricasPage />} />
        <Route path="mapa-slide" element={<MapaSlidePage />} />
        <Route path="dados" element={<Navigate to="/dados/historias" replace />} />
        <Route path="dados/:aba" element={<DadosPage />} />
        <Route path="equipe" element={<EquipePage />} />
        <Route path="horas-equipe" element={<HorasEquipePage />} />
        <Route path="configuracoes" element={<ConfiguracoesPage />} />
        <Route path="sincronizacao" element={<SincronizacaoPage />} />
        <Route path="glossario" element={<GlossarioPage />} />
        <Route path="jql" element={<JqlPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}
