import React, { useState, useEffect } from 'react';
import Header from '../components/layout/Header';
import Footer from '../components/layout/Footer';
import HealthCard from '../components/dashboard/HealthCard';
import TargetsHeader from '../components/dashboard/TargetsHeader';
import LoadingSpinner from '../components/common/LoadingSpinner';
import ErrorDisplay from '../components/common/ErrorDisplay';
import IncidentLogModal from '../components/dashboard/IncidentLogModal';
import { useHealthQuery } from '../hooks/useHealthQuery';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../components/common/ToastContext';
import { healthService } from '../services/healthService';
import { Settings, Plus, Trash2, Globe, Sparkles } from 'lucide-react';
import { getEnv } from '../config/env';

interface DashboardPageProps {
  theme?: 'light' | 'dark';
  toggleTheme?: () => void;
}

const DashboardPage: React.FC<DashboardPageProps> = ({ theme, toggleTheme }) => {
  const { 
    data, 
    isLoading, 
    isError, 
    error, 
    dataUpdatedAt, 
    isFetching, 
    refetch 
  } = useHealthQuery();

  const { isAuthenticated, getAccessToken, isAdmin } = useAuth();
  const toast = useToast();
  const queryClient = useQueryClient();

  const [showManage, setShowManage] = useState(false);
  const [newName, setNewName] = useState('');
  const [newUrl, setNewUrl] = useState('');
  const [method, setMethod] = useState('GET');
  const [headers, setHeaders] = useState('');
  const [expectedStatus, setExpectedStatus] = useState<number>(200);
  const [responseContains, setResponseContains] = useState('');
  const [failureThreshold, setFailureThreshold] = useState<number>(3);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [selectedTarget, setSelectedTarget] = useState<{ target: string; name: string } | null>(null);

  // Query to fetch all targets
  const { data: targetsList } = useQuery({
    queryKey: ['targets'],
    queryFn: () => healthService.getTargets(),
    enabled: isAuthenticated && showManage && isAdmin,
  });

  // Mutation to add a target
  const addTargetMutation = useMutation({ 
    mutationFn: ({ 
      name, 
      url, 
      method, 
      headers, 
      expectedStatus, 
      responseContains, 
      failureThreshold 
    }: { 
      name: string; 
      url: string; 
      method: string; 
      headers?: string; 
      expectedStatus: number; 
      responseContains?: string; 
      failureThreshold: number; 
    }) => healthService.createTarget(name, url, method, headers || undefined, expectedStatus, responseContains || undefined, failureThreshold), 
    onSuccess: () => { 
      queryClient.invalidateQueries({ queryKey: ['targets'] }); 
      queryClient.invalidateQueries({ queryKey: ['healthStatus'] }); 
      setNewName(''); 
      setNewUrl(''); 
      setMethod('GET'); 
      setHeaders(''); 
      setExpectedStatus(200); 
      setResponseContains(''); 
      setFailureThreshold(3); 
      setErrorMsg(null); 
      toast.success('Target added successfully');
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } }; message?: string };
      const msg = error?.response?.data?.error || error.message || 'Failed to add target';
      setErrorMsg(msg);
      toast.error(msg);
    }
  });

  // Mutation to delete a target
  const deleteTargetMutation = useMutation({
    mutationFn: (id: number) => healthService.deleteTarget(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['targets'] });
      queryClient.invalidateQueries({ queryKey: ['healthStatus'] });
      toast.success('Target deleted successfully');
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } }; message?: string };
      toast.error(error?.response?.data?.error || error.message || 'Failed to delete target');
    }
  });

  // Establish real-time SSE stream connection
  useEffect(() => {
    let eventSource: EventSource | null = null;
    let isCancelled = false;

    const connectSSE = async () => {
      try {
        const token = await getAccessToken();
        if (isCancelled) return;

        const url = `${getEnv('VITE_API_URL')}/api/status/stream?token=${encodeURIComponent(token)}`;
        eventSource = new EventSource(url);

        eventSource.onmessage = (event) => {
          try {
            const update = JSON.parse(event.data);
            queryClient.setQueryData(['healthStatus'], update);
          } catch (e) {
            console.error('Failed to parse SSE update:', e);
          }
        };

        eventSource.onerror = (err) => {
          console.error('SSE connection error:', err);
        };
      } catch (err) {
        console.error('SSE authentication/setup failed:', err);
      }
    };

    if (isAuthenticated) {
      connectSSE();
    }

    return () => {
      isCancelled = true;
      if (eventSource) {
        eventSource.close();
      }
    };
  }, [isAuthenticated, queryClient, getAccessToken]);

  return (
    <div className="min-h-screen bg-bg-base text-text-primary font-sans selection:bg-indigo-500/30 transition-colors duration-300">
      <Header 
        error={isError} 
        lastUpdated={new Date(dataUpdatedAt)} 
        isRefreshing={isFetching} 
        onRefresh={() => refetch()} 
        theme={theme}
        onToggleTheme={toggleTheme}
      />

      <main className="max-w-6xl mx-auto px-6 py-12">
        {isLoading ? (
          <LoadingSpinner />
        ) : isError ? (
          <ErrorDisplay error={error instanceof Error ? error.message : 'An error occurred'} onRetry={() => refetch()} />
        ) : (
          <div className="space-y-8">
            <div className="flex items-center justify-between">
              <TargetsHeader count={data?.count || 0} />
              {isAdmin && (
                <button
                  onClick={() => setShowManage(!showManage)}
                  className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold transition-all duration-300 active:scale-95 border cursor-pointer ${
                    showManage 
                      ? 'bg-indigo-500/10 text-indigo-400 border-indigo-500/30' 
                      : 'bg-bg-card/50 text-text-secondary border-border-primary hover:border-indigo-500/50 hover:text-text-primary'
                  }`}
                >
                  <Settings className={`w-4 h-4 ${showManage ? 'rotate-45' : ''} transition-transform duration-500`} />
                  <span>{showManage ? 'Hide Settings' : 'Manage Targets'}</span>
                </button>
              )}
            </div>

            {showManage && isAdmin && (
              <div className="bg-bg-card/40 border border-border-primary rounded-3xl p-6 backdrop-blur-xl animate-in fade-in slide-in-from-top-4 duration-300 space-y-6">
                <div className="flex items-center gap-2 pb-4 border-b border-border-primary/50">
                  <Sparkles className="w-5 h-5 text-indigo-400" />
                  <h2 className="text-lg font-bold text-text-primary">Configure Targets</h2>
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                  {/* Form to Add Target */}
                  <form onSubmit={(e) => {
                    e.preventDefault();
                    if (!newName || !newUrl) return;

                    if (headers.trim() !== '') {
                      try {
                        JSON.parse(headers);
                      } catch {
                        const msg = 'Custom Headers must be valid JSON (e.g. {"Key": "Value"})';
                        setErrorMsg(msg);
                        toast.error(msg);
                        return;
                      }
                    }

                    addTargetMutation.mutate({ 
                      name: newName, 
                      url: newUrl, 
                      method, 
                      headers: headers.trim() !== '' ? headers : undefined, 
                      expectedStatus: Number(expectedStatus), 
                      responseContains: responseContains.trim() !== '' ? responseContains : undefined, 
                      failureThreshold: Number(failureThreshold) 
                    });
                  }} className="space-y-4">
                    <h3 className="text-xs font-bold text-text-secondary uppercase tracking-wider">Add New Endpoint</h3>
                    
                    <div className="space-y-3">
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="block text-[10px] font-semibold text-text-secondary/80 mb-1" htmlFor="target-name">Service Name</label>
                          <input
                            id="target-name"
                            type="text"
                            placeholder="e.g. Google Search"
                            value={newName}
                            onChange={(e) => setNewName(e.target.value)}
                            className="w-full bg-bg-base/60 border border-border-primary rounded-xl px-4 py-2 text-sm text-text-primary focus:outline-none focus:border-indigo-500 transition-colors"
                            required
                          />
                        </div>
                        <div>
                          <label className="block text-[10px] font-semibold text-text-secondary/80 mb-1" htmlFor="target-method">HTTP Method</label>
                          <select
                            id="target-method"
                            value={method}
                            onChange={(e) => setMethod(e.target.value)}
                            className="w-full bg-bg-base/60 border border-border-primary rounded-xl px-4 py-2 text-sm text-text-primary focus:outline-none focus:border-indigo-500 transition-colors"
                          >
                            <option value="GET">GET</option>
                            <option value="POST">POST</option>
                            <option value="PUT">PUT</option>
                            <option value="DELETE">DELETE</option>
                            <option value="HEAD">HEAD</option>
                            <option value="PATCH">PATCH</option>
                            <option value="OPTIONS">OPTIONS</option>
                          </select>
                        </div>
                      </div>
                      <div>
                        <label className="block text-[10px] font-semibold text-text-secondary/80 mb-1" htmlFor="target-url">HTTP(S) Endpoint URL</label>
                        <input
                          id="target-url"
                          type="url"
                          placeholder="https://example.com"
                          value={newUrl}
                          onChange={(e) => setNewUrl(e.target.value)}
                          className="w-full bg-bg-base/60 border border-border-primary rounded-xl px-4 py-2 text-sm text-text-primary focus:outline-none focus:border-indigo-500 transition-colors font-mono"
                          required
                        />
                      </div>
                      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                        <div>
                          <label className="block text-[10px] font-semibold text-text-secondary/80 mb-1" htmlFor="target-status">Expected Status</label>
                          <input
                            id="target-status"
                            type="number"
                            placeholder="200"
                            value={expectedStatus}
                            onChange={(e) => setExpectedStatus(Number(e.target.value))}
                            className="w-full bg-bg-base/60 border border-border-primary rounded-xl px-4 py-2 text-sm text-text-primary focus:outline-none focus:border-indigo-500 transition-colors"
                            min="100"
                            max="599"
                            required
                          />
                        </div>
                        <div>
                          <label className="block text-[10px] font-semibold text-text-secondary/80 mb-1" htmlFor="target-threshold">Failure Threshold</label>
                          <input
                            id="target-threshold"
                            type="number"
                            placeholder="3"
                            value={failureThreshold}
                            onChange={(e) => setFailureThreshold(Number(e.target.value))}
                            className="w-full bg-bg-base/60 border border-border-primary rounded-xl px-4 py-2 text-sm text-text-primary focus:outline-none focus:border-indigo-500 transition-colors"
                            min="1"
                            max="10"
                            required
                          />
                        </div>
                        <div>
                          <label className="block text-[10px] font-semibold text-text-secondary/80 mb-1" htmlFor="target-contains">Response Body Match</label>
                          <input
                            id="target-contains"
                            type="text"
                            placeholder="e.g. success"
                            value={responseContains}
                            onChange={(e) => setResponseContains(e.target.value)}
                            className="w-full bg-bg-base/60 border border-border-primary rounded-xl px-4 py-2 text-sm text-text-primary focus:outline-none focus:border-indigo-500 transition-colors"
                          />
                        </div>
                      </div>
                      <div>
                        <label className="block text-[10px] font-semibold text-text-secondary/80 mb-1" htmlFor="target-headers">Custom Headers (JSON)</label>
                        <textarea
                          id="target-headers"
                          placeholder='e.g. {"Authorization": "Bearer token", "X-Custom": "Value"}'
                          value={headers}
                          onChange={(e) => setHeaders(e.target.value)}
                          className="w-full bg-bg-base/60 border border-border-primary rounded-xl px-4 py-2 text-sm text-text-primary focus:outline-none focus:border-indigo-500 transition-colors font-mono h-16 resize-none"
                        />
                      </div>
                    </div>

                    {errorMsg && <p className="text-xs text-red-400 font-medium">{errorMsg}</p>}

                    <button
                      type="submit"
                      disabled={addTargetMutation.isPending}
                      className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-800 text-white text-sm font-semibold py-2.5 rounded-xl transition-colors flex items-center justify-center gap-2 active:scale-[0.99] cursor-pointer"
                    >
                      <Plus className="w-4 h-4" />
                      <span>{addTargetMutation.isPending ? 'Adding target...' : 'Add Target'}</span>
                    </button>
                  </form>

                  {/* List of current targets */}
                  <div className="space-y-4">
                    <h3 className="text-xs font-bold text-text-secondary uppercase tracking-wider">Current Targets</h3>
                    <div className="bg-bg-base/40 border border-border-primary rounded-2xl overflow-hidden max-h-[220px] overflow-y-auto">
                      {targetsList && targetsList.length > 0 ? (
                        <div className="divide-y divide-border-primary/50">
                          {targetsList.map((target) => (
                            <div key={target.id} className="flex items-center justify-between p-3.5 hover:bg-bg-base/50 transition-colors">
                              <div className="flex items-center gap-3 truncate pr-4">
                                <div className="p-1.5 bg-indigo-500/10 rounded-lg text-indigo-400 flex-shrink-0">
                                  <Globe className="w-4 h-4" />
                                </div>
                                <div className="truncate">
                                  <div className="flex items-center gap-2">
                                    <h4 className="text-sm font-semibold text-text-primary">{target.name}</h4>
                                    <span className="text-[9px] font-bold px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">{target.method || 'GET'}</span>
                                    <span className="text-[9px] font-bold px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">{target.expected_status || 200}</span>
                                    <span className="text-[9px] font-bold px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20" title="Failure threshold">Threshold: {target.failure_threshold || 3}</span>
                                  </div>
                                  <p className="text-xs text-text-secondary/70 font-mono truncate">{target.url}</p>
                                </div>
                              </div>
                              <button
                                onClick={() => deleteTargetMutation.mutate(target.id)}
                                disabled={deleteTargetMutation.isPending}
                                className="p-2 hover:bg-red-500/10 text-text-secondary hover:text-red-400 rounded-lg transition-colors flex-shrink-0 cursor-pointer"
                                title="Delete Target"
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div className="text-center py-10">
                          <p className="text-xs text-text-secondary">No targets configured. Please add one.</p>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {data?.checks.map((check, index) => (
                <HealthCard 
                  key={`${check.target}-${index}`} 
                  check={check} 
                  onClick={() => setSelectedTarget({ target: check.target, name: check.name })}
                />
              ))}
            </div>
            
            {data?.checks?.length === 0 && (
              <div className="text-center py-20 border-2 border-dashed border-border-primary rounded-3xl">
                <p className="text-text-secondary font-medium">No data received yet. The worker might still be initializing.</p>
              </div>
            )}
          </div>
        )}
      </main>
      
      <Footer count={data?.count || 0} />

      {selectedTarget && (
        <IncidentLogModal 
          target={selectedTarget.target}
          name={selectedTarget.name}
          onClose={() => setSelectedTarget(null)}
        />
      )}
    </div>
  );
};

export default DashboardPage;
