import React, { useState } from 'react';
import Header from '../components/layout/Header';
import Footer from '../components/layout/Footer';
import HealthCard from '../components/dashboard/HealthCard';
import TargetsHeader from '../components/dashboard/TargetsHeader';
import LoadingSpinner from '../components/common/LoadingSpinner';
import ErrorDisplay from '../components/common/ErrorDisplay';
import { useHealthQuery } from '../hooks/useHealthQuery';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '../hooks/useAuth';
import { healthService } from '../services/healthService';
import { Settings, Plus, Trash2, Globe, Sparkles } from 'lucide-react';

const DashboardPage: React.FC = () => {
  const { 
    data, 
    isLoading, 
    isError, 
    error, 
    dataUpdatedAt, 
    isFetching, 
    refetch 
  } = useHealthQuery();

  const { isAuthenticated } = useAuth();
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

  // Query to fetch all targets
  const { data: targetsList } = useQuery({
    queryKey: ['targets'],
    queryFn: () => healthService.getTargets(),
    enabled: isAuthenticated && showManage,
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
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } }; message?: string };
      setErrorMsg(error?.response?.data?.error || error.message || 'Failed to add target');
    }
  });

  // Mutation to delete a target
  const deleteTargetMutation = useMutation({
    mutationFn: (id: number) => healthService.deleteTarget(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['targets'] });
      queryClient.invalidateQueries({ queryKey: ['healthStatus'] });
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { error?: string } }; message?: string };
      alert(error?.response?.data?.error || error.message || 'Failed to delete target');
    }
  });

  return (
    <div className="min-h-screen bg-[#0f172a] text-slate-100 font-sans selection:bg-indigo-500/30">
      <Header 
        error={isError} 
        lastUpdated={new Date(dataUpdatedAt)} 
        isRefreshing={isFetching} 
        onRefresh={() => refetch()} 
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
              <button
                onClick={() => setShowManage(!showManage)}
                className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold transition-all duration-300 active:scale-95 border ${
                  showManage 
                    ? 'bg-indigo-500/10 text-indigo-400 border-indigo-500/30' 
                    : 'bg-slate-900/50 text-slate-400 border-slate-800 hover:border-slate-700 hover:text-slate-200'
                }`}
              >
                <Settings className={`w-4 h-4 ${showManage ? 'rotate-45' : ''} transition-transform duration-500`} />
                <span>{showManage ? 'Hide Settings' : 'Manage Targets'}</span>
              </button>
            </div>

            {showManage && (
              <div className="bg-slate-900/40 border border-slate-800/80 rounded-3xl p-6 backdrop-blur-xl animate-in fade-in slide-in-from-top-4 duration-300 space-y-6">
                <div className="flex items-center gap-2 pb-4 border-b border-slate-850/50">
                  <Sparkles className="w-5 h-5 text-indigo-400" />
                  <h2 className="text-lg font-bold text-slate-200">Configure Targets</h2>
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
                        setErrorMsg('Custom Headers must be valid JSON (e.g. {"Key": "Value"})');
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
                    <h3 className="text-xs font-bold text-slate-400 uppercase tracking-wider">Add New Endpoint</h3>
                    
                    <div className="space-y-3">
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="block text-[10px] font-semibold text-slate-500 mb-1" htmlFor="target-name">Service Name</label>
                          <input
                            id="target-name"
                            type="text"
                            placeholder="e.g. Google Search"
                            value={newName}
                            onChange={(e) => setNewName(e.target.value)}
                            className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors"
                            required
                          />
                        </div>
                        <div>
                          <label className="block text-[10px] font-semibold text-slate-500 mb-1" htmlFor="target-method">HTTP Method</label>
                          <select
                            id="target-method"
                            value={method}
                            onChange={(e) => setMethod(e.target.value)}
                            className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors"
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
                        <label className="block text-[10px] font-semibold text-slate-500 mb-1" htmlFor="target-url">HTTP(S) Endpoint URL</label>
                        <input
                          id="target-url"
                          type="url"
                          placeholder="https://example.com"
                          value={newUrl}
                          onChange={(e) => setNewUrl(e.target.value)}
                          className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors font-mono"
                          required
                        />
                      </div>
                      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                        <div>
                          <label className="block text-[10px] font-semibold text-slate-500 mb-1" htmlFor="target-status">Expected Status</label>
                          <input
                            id="target-status"
                            type="number"
                            placeholder="200"
                            value={expectedStatus}
                            onChange={(e) => setExpectedStatus(Number(e.target.value))}
                            className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors"
                            min="100"
                            max="599"
                            required
                          />
                        </div>
                        <div>
                          <label className="block text-[10px] font-semibold text-slate-500 mb-1" htmlFor="target-threshold">Failure Threshold</label>
                          <input
                            id="target-threshold"
                            type="number"
                            placeholder="3"
                            value={failureThreshold}
                            onChange={(e) => setFailureThreshold(Number(e.target.value))}
                            className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors"
                            min="1"
                            max="10"
                            required
                          />
                        </div>
                        <div>
                          <label className="block text-[10px] font-semibold text-slate-500 mb-1" htmlFor="target-contains">Response Body Match</label>
                          <input
                            id="target-contains"
                            type="text"
                            placeholder="e.g. success"
                            value={responseContains}
                            onChange={(e) => setResponseContains(e.target.value)}
                            className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors"
                          />
                        </div>
                      </div>
                      <div>
                        <label className="block text-[10px] font-semibold text-slate-500 mb-1" htmlFor="target-headers">Custom Headers (JSON)</label>
                        <textarea
                          id="target-headers"
                          placeholder='e.g. {"Authorization": "Bearer token", "X-Custom": "Value"}'
                          value={headers}
                          onChange={(e) => setHeaders(e.target.value)}
                          className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-2 text-sm text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors font-mono h-16 resize-none"
                        />
                      </div>
                    </div>

                    {errorMsg && <p className="text-xs text-red-400 font-medium">{errorMsg}</p>}

                    <button
                      type="submit"
                      disabled={addTargetMutation.isPending}
                      className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:bg-indigo-800 text-white text-sm font-semibold py-2.5 rounded-xl transition-colors flex items-center justify-center gap-2 active:scale-[0.99]"
                    >
                      <Plus className="w-4 h-4" />
                      <span>{addTargetMutation.isPending ? 'Adding target...' : 'Add Target'}</span>
                    </button>
                  </form>

                  {/* List of current targets */}
                  <div className="space-y-4">
                    <h3 className="text-xs font-bold text-slate-400 uppercase tracking-wider">Current Targets</h3>
                    <div className="bg-slate-950/40 border border-slate-800 rounded-2xl overflow-hidden max-h-[220px] overflow-y-auto">
                      {targetsList && targetsList.length > 0 ? (
                        <div className="divide-y divide-slate-850/50">
                          {targetsList.map((target) => (
                            <div key={target.id} className="flex items-center justify-between p-3.5 hover:bg-slate-900/30 transition-colors">
                              <div className="flex items-center gap-3 truncate pr-4">
                                <div className="p-1.5 bg-indigo-500/10 rounded-lg text-indigo-400 flex-shrink-0">
                                  <Globe className="w-4 h-4" />
                                </div>
                                <div className="truncate">
                                  <div className="flex items-center gap-2">
                                    <h4 className="text-sm font-semibold text-slate-200">{target.name}</h4>
                                    <span className="text-[9px] font-bold px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">{target.method || 'GET'}</span>
                                    <span className="text-[9px] font-bold px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">{target.expected_status || 200}</span>
                                    <span className="text-[9px] font-bold px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-400 border border-amber-500/20" title="Failure threshold">Threshold: {target.failure_threshold || 3}</span>
                                  </div>
                                  <p className="text-xs text-slate-500 font-mono truncate">{target.url}</p>
                                </div>
                              </div>
                              <button
                                onClick={() => deleteTargetMutation.mutate(target.id)}
                                disabled={deleteTargetMutation.isPending}
                                className="p-2 hover:bg-red-500/10 text-slate-500 hover:text-red-400 rounded-lg transition-colors flex-shrink-0"
                                title="Delete Target"
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div className="text-center py-10">
                          <p className="text-xs text-slate-500">No targets configured. Please add one.</p>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {data?.checks.map((check, index) => (
                <HealthCard key={`${check.target}-${index}`} check={check} />
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
      
      <Footer count={data?.count || 0} />
    </div>
  );
};

export default DashboardPage;
