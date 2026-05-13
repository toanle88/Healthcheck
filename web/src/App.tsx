import { useEffect, useState, useCallback } from 'react'
import { Activity, ShieldCheck, ShieldAlert, Clock, RefreshCw, Server } from 'lucide-react'

interface Check {
  target: string
  status: string
  latency_ms: number
  checked_at: string
}

interface ApiResponse {
  checks: Check[]
  count: number
}

function App() {
  const [data, setData] = useState<ApiResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date>(new Date())
  const [isRefreshing, setIsRefreshing] = useState(false)

  // Use the environment variable if present, otherwise fall back to the placeholder for Azure injection.
  // Locally, this will use the value from your .env file or localhost.
  const API_BASE_URL = import.meta.env.VITE_API_URL || 'VITE_API_URL_PLACEHOLDER';

  const fetchData = useCallback(async () => {
    setIsRefreshing(true)
    try {
      const response = await fetch(`${API_BASE_URL}/api/status`)
      if (!response.ok) {
        throw new Error('Failed to fetch status')
      }
      const result = await response.json()
      setData(result)
      setLastUpdated(new Date())
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    } finally {
      setLoading(false)
      setTimeout(() => setIsRefreshing(false), 500)
    }
  }, [])

  useEffect(() => {
    // Defer the initial fetch to avoid the "set-state-in-effect" lint warning.
    // This ensures the state update happens after the initial render cycle.
    const timeoutId = setTimeout(() => {
      fetchData()
    }, 0)

    const interval = setInterval(fetchData, 10000) // Refresh every 10 seconds

    return () => {
      clearTimeout(timeoutId)
      clearInterval(interval)
    }
  }, [fetchData])

  return (
    <div className="min-h-screen bg-[#0f172a] text-slate-100 font-sans selection:bg-indigo-500/30">
      {/* Header */}
      <header className="border-b border-slate-800 bg-slate-900/50 backdrop-blur-xl sticky top-0 z-50">
        <div className="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-indigo-500/10 rounded-lg">
              <Activity className="w-6 h-6 text-indigo-400 animate-pulse" />
            </div>
            <div>
              <h1 className="text-xl font-bold tracking-tight">Healthcheck <span className="text-indigo-400">Dashboard</span></h1>
              <p className="text-xs text-slate-400 font-medium flex items-center gap-1.5">
                <div className={`w-1.5 h-1.5 rounded-full ${error ? 'bg-red-500' : 'bg-emerald-500 animate-pulse'}`} />
                System {error ? 'Degraded' : 'Operational'}
              </p>
            </div>
          </div>
          
          <div className="flex items-center gap-4 text-sm font-medium">
            <div className="flex items-center gap-2 text-slate-400 bg-slate-800/50 px-3 py-1.5 rounded-full border border-slate-700/50">
              <Clock className="w-4 h-4" />
              <span>{lastUpdated.toLocaleTimeString()}</span>
            </div>
            <button 
              onClick={fetchData}
              disabled={isRefreshing}
              className="p-2 hover:bg-slate-800 rounded-lg transition-colors group active:scale-95 disabled:opacity-50"
            >
              <RefreshCw className={`w-5 h-5 text-slate-400 group-hover:text-indigo-400 transition-colors ${isRefreshing ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 py-12">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-32 gap-4">
            <div className="w-12 h-12 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin" />
            <p className="text-slate-400 animate-pulse font-medium">Initializing monitoring hooks...</p>
          </div>
        ) : error ? (
          <div className="bg-red-500/10 border border-red-500/20 rounded-2xl p-8 text-center max-w-lg mx-auto">
            <ShieldAlert className="w-12 h-12 text-red-400 mx-auto mb-4" />
            <h2 className="text-xl font-bold text-red-100 mb-2">Connection Error</h2>
            <p className="text-red-400/80 mb-6">{error}</p>
            <button 
              onClick={fetchData}
              className="bg-red-500 hover:bg-red-600 text-white px-6 py-2 rounded-xl font-bold transition-all active:scale-95 shadow-lg shadow-red-500/20"
            >
              Retry Connection
            </button>
          </div>
        ) : (
          <div className="space-y-8">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                <Server className="w-5 h-5 text-indigo-400" />
                Active Targets
                <span className="ml-2 px-2 py-0.5 bg-slate-800 rounded text-xs text-slate-400">{data?.count || 0}</span>
              </h2>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {data?.checks.map((check, index) => (
                <div 
                  key={`${check.target}-${index}`}
                  className="group bg-slate-900/50 border border-slate-800 rounded-2xl p-6 hover:border-slate-700 hover:bg-slate-800/50 transition-all duration-300 hover:shadow-2xl hover:shadow-indigo-500/5"
                >
                  <div className="flex items-start justify-between mb-4">
                    <div className={`p-2.5 rounded-xl ${check.status === 'up' ? 'bg-emerald-500/10' : 'bg-red-500/10'}`}>
                      {check.status === 'up' ? (
                        <ShieldCheck className="w-6 h-6 text-emerald-400" />
                      ) : (
                        <ShieldAlert className="w-6 h-6 text-red-400" />
                      )}
                    </div>
                    <div className="flex flex-col items-end">
                      <span className={`text-xs font-bold uppercase tracking-wider px-2 py-1 rounded-md ${
                        check.status === 'up' ? 'text-emerald-400 bg-emerald-400/10' : 'text-red-400 bg-red-400/10'
                      }`}>
                        {check.status}
                      </span>
                      <span className="text-[10px] text-slate-500 mt-2 font-mono">{new Date(check.checked_at).toLocaleTimeString()}</span>
                    </div>
                  </div>
                  
                  <div className="space-y-1 mb-6">
                    <h3 className="font-bold text-slate-200 truncate group-hover:text-white transition-colors" title={check.target}>
                      {check.target.replace(/^https?:\/\//, '')}
                    </h3>
                    <p className="text-xs text-slate-500 font-mono truncate">{check.target}</p>
                  </div>

                  <div className="flex items-center justify-between pt-4 border-t border-slate-800/50">
                    <div className="flex items-center gap-2 text-slate-400">
                      <Clock className="w-3.5 h-3.5" />
                      <span className="text-xs font-medium">Latency</span>
                    </div>
                    <span className={`text-sm font-bold font-mono ${
                      check.latency_ms < 200 ? 'text-emerald-400' : check.latency_ms < 500 ? 'text-amber-400' : 'text-red-400'
                    }`}>
                      {check.latency_ms}ms
                    </span>
                  </div>
                </div>
              ))}
            </div>
            
            {data?.checks?.length === 0 && (
              <div className="text-center py-20 border-2 border-dashed border-slate-800 rounded-3xl">
                <p className="text-slate-500 font-medium">No data received yet. The worker might still be initializing.</p>
              </div>
            )}
          </div>
        )}
      </main>
      
      <footer className="mt-auto border-t border-slate-800 py-8 px-6">
        <div className="max-w-6xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
          <p className="text-sm text-slate-500 font-medium">
            Monitoring {data?.count || 0} endpoints across global infrastructure
          </p>
          <div className="flex items-center gap-3">
            <span className="text-[10px] font-bold uppercase tracking-widest text-slate-600">Version</span>
            <span className="px-2 py-0.5 bg-slate-800 text-slate-400 rounded text-[10px] font-mono border border-slate-700/50">
              {import.meta.env.VITE_APP_VERSION || 'VITE_APP_VERSION_PLACEHOLDER'}
            </span>
          </div>
        </div>
      </footer>
    </div>
  )
}

export default App
