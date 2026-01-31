export async function GET(request) {
  const BFF = process.env.BFF_URL || 'http://localhost:8080'
  const url = new URL(request.url)
  const path = url.pathname.replace('/api/v1', '')

  try {
    const resp = await fetch(`${BFF}/api/v1${path}`, {
      method: request.method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': request.headers.get('Authorization') || '',
      },
    })

    const data = await resp.json()
    return Response.json(data, { status: resp.status })
  } catch (err) {
    return Response.json(
      { error: `Failed to reach BFF: ${err.message}` },
      { status: 503 }
    )
  }
}

export async function POST(request) {
  const BFF = process.env.BFF_URL || 'http://localhost:8080'
  const url = new URL(request.url)
  const path = url.pathname.replace('/api/v1', '')
  const body = await request.json()

  try {
    const resp = await fetch(`${BFF}/api/v1${path}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': request.headers.get('Authorization') || '',
      },
      body: JSON.stringify(body),
    })

    const data = await resp.json()
    return Response.json(data, { status: resp.status })
  } catch (err) {
    return Response.json(
      { error: `Failed to reach BFF: ${err.message}` },
      { status: 503 }
    )
  }
}

export async function PATCH(request) {
  const BFF = process.env.BFF_URL || 'http://localhost:8080'
  const url = new URL(request.url)
  const path = url.pathname.replace('/api/v1', '')
  const body = await request.json()

  try {
    const resp = await fetch(`${BFF}/api/v1${path}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': request.headers.get('Authorization') || '',
      },
      body: JSON.stringify(body),
    })

    const data = await resp.json()
    return Response.json(data, { status: resp.status })
  } catch (err) {
    return Response.json(
      { error: `Failed to reach BFF: ${err.message}` },
      { status: 503 }
    )
  }
}

export async function DELETE(request) {
  const BFF = process.env.BFF_URL || 'http://localhost:8080'
  const url = new URL(request.url)
  const path = url.pathname.replace('/api/v1', '')

  try {
    const resp = await fetch(`${BFF}/api/v1${path}`, {
      method: 'DELETE',
      headers: {
        'Authorization': request.headers.get('Authorization') || '',
      },
    })

    const data = await resp.json()
    return Response.json(data, { status: resp.status })
  } catch (err) {
    return Response.json(
      { error: `Failed to reach BFF: ${err.message}` },
      { status: 503 }
    )
  }
}
