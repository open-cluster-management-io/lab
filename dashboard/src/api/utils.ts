/**
 * Create headers for API requests
 * @returns Headers object with Authorization and Content-Type
 */
export const createHeaders = (): HeadersInit => {
  const token = localStorage.getItem('authToken');
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };

  if (token) {
    // Token is already stored with 'Bearer ' prefix
    headers['Authorization'] = token;
  }

  return headers;
};
