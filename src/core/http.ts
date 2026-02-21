export interface RetryOptions {
  retries?: number;
  delayMs?: number;
}

/**
 * Source: New helper for CLI reliability.
 */
export async function retryOperation<T>(
  operation: () => Promise<T>,
  options: RetryOptions = {},
): Promise<T> {
  const retries = options.retries ?? 3;
  const delayMs = options.delayMs ?? 300;
  let lastError: unknown;

  for (let attempt = 0; attempt <= retries; attempt += 1) {
    try {
      return await operation();
    } catch (error) {
      lastError = error;
      if (attempt >= retries) {
        break;
      }
      await sleep(delayMs * (attempt + 1));
    }
  }
  throw lastError;
}

/**
 * Source: New helper for CLI reliability.
 */
export async function fetchWithRetry(
  input: string | URL,
  init: RequestInit | undefined = undefined,
  options: RetryOptions = {},
): Promise<Response> {
  return retryOperation(async () => {
    return fetch(input, init);
  }, options);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
