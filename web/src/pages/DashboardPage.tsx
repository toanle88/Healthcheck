import React from 'react';
import Header from '../components/layout/Header';
import Footer from '../components/layout/Footer';
import HealthCard from '../components/dashboard/HealthCard';
import TargetsHeader from '../components/dashboard/TargetsHeader';
import LoadingSpinner from '../components/common/LoadingSpinner';
import ErrorDisplay from '../components/common/ErrorDisplay';
import { useHealthQuery } from '../hooks/useHealthQuery';

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
            <TargetsHeader count={data?.count || 0} />

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
