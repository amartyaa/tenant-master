import '@/styles/globals.css'

export const metadata = {
  title: 'Tenant Master - Management Console',
  description: 'Enterprise multi-tenant platform management',
}

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body className="bg-slate-50 text-slate-900">
        {children}
      </body>
    </html>
  )
}
