import Link from 'next/link'
import { BarChart3 } from 'lucide-react'

export default function Metrics() {
  return (
    <div className="min-h-screen bg-slate-50">
      <div className="max-w-6xl mx-auto p-8">
        <Link href="/" className="text-blue-600 hover:text-blue-700 mb-6">
          ← Back to Tenants
        </Link>
        <div className="card text-center py-12">
          <BarChart3 size={48} className="mx-auto text-slate-400 mb-4" />
          <h2 className="text-2xl font-bold text-slate-900 mb-2">Cluster Metrics</h2>
          <p className="text-slate-600">Prometheus integration coming soon</p>
        </div>
      </div>
    </div>
  )
}
