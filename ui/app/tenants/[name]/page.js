'use client'

import { useState } from 'react'
import useSWR from 'swr'
import Link from 'next/link'
import { formatDistanceToNow } from 'date-fns'
import { ArrowLeft, Copy, Download, Loader, AlertCircle, CheckCircle } from 'lucide-react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

const fetcher = (url) => fetch(url).then(r => r.json())

const mockMetricsHistory = [
  { time: '00:00', cpu: 100, memory: 256 },
  { time: '04:00', cpu: 150, memory: 384 },
  { time: '08:00', cpu: 350, memory: 512 },
  { time: '12:00', cpu: 280, memory: 450 },
  { time: '16:00', cpu: 400, memory: 600 },
  { time: '20:00', cpu: 320, memory: 520 },
  { time: '24:00', cpu: 250, memory: 400 },
]

export default function TenantDetail({ params }) {
  const { name } = params
  const [copied, setCopied] = useState(false)
  const { data: tenant, error, isLoading } = useSWR(`/api/v1/tenants/${name}`, fetcher)
  const { data: metrics } = useSWR(`/api/v1/tenants/${name}/metrics`, fetcher)

  const copyToClipboard = async () => {
    try {
      const res = await fetch(`/api/v1/tenants/${name}/kubeconfig`)
      if (!res.ok) throw new Error('Failed to fetch kubeconfig')
      const kubeconfig = await res.text()
      await navigator.clipboard.writeText(kubeconfig)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      alert('Failed to copy kubeconfig')
    }
  }

  if (error) {
    return (
      <div className="min-h-screen bg-slate-50 p-8">
        <div className="max-w-4xl mx-auto">
          <Link href="/" className="flex items-center gap-2 text-blue-600 hover:text-blue-700 mb-6">
            <ArrowLeft size={20} />
            Back to Tenants
          </Link>
          <div className="bg-red-50 border border-red-200 rounded-lg p-6">
            <p className="text-red-800">Tenant not found or failed to load.</p>
          </div>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center">
        <Loader size={40} className="animate-spin text-blue-600" />
      </div>
    )
  }

  const stateColor = {
    Ready: 'text-green-600',
    Provisioning: 'text-yellow-600',
    Failed: 'text-red-600',
    Suspended: 'text-slate-600',
  }

  const tierColor = {
    Bronze: 'bg-amber-50 border-amber-200',
    Silver: 'bg-slate-50 border-slate-200',
    Gold: 'bg-yellow-50 border-yellow-200',
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="max-w-6xl mx-auto p-8">
        <Link href="/" className="flex items-center gap-2 text-blue-600 hover:text-blue-700 mb-6">
          <ArrowLeft size={20} />
          Back to Tenants
        </Link>

        {/* Header */}
        <div className={`card border-l-4 mb-8 ${tierColor[tenant.tier] || tierColor.Bronze}`}>
          <div className="flex justify-between items-start">
            <div>
              <h1 className="text-4xl font-bold text-slate-900 mb-2">{tenant.name}</h1>
              <p className="text-slate-600">Owner: {tenant.owner}</p>
              {tenant.createdAt && (
                <p className="text-sm text-slate-500 mt-1">
                  Created {formatDistanceToNow(new Date(tenant.createdAt), { addSuffix: true })}
                </p>
              )}
            </div>
            <div className="text-right">
              <div className={`inline-flex items-center gap-2 px-4 py-2 rounded-lg mb-2 ${
                tenant.tier === 'Bronze' ? 'bg-amber-100 text-amber-800' :
                tenant.tier === 'Silver' ? 'bg-slate-100 text-slate-800' :
                'bg-yellow-100 text-yellow-800'
              }`}>
                {tenant.tier} Tier
              </div>
              <div className={`inline-flex items-center gap-2 px-4 py-2 rounded-lg ml-2 ${
                stateColor[tenant.state] === 'text-green-600' ? 'bg-green-100 text-green-800' :
                stateColor[tenant.state] === 'text-yellow-600' ? 'bg-yellow-100 text-yellow-800' :
                stateColor[tenant.state] === 'text-red-600' ? 'bg-red-100 text-red-800' :
                'bg-slate-100 text-slate-800'
              }`}>
                <CheckCircle size={16} />
                {tenant.state || 'Unknown'}
              </div>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-3 gap-6 mb-8">
          {/* Details Card */}
          <div className="card">
            <h3 className="text-lg font-bold text-slate-900 mb-4">Details</h3>
            <div className="space-y-3">
              <div>
                <p className="text-xs text-slate-600">Namespace</p>
                <p className="font-mono text-slate-900">{tenant.namespace || '-'}</p>
              </div>
              <div>
                <p className="text-xs text-slate-600">CPU</p>
                <p className="font-mono text-slate-900">{tenant.cpu || '-'}</p>
              </div>
              <div>
                <p className="text-xs text-slate-600">Memory</p>
                <p className="font-mono text-slate-900">{tenant.memory || '-'}</p>
              </div>
            </div>
          </div>

          {/* Metrics Card */}
          <div className="card">
            <h3 className="text-lg font-bold text-slate-900 mb-4">Current Metrics</h3>
            {metrics ? (
              <div className="space-y-3">
                <div>
                  <p className="text-xs text-slate-600">CPU Usage</p>
                  <p className="font-mono text-slate-900">{metrics.metrics.cpu_usage}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-600">Memory Usage</p>
                  <p className="font-mono text-slate-900">{metrics.metrics.memory_usage}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-600">Status</p>
                  <p className="font-mono text-green-600">{metrics.metrics.active ? 'Active' : 'Inactive'}</p>
                </div>
              </div>
            ) : (
              <p className="text-slate-600">Loading...</p>
            )}
          </div>

          {/* Actions Card */}
          <div className="card">
            <h3 className="text-lg font-bold text-slate-900 mb-4">Actions</h3>
            {tenant.tier === 'Gold' && tenant.kubeconfigSecret && (
              <button
                onClick={copyToClipboard}
                className="w-full btn-primary flex items-center justify-center gap-2 mb-2"
              >
                {copied ? (
                  <>
                    <CheckCircle size={16} />
                    Copied!
                  </>
                ) : (
                  <>
                    <Copy size={16} />
                    Copy Kubeconfig
                  </>
                )}
              </button>
            )}
            <button className="w-full btn-secondary mb-2">
              <Download size={16} className="inline mr-2" />
              Export
            </button>
            <button className="w-full btn-danger">
              <AlertCircle size={16} className="inline mr-2" />
              Delete
            </button>
          </div>
        </div>

        {/* Metrics Chart */}
        <div className="card mb-8">
          <h3 className="text-lg font-bold text-slate-900 mb-6">Resource Usage (24h)</h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={mockMetricsHistory}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <YAxis yAxisId="left" />
              <YAxis yAxisId="right" orientation="right" />
              <Tooltip />
              <Legend />
              <Line yAxisId="left" type="monotone" dataKey="cpu" stroke="#3b82f6" name="CPU (m)" />
              <Line yAxisId="right" type="monotone" dataKey="memory" stroke="#10b981" name="Memory (Mi)" />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Network Policy Info */}
        {tenant.tier !== 'Bronze' && (
          <div className="card">
            <h3 className="text-lg font-bold text-slate-900 mb-4">Network Policy</h3>
            <div className="bg-slate-50 rounded p-4 text-sm text-slate-600">
              <p>Default Deny ingress and egress policies are enforced.</p>
              <p className="mt-2">Whitelisted services can be configured in the tenant specification.</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
