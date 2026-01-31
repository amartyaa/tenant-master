'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, Loader } from 'lucide-react'

export default function NewTenant() {
  const router = useRouter()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [formData, setFormData] = useState({
    name: '',
    tier: 'Silver',
    owner: '',
    cpu: '4000m',
    memory: '8Gi',
    allowInternetAccess: false,
  })

  const handleInputChange = (e) => {
    const { name, value, type, checked } = e.target
    setFormData({
      ...formData,
      [name]: type === 'checkbox' ? checked : value,
    })
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      const payload = {
        name: formData.name,
        tier: formData.tier,
        owner: formData.owner,
        resources: {
          cpu: formData.cpu,
          memory: formData.memory,
        },
        network: {
          allowInternetAccess: formData.allowInternetAccess,
          whitelistedServices: [],
        },
      }

      const res = await fetch('/api/v1/tenants', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })

      if (!res.ok) {
        throw new Error(`Failed to create tenant: ${res.statusText}`)
      }

      router.push('/')
    } catch (err) {
      setError(err.message)
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="max-w-2xl mx-auto p-8">
        <Link href="/" className="flex items-center gap-2 text-blue-600 hover:text-blue-700 mb-6">
          <ArrowLeft size={20} />
          Back to Tenants
        </Link>

        <div className="card">
          <h1 className="text-3xl font-bold text-slate-900 mb-6">Create New Tenant</h1>

          {error && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
              <p className="text-red-800">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Tenant Name */}
            <div>
              <label className="block text-sm font-semibold text-slate-900 mb-2">
                Tenant Name *
              </label>
              <input
                type="text"
                name="name"
                value={formData.name}
                onChange={handleInputChange}
                placeholder="e.g., acme-payments"
                required
                className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <p className="text-xs text-slate-500 mt-1">Lowercase alphanumeric and hyphens only</p>
            </div>

            {/* Tier Selection */}
            <div>
              <label className="block text-sm font-semibold text-slate-900 mb-3">
                Isolation Tier *
              </label>
              <div className="grid grid-cols-3 gap-4">
                {[
                  { value: 'Bronze', desc: 'Soft isolation, shared resources' },
                  { value: 'Silver', desc: 'Namespace isolation, policy enforcement' },
                  { value: 'Gold', desc: 'vCluster, complete isolation' },
                ].map((tier) => (
                  <label
                    key={tier.value}
                    className={`border-2 rounded-lg p-4 cursor-pointer transition ${
                      formData.tier === tier.value
                        ? 'border-blue-600 bg-blue-50'
                        : 'border-slate-200 hover:border-slate-300'
                    }`}
                  >
                    <input
                      type="radio"
                      name="tier"
                      value={tier.value}
                      checked={formData.tier === tier.value}
                      onChange={handleInputChange}
                      className="mr-2"
                    />
                    <div>
                      <p className="font-semibold text-slate-900">{tier.value}</p>
                      <p className="text-xs text-slate-600">{tier.desc}</p>
                    </div>
                  </label>
                ))}
              </div>
            </div>

            {/* Owner Email */}
            <div>
              <label className="block text-sm font-semibold text-slate-900 mb-2">
                Owner Email *
              </label>
              <input
                type="email"
                name="owner"
                value={formData.owner}
                onChange={handleInputChange}
                placeholder="owner@company.com"
                required
                className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            {/* Resources */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-semibold text-slate-900 mb-2">
                  CPU Request
                </label>
                <input
                  type="text"
                  name="cpu"
                  value={formData.cpu}
                  onChange={handleInputChange}
                  placeholder="4000m"
                  className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-semibold text-slate-900 mb-2">
                  Memory Request
                </label>
                <input
                  type="text"
                  name="memory"
                  value={formData.memory}
                  onChange={handleInputChange}
                  placeholder="8Gi"
                  className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>

            {/* Network */}
            <div>
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  name="allowInternetAccess"
                  checked={formData.allowInternetAccess}
                  onChange={handleInputChange}
                  className="w-4 h-4 rounded"
                />
                <span className="text-sm font-medium text-slate-900">Allow Internet Access</span>
              </label>
              <p className="text-xs text-slate-500 mt-1">Enable egress to external networks</p>
            </div>

            {/* Submit */}
            <div className="flex gap-4 pt-6 border-t border-slate-200">
              <button
                type="submit"
                disabled={loading}
                className="btn-primary flex items-center gap-2 flex-1 justify-center"
              >
                {loading && <Loader size={20} className="animate-spin" />}
                {loading ? 'Creating...' : 'Create Tenant'}
              </button>
              <Link href="/" className="btn-secondary flex-1 text-center">
                Cancel
              </Link>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
