# Tenant-Master Management Console

A production-grade Next.js UI for managing multi-tenant Kubernetes environments. Inspired by OpenShift and GitHub dashboards.

## Features

- **Tenant Dashboard**: Real-time list of all tenants with tier and status indicators
- **Tenant Creation**: Wizard form for creating new tenants with validation
- **Tenant Details**: Comprehensive detail page with metrics charts and resource allocation
- **Metrics**: CPU/Memory usage graphs (powered by Recharts)
- **Gold-tier Support**: Kubeconfig export and copy-to-clipboard for vCluster access
- **Responsive Design**: Tailwind CSS with mobile-first approach
- **Modern UI**: Icon-based navigation (Lucide React), smooth transitions, dark sidebar

## Quick Start

### Prerequisites

- Node.js 18+
- BFF running on `http://localhost:8080` (or set `BFF_URL`)

### Development

```bash
cd ui
npm install
npm run dev
# Open http://localhost:3000
```

### Production Build

```bash
npm run build
npm start
```

## Environment Variables

```bash
BFF_URL=http://localhost:8080  # Backend for Frontend service URL
```

## Architecture

- **Pages**: Dashboard (home), tenant list, tenant detail, tenant create, metrics
- **Styling**: Tailwind CSS with custom component classes
- **State Management**: SWR for data fetching and caching
- **Charts**: Recharts for metrics visualization
- **Icons**: Lucide React

## API Integration

The UI proxies all requests to the BFF:

- `GET /api/v1/tenants` - List all tenants
- `POST /api/v1/tenants` - Create new tenant
- `GET /api/v1/tenants/:name` - Get tenant details
- `GET /api/v1/tenants/:name/metrics` - Get metrics
- `GET /api/v1/tenants/:name/kubeconfig` - Export kubeconfig (Gold tier only)
- `PATCH /api/v1/tenants/:name` - Update tenant
- `DELETE /api/v1/tenants/:name` - Delete tenant

## UI Components

- **Sidebar**: Navigation with branding and links
- **Header**: Page title and action buttons
- **TenantCard**: Compact tenant view with status and tier badges
- **StateBadge**: Status indicator with icons (Provisioning, Ready, Failed, Suspended)
- **TierBadge**: Colored badge for isolation tier
- **Charts**: Line charts for metrics trends

## Styling Classes

```css
.btn-primary      /* Blue CTA button */
.btn-secondary    /* Secondary action button */
.btn-danger       /* Red destructive action */
.card             /* White card with shadow and border */
.tier-bronze      /* Amber badge */
.tier-silver      /* Slate badge */
.tier-gold        /* Yellow badge */
.state-ready      /* Green state */
.state-provisioning  /* Yellow state with spinner */
```

## Next Steps(Not Very Sure,If we should do it)

- Add OIDC/OAuth integration for authentication
- Implement real-time updates with WebSockets
- Add tenant edit/update forms
- Integrate with Prometheus for live metrics
- Add audit logging and role-based access control
- Implement tenant search and filtering
- Add multi-language support

