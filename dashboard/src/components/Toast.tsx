import { useState, useEffect, useCallback, createContext, useContext } from 'react';

// ── Toast System ────────────────────────────────────────────────────────

interface ToastItem {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
  exiting: boolean;
}

interface ToastContextValue {
  showToast: (message: string, type?: 'success' | 'error' | 'info') => void;
}

const ToastContext = createContext<ToastContextValue>({ showToast: () => {} });

export function useToast() {
  return useContext(ToastContext);
}

let toastIdCounter = 0;

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const showToast = useCallback(
    (message: string, type: 'success' | 'error' | 'info' = 'info') => {
      const id = ++toastIdCounter;
      setToasts((prev) => [...prev, { id, message, type, exiting: false }]);
    },
    []
  );

  const removeToast = useCallback((id: number) => {
    setToasts((prev) =>
      prev.map((t) => (t.id === id ? { ...t, exiting: true } : t))
    );
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 300);
  }, []);

  useEffect(() => {
    if (toasts.length === 0) return;
    const latest = toasts[toasts.length - 1];
    if (latest.exiting) return;
    const timer = setTimeout(() => removeToast(latest.id), 3500);
    return () => clearTimeout(timer);
  }, [toasts, removeToast]);

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      <div
        style={{
          position: 'fixed',
          bottom: '24px',
          right: '24px',
          display: 'flex',
          flexDirection: 'column',
          gap: '8px',
          zIndex: 9999,
          pointerEvents: 'none',
        }}
      >
        {toasts.map((toast) => (
          <div
            key={toast.id}
            onClick={() => removeToast(toast.id)}
            style={{
              pointerEvents: 'auto',
              cursor: 'pointer',
              padding: '12px 20px',
              borderRadius: 'var(--radius-md)',
              fontFamily: 'var(--font-sans)',
              fontSize: '13px',
              fontWeight: 500,
              lineHeight: 1.4,
              maxWidth: '380px',
              border: '1px solid',
              animation: toast.exiting
                ? 'toastOut 0.3s ease forwards'
                : 'toastIn 0.3s ease forwards',
              ...(toast.type === 'success'
                ? {
                    background: 'var(--color-status-ok-bg)',
                    color: 'var(--color-status-ok-text)',
                    borderColor: 'var(--color-status-ok)',
                  }
                : toast.type === 'error'
                ? {
                    background: 'var(--color-status-danger-bg)',
                    color: 'var(--color-status-danger-text)',
                    borderColor: 'var(--color-status-danger)',
                  }
                : {
                    background: 'var(--color-bg-elevated)',
                    color: 'var(--color-text-primary)',
                    borderColor: 'var(--color-border-hover)',
                  }),
            }}
          >
            {toast.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
