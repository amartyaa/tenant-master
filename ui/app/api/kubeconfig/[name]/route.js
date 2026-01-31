import { NextResponse } from 'next/server'

export async function GET(request, { params }) {
  const BFF = process.env.BFF_URL || 'http://localhost:8080'
  const { name } = params

  try {
    const resp = await fetch(`${BFF}/api/v1/tenants/${name}/kubeconfig`, {
      headers: {
        'Authorization': request.headers.get('Authorization') || '',
      },
    })

    if (!resp.ok) {
      throw new Error(`BFF returned ${resp.status}`)
    }

    const kubeconfig = await resp.text()
    return new NextResponse(kubeconfig, {
      status: 200,
      headers: { 'Content-Type': 'text/plain' },
    })
  } catch (err) {
    return NextResponse.json(
      { error: `Failed to get kubeconfig: ${err.message}` },
      { status: 503 }
    )
  }
}
