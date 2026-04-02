import { useMsal } from "@azure/msal-react";
import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Polls a data-fetching function on mount and every `intervalMs` milliseconds.
 * Returns `{ data, loading, error, refetch }`.
 */
export function usePolling<T>(
  fetcher: () => Promise<T>,
  intervalMs = 30_000
): { data: T | null; loading: boolean; error: string | null; refetch: () => void } {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetch_ = useCallback(async () => {
    try {
      const result = await fetcher();
      setData(result);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [fetcher]);

  useEffect(() => {
    fetch_();
    timerRef.current = setInterval(fetch_, intervalMs);
    return () => {
      if (timerRef.current !== null) clearInterval(timerRef.current);
    };
  }, [fetch_, intervalMs]);

  return { data, loading, error, refetch: fetch_ };
}

/** Convenience hook that provides the MSAL instance and the first active account. */
export function useAuth() {
  const { instance, accounts } = useMsal();
  const account = accounts[0] ?? null;
  return { instance, account };
}
