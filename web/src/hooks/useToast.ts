import { useContext } from 'react';
import { ToastContext } from '../components/common/ToastContext';

/**
 * Returns the toast helper object from ToastContext.
 * Must be used inside a <ToastProvider>.
 *
 * Kept in a dedicated file so ToastContext.tsx only exports
 * React components, satisfying the react-refresh/only-export-components rule.
 */
export const useToast = () => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context.toast;
};
