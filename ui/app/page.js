"use client"

import { useState } from 'react'
import useSWR from 'swr'
import Link from 'next/link'
import { formatDistanceToNow } from 'date-fns'
import { Plus, Activity, Settings } from 'lucide-react'

const fetcher = (url) => fetch(url).then(r => r.json())

const Sidebar = () => (
  <div className="w-64 bg-slate-900 text-white h-screen fixed left-0 top-0 p-6 flex flex-col">
    <div className="mb-8">
      <h1 className="text-2xl font-bold">Tenant Master</h1>
      <p className="text-slate-400 text-sm">Management Console</p>
    </div>
    <nav className="space-y-2 flex-1">
      <Link href="/" className="flex items-center gap-3 px-4 py-3 rounded-lg bg-slate-800 text-white">
        <Activity size={20} />
        Tenants
      </Link>
      <Link href="/metrics" className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-800 transition">
        <Settings size={20} />
        Metrics
      </Link>
    </nav>
    <div className="border-t border-slate-700 pt-4 text-xs text-slate-400">
      <p>v0.1.0 - Production Grade</p>
      <p>© 2026 Tenant Master</p>
    </div>
  </div>
)

const Header = () => (
  <div className="ml-64 bg-white border-b border-slate-200 p-6 flex justify-between items-center">
    <div>
      <h2 className="text-3xl font-bold text-slate-900">Tenant Management</h2>
      <p className="text-slate-600 mt-1">Manage multi-tenant environments with ease</p>
    </div>
    <Link href="/tenants/new" className="btn-primary flex items-center gap-2">
      <Plus size={20} />
      New Tenant
    </Link>
  </div>
)

const TierBadge = ({ tier }) => (
  <span className={`tier-badge tier-${tier.toLowerCase()}`}>
    {tier}
  </span>
)

const StateBadge = ({ state }) => {
  const icons = {
    'Provisioning': <Activity size={16} className="animate-spin" />,
    'Ready': <div className="w-2 h-2 rounded-full bg-green-600" />,
    'Failed': <div className="w-2 h-2 rounded-full bg-red-600" />,
    'Suspended': <div className="w-2 h-2 rounded-full bg-slate-400" />,
  }
  return (
    <span className={`state-${(state || 'failed').toLowerCase()}`}>
      {icons[state] || null}
      {state || 'Unknown'}
    </span>
  )
}

const TenantCard = ({ tenant }) => (
  <Link href={`/tenants/${tenant.name}`}>
    <div className="card hover:shadow-md transition cursor-pointer border-l-4 border-l-blue-600">
      <div className="flex justify-between items-start mb-4">
        <div>
          <h3 className="text-lg font-bold text-slate-900">{tenant.name}</h3>
          <p className="text-sm text-slate-600">{tenant.owner}</p>
        </div>
        <TierBadge tier={tenant.tier || 'Bronze'} />
      </div>
      <div className="grid grid-cols-2 gap-4 mb-4 text-sm">
        <div>
          <p className="text-slate-600">Resources</p>
          <p className="font-mono text-slate-900">{tenant.cpu || 'N/A'} / {tenant.memory || 'N/A'}</p>
        </div>
        <div>
          <p className="text-slate-600">Namespace</p>
          <p className="font-mono text-slate-900">{tenant.namespace || '-'}</p>
        </div>
      </div>
      <div className="flex justify-between items-center">
        <StateBadge state={tenant.state} />
        {tenant.createdAt && (
          <span className="text-xs text-slate-500">
            Created {formatDistanceToNow(new Date(tenant.createdAt), { addSuffix: true })}
          </span>
        )}
      </div>
    </div>
  </Link>
)

export default function Home() {
  const { data: tenants, error, isLoading } = useSWR('/api/v1/tenants', (url) => fetch(url).then(r => r.json()), {
    refreshInterval: 5000,
  })

  return (
    <div className="flex min-h-screen bg-slate-50">
      <Sidebar />
      <main className="ml-64 flex-1">
        <Header />
        <div className="p-8">
          {error && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
              <p className="text-red-800">Failed to load tenants. Make sure the BFF is running.</p>
            </div>
          )}

          {isLoading && (
            <div className="grid grid-cols-1 gap-6">
              {[1, 2, 3].map((i) => (
                <div key={i} className="card h-40 bg-slate-100 animate-pulse" />
              ))}
            </div>
          )}

          {tenants && tenants.length === 0 && (
            <div className="text-center py-12">
              <Activity size={48} className="mx-auto text-slate-400 mb-4" />
              <h3 className="text-lg font-semibold text-slate-600 mb-2">No tenants yet</h3>
              <p className="text-slate-500 mb-6">Create your first tenant to get started</p>
              <Link href="/tenants/new" className="btn-primary inline-flex items-center gap-2">
                <Plus size={20} />
                Create Tenant
              </Link>
            </div>
          )}

          {tenants && tenants.length > 0 && (
            <div>
              <div className="mb-6 flex justify-between items-center">
                <h3 className="text-xl font-bold text-slate-900">
                  {tenants.length} Tenant{tenants.length !== 1 ? 's' : ''}
                </h3>
                <div className="flex gap-2 text-sm">
                  {['Bronze', 'Silver', 'Gold'].map((tier) => {
                    const count = tenants.filter((t) => t.tier === tier).length
                    return (
                      <span key={tier} className="text-slate-600">
                        {tier}: <span className="font-semibold">{count}</span>
                      </span>
                    )
                  })}
                </div>
              </div>
              <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                {tenants.map((tenant) => (
                  <TenantCard key={tenant.name} tenant={tenant} />
                ))}
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
